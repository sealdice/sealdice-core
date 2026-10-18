//nolint:testpackage
package dice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"github.com/gorilla/websocket"
)

type milkyRecoveryFixture struct {
	pa         *PlatformAdapterMilky
	session    *milky.Session
	apiHealthy atomic.Bool
	loggedIn   atomic.Bool
	apiStatus  atomic.Int32
	apiCalls   atomic.Int32
	uin        atomic.Int64
	wsHealthy  atomic.Bool
	peers      chan *websocket.Conn
	done       chan struct{}
}

func newMilkyGatewayFixture(t *testing.T) *milkyRecoveryFixture {
	t.Helper()
	_, pa := newMilkyStartupTestAdapter(t, "")
	f := &milkyRecoveryFixture{pa: pa, peers: make(chan *websocket.Conn, 16)}
	f.apiHealthy.Store(true)
	f.loggedIn.Store(true)
	f.apiStatus.Store(http.StatusOK)
	f.uin.Store(10010)
	f.wsHealthy.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/get_login_info" {
			f.apiCalls.Add(1)
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !f.apiHealthy.Load() {
				http.Error(w, "gateway unavailable", http.StatusServiceUnavailable)
				return
			}
			if status := int(f.apiStatus.Load()); status != http.StatusOK {
				http.Error(w, "API unavailable", status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if !f.loggedIn.Load() {
				_, _ = w.Write([]byte(`{"status":"failed","retcode":-403,"message":"Bot is not logged in"}`))
				return
			}
			_, _ = fmt.Fprintf(w, `{"status":"ok","retcode":0,"data":{"uin":%d,"nickname":"MilkyBot"}}`, f.uin.Load())
			return
		}
		if !f.wsHealthy.Load() {
			http.Error(w, "event gateway unavailable", http.StatusServiceUnavailable)
			return
		}
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		f.peers <- conn
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	session, err := milky.New("ws"+strings.TrimPrefix(server.URL, "http")+"/event", server.URL+"/api", "test-token", noopMilkyLogger{})
	if err != nil {
		t.Fatal(err)
	}
	f.session = session
	f.pa.WsGateway = session.WSGateway
	f.pa.RestGateway = session.RestGateway
	f.pa.Token = session.Token
	t.Cleanup(func() {
		f.pa.stopMilkySession()
		if f.done != nil {
			select {
			case <-f.done:
			case <-time.After(3 * time.Second):
				t.Error("recovery worker did not stop")
			}
		}
		server.Close()
	})
	return f
}

func newMilkyRecoveryFixture(t *testing.T) *milkyRecoveryFixture {
	t.Helper()
	f := newMilkyGatewayFixture(t)
	session := f.session
	f.pa.startMilkySession(session, 0)
	f.pa.prepareMilkyTransport(session)
	if !session.ShouldReconnectOnError {
		t.Fatal("SDK must own automatic websocket reconnects")
	}
	if err := f.pa.refreshMilkyTransport(session, newMilkyLoginClient(100*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	assertMilkyState(t, f.pa, StateConnected)
	f.startRecovery()
	return f
}

func (f *milkyRecoveryFixture) startRecovery() {
	f.done = make(chan struct{})
	f.pa.lifecycleMu.Lock()
	ctx := f.pa.sessionContext
	f.pa.lifecycleMu.Unlock()
	go func() {
		defer close(f.done)
		f.pa.watchMilkyRecovery(ctx, f.session, 20*time.Millisecond, 100*time.Millisecond)
	}()
}

func TestMilkySDKDisconnectAndRecovery(t *testing.T) {
	f := newMilkyRecoveryFixture(t)
	f.wsHealthy.Store(false)
	_ = (<-f.peers).Close()
	assertMilkyState(t, f.pa, StateDisconnected)
	f.pa.lifecycleMu.Lock()
	enabled := f.pa.EndPoint.Enable
	f.pa.lifecycleMu.Unlock()
	if !enabled {
		t.Fatal("temporary outage disabled the account")
	}
	// REST 正常但 WS 不可用时保持离线，并由 SDK 负责重连。
	calls := f.apiCalls.Load()
	time.Sleep(1100 * time.Millisecond)
	assertMilkyState(t, f.pa, StateDisconnected)
	if got := f.apiCalls.Load(); got != calls {
		t.Fatalf("core polled login API while SDK was reconnecting: %d -> %d", calls, got)
	}
	f.wsHealthy.Store(true)
	assertMilkyState(t, f.pa, StateConnected)
}

func TestMilkySDKDoesNotPollWhileConnected(t *testing.T) {
	f := newMilkyRecoveryFixture(t)
	calls := f.apiCalls.Load()
	// 恢复间隔为 20ms；正常连接跨过多个间隔也不应再次请求登录 API。
	time.Sleep(200 * time.Millisecond)
	if got := f.apiCalls.Load(); got != calls {
		t.Fatalf("healthy session polled login API: calls changed from %d to %d", calls, got)
	}
}

func TestMilkySDKStaleConnectKeepsReconnectOwnership(t *testing.T) {
	f := newMilkyGatewayFixture(t)
	f.pa.startMilkySession(f.session, 0)
	f.pa.prepareMilkyTransport(f.session)
	// 模拟握手通知执行前，连接已断开；成功握手仍意味着 SDK 已接管重连。
	f.session.OnConnectionChange(f.session, true)
	f.pa.lifecycleMu.Lock()
	opened := f.pa.sessionOpened
	f.pa.lifecycleMu.Unlock()
	if !opened {
		t.Fatal("stale handshake notification lost SDK reconnect ownership")
	}
	if err := f.pa.openMilkyTransport(f.session); err == nil {
		t.Fatal("core competed with SDK by opening a replacement websocket")
	}
}

func TestMilkySDKReconnectWaitsForAccountRecovery(t *testing.T) {
	f := newMilkyRecoveryFixture(t)
	f.loggedIn.Store(false)
	_ = (<-f.peers).Close()
	assertMilkyState(t, f.pa, StateDisconnected)
	select {
	case <-f.peers:
	case <-time.After(5 * time.Second):
		t.Fatal("SDK did not reconnect")
	}
	assertMilkyState(t, f.pa, StateDisconnected)
	f.loggedIn.Store(true)
	// 不发送 QQ 消息，恢复流程仍应在账号上线后更新状态。
	assertMilkyState(t, f.pa, StateConnected)
}

func TestMilkySDKIgnoresReplacedSession(t *testing.T) {
	f := newMilkyRecoveryFixture(t)
	current, err := milky.New(f.session.WSGateway, f.session.RestGateway, f.session.Token, noopMilkyLogger{})
	if err != nil {
		t.Fatal(err)
	}
	f.pa.startMilkySession(current, 0)
	f.pa.prepareMilkyTransport(current)
	if err := f.pa.refreshMilkyTransport(current, newMilkyLoginClient(time.Second)); err != nil {
		t.Fatal(err)
	}
	f.session.OnConnectionChange(f.session, false)
	f.pa.onMilkyBotOffline(f.session, "old session")
	f.pa.onMilkyMessage(f.session)
	assertMilkyState(t, f.pa, StateConnected)
}

func TestMilkySDKRecoversBotOfflineAndIgnoresStoppedSessions(t *testing.T) {
	f := newMilkyRecoveryFixture(t)
	f.loggedIn.Store(false)
	f.pa.onMilkyBotOffline(f.session, "kicked offline")
	if f.pa.onMilkyMessage(f.session) {
		t.Fatal("late message cleared the offline event")
	}
	assertMilkyState(t, f.pa, StateDisconnected)
	f.loggedIn.Store(true)
	// 不发送任何 QQ 消息，成功的登录探测与 WS 连接应足以恢复状态。
	assertMilkyState(t, f.pa, StateConnected)
	f.pa.stopMilkySession()
	if err := f.session.Open(); err == nil {
		t.Fatal("stopped session could reopen")
	}
	f.pa.lifecycleMu.Lock()
	f.pa.EndPoint.State = StateDisconnected
	f.pa.EndPoint.Enable = false
	f.pa.lifecycleMu.Unlock()
	f.pa.onMilkyConnectionChange(f.session, true)
	f.pa.onMilkyAccountOnline(f.session, 0)
	f.pa.onMilkyMessage(f.session)
	assertMilkyState(t, f.pa, StateDisconnected)
	select {
	case <-f.done:
	case <-time.After(time.Second):
		t.Fatal("disabled account retained recovery worker")
	}
}

func TestMilkySDKRejectsInvalidResponses(t *testing.T) {
	for _, body := range []string{
		`{"status":"failed","retcode":0,"data":{"uin":10010}}`,
		`{"status":"ok","retcode":1,"data":{"uin":10010}}`,
		`{"status":"ok","retcode":0,"data":null}`,
		`{"status":"ok","retcode":0,"data":{"uin":0}}`,
		`not JSON`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			if _, err := probeMilkyLogin(context.Background(), server.Client(), server.URL, "", "QQ:10010"); err == nil {
				t.Fatal("invalid login response reported healthy")
			}
		})
	}
}

func TestMilkySDKRequestCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := probeMilkyLogin(ctx, server.Client(), server.URL, "", "QQ:10010")
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled probe succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not interrupt probe")
	}
}

func TestMilkyInitialTransientFailureRetries(t *testing.T) {
	for _, failure := range []string{"API unavailable", "QQ offline", "WebSocket unavailable"} {
		t.Run(failure, func(t *testing.T) {
			f := newMilkyGatewayFixture(t)
			switch failure {
			case "API unavailable":
				f.apiHealthy.Store(false)
				f.pa.EndPoint.UserID = "" // 新添加的分离连接可能还没有账号信息。
			case "QQ offline":
				f.loggedIn.Store(false)
			case "WebSocket unavailable":
				f.wsHealthy.Store(false)
			}
			f.pa.EndPoint.State = StateConnected // 模拟启动时配置中残留的状态。
			if f.pa.Serve() != 0 {
				t.Fatal("temporary startup failure did not hand off retry")
			}
			assertMilkyState(t, f.pa, StateDisconnected)
			f.pa.lifecycleMu.Lock()
			active := f.pa.sessionActive && !f.pa.sessionReady && f.pa.EndPoint.Enable
			f.pa.lifecycleMu.Unlock()
			if !active {
				t.Fatal("temporary startup failure disabled or cancelled the session")
			}
			f.apiHealthy.Store(true)
			f.loggedIn.Store(true)
			f.wsHealthy.Store(true)
			assertMilkyState(t, f.pa, StateConnected)
			f.pa.lifecycleMu.Lock()
			userID, nickname := f.pa.EndPoint.UserID, f.pa.EndPoint.Nickname
			f.pa.lifecycleMu.Unlock()
			if userID != "QQ:10010" || nickname != "MilkyBot" {
				t.Fatal("retry did not initialize account information")
			}
		})
	}
}

func TestMilkyInitialAuthenticationFailureDoesNotRetry(t *testing.T) {
	f := newMilkyGatewayFixture(t)
	f.apiStatus.Store(http.StatusUnauthorized)
	if f.pa.Serve() == 0 {
		t.Fatal("authentication failure was treated as transient")
	}
	assertMilkyState(t, f.pa, StateConnectionFailed)
	f.pa.lifecycleMu.Lock()
	active, enabled, err := f.pa.sessionActive, f.pa.EndPoint.Enable, f.pa.sessionContext.Err()
	f.pa.lifecycleMu.Unlock()
	if active || enabled || err == nil || f.apiCalls.Load() != 1 {
		t.Fatal("authentication failure retained a retrying session")
	}
}

func TestMilkySDKPermanentFailureStopsMonitor(t *testing.T) {
	f := newMilkyRecoveryFixture(t)
	f.apiStatus.Store(http.StatusUnauthorized)
	f.pa.onMilkyBotOffline(f.session, "session expired")
	assertMilkyState(t, f.pa, StateConnectionFailed)
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("authentication failure did not stop monitoring")
	}
}

func TestMilkySDKFailureClassification(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			_, err := probeMilkyLogin(context.Background(), server.Client(), server.URL, "", "QQ:10010")
			wantPermanent := status != http.StatusTooManyRequests && status != http.StatusServiceUnavailable
			if err == nil || errors.Is(err, errMilkyPermanentFailure) != wantPermanent {
				t.Fatalf("unexpected failure classification for HTTP %d: %v", status, err)
			}
		})
	}
}

func TestMilkySDKInitialUnknownAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"uin":10010,"nickname":"MilkyBot"}}`))
	}))
	defer server.Close()
	info, err := probeMilkyLogin(context.Background(), server.Client(), server.URL, "", "")
	if err != nil || info == nil || info.UIN != 10010 || info.Nickname != "MilkyBot" {
		t.Fatalf("new account login information was rejected: %v", err)
	}
	if _, err := probeMilkyLogin(context.Background(), server.Client(), server.URL, "", "QQ:20020"); !errors.Is(err, errMilkyPermanentFailure) {
		t.Fatal("changed account was not rejected as a permanent failure")
	}
}
