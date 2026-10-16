//nolint:testpackage
package dice

import (
	"context"
	"errors"
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

type milkyHealthFixture struct {
	pa         *PlatformAdapterMilky
	session    *milky.Session
	apiHealthy atomic.Bool
	loggedIn   atomic.Bool
	apiStatus  atomic.Int32
	apiCalls   atomic.Int32
	wsHealthy  atomic.Bool
	peers      chan *websocket.Conn
	done       chan struct{}
}

func newMilkyGatewayFixture(t *testing.T, mode string) *milkyHealthFixture {
	t.Helper()
	_, pa := newMilkyStartupTestAdapter(t, mode)
	f := &milkyHealthFixture{pa: pa, peers: make(chan *websocket.Conn, 16)}
	f.apiHealthy.Store(true)
	f.loggedIn.Store(true)
	f.apiStatus.Store(http.StatusOK)
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
			_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"uin":10010,"nickname":"MilkyBot"}}`))
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
				t.Error("health monitor did not stop")
			}
		}
		server.Close()
	})
	return f
}

func newMilkyHealthFixture(t *testing.T, mode string) *milkyHealthFixture {
	t.Helper()
	f := newMilkyGatewayFixture(t, mode)
	session := f.session
	f.pa.startMilkySession(session, 0)
	f.pa.prepareMilkyTransport(session)
	if session.ShouldReconnectOnError {
		t.Fatal("SDK auto reconnect must be disabled")
	}
	if err := f.pa.openMilkyTransport(session); err != nil {
		t.Fatal(err)
	}
	assertMilkyState(t, f.pa, StateConnecting)
	f.pa.finishMilkySession(session, &milky.LoginInfo{UIN: 10010, Nickname: "MilkyBot"})
	f.done = make(chan struct{})
	go func() {
		defer close(f.done)
		f.pa.watchMilkyHealth(f.pa.sessionContext, session, 20*time.Millisecond, 100*time.Millisecond)
	}()
	return f
}

func TestMilkyHealthDisconnectAndRecovery(t *testing.T) {
	for _, mode := range []string{"", "yogurt", "lagrangeV2"} {
		t.Run(mode, func(t *testing.T) {
			f := newMilkyHealthFixture(t, mode)
			assertMilkyState(t, f.pa, StateConnected)
			f.apiHealthy.Store(false)
			assertMilkyState(t, f.pa, StateDisconnected)
			f.pa.lifecycleMu.Lock()
			enabled := f.pa.EndPoint.Enable
			f.pa.lifecycleMu.Unlock()
			if !enabled {
				t.Fatal("temporary outage disabled the account")
			}
			// REST 恢复但 WS 握手失败时仍须显示断开。
			f.wsHealthy.Store(false)
			f.apiHealthy.Store(true)
			_ = (<-f.peers).Close()
			time.Sleep(1100 * time.Millisecond)
			assertMilkyState(t, f.pa, StateDisconnected)
			f.wsHealthy.Store(true)
			assertMilkyState(t, f.pa, StateConnected)
		})
	}
}

func TestMilkyHealthDetectsEventGatewayFailure(t *testing.T) {
	f := newMilkyHealthFixture(t, "")
	f.wsHealthy.Store(false)
	_ = (<-f.peers).Close()
	assertMilkyState(t, f.pa, StateDisconnected)
	f.wsHealthy.Store(true)
	assertMilkyState(t, f.pa, StateConnected)
}

func TestMilkyHealthRecoversBotOfflineAndIgnoresStoppedSessions(t *testing.T) {
	f := newMilkyHealthFixture(t, "")
	f.loggedIn.Store(false)
	f.pa.onMilkyBotOffline(f.session, "kicked offline")
	time.Sleep(100 * time.Millisecond)
	assertMilkyState(t, f.pa, StateDisconnected)
	// QQ 登录恢复但 WS 仍不可用时，不能显示已连接。
	f.wsHealthy.Store(false)
	f.loggedIn.Store(true)
	time.Sleep(1100 * time.Millisecond)
	assertMilkyState(t, f.pa, StateDisconnected)
	f.wsHealthy.Store(true)
	// 不发送任何 QQ 消息，成功的登录探测与 WS 连接应足以恢复状态。
	assertMilkyState(t, f.pa, StateConnected)
	f.pa.stopMilkySession()
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
		t.Fatal("disabled account retained health monitor")
	}
}

func TestMilkyHealthRejectsInvalidResponses(t *testing.T) {
	for _, body := range []string{
		`{"status":"failed","retcode":0,"data":{"uin":10010}}`,
		`{"status":"ok","retcode":1,"data":{"uin":10010}}`,
		`{"status":"ok","retcode":0,"data":null}`,
		`{"status":"ok","retcode":0,"data":{"uin":0}}`,
		`{"status":"ok","retcode":0,"data":{"uin":20020}}`,
		`not JSON`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			if _, err := probeMilkyHealth(context.Background(), server.Client(), server.URL, "", "QQ:10010"); err == nil {
				t.Fatal("invalid login response reported healthy")
			}
		})
	}
}

func TestMilkyHealthRequestCancellation(t *testing.T) {
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
		_, err := probeMilkyHealth(ctx, server.Client(), server.URL, "", "QQ:10010")
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
			f := newMilkyGatewayFixture(t, "yogurt")
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
	f := newMilkyGatewayFixture(t, "yogurt")
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

func TestMilkyHealthPermanentFailureStopsMonitor(t *testing.T) {
	f := newMilkyHealthFixture(t, "yogurt")
	f.apiStatus.Store(http.StatusUnauthorized)
	assertMilkyState(t, f.pa, StateConnectionFailed)
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("authentication failure did not stop monitoring")
	}
}

func TestMilkyHealthFailureClassification(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			_, err := probeMilkyHealth(context.Background(), server.Client(), server.URL, "", "QQ:10010")
			wantPermanent := status != http.StatusTooManyRequests && status != http.StatusServiceUnavailable
			if err == nil || errors.Is(err, errMilkyPermanentFailure) != wantPermanent {
				t.Fatalf("unexpected failure classification for HTTP %d: %v", status, err)
			}
		})
	}
}

func TestMilkyHealthInitialUnknownAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"uin":10010,"nickname":"MilkyBot"}}`))
	}))
	defer server.Close()
	info, err := probeMilkyHealth(context.Background(), server.Client(), server.URL, "", "")
	if err != nil || info == nil || info.UIN != 10010 || info.Nickname != "MilkyBot" {
		t.Fatalf("new account login information was rejected: %v", err)
	}
	if _, err := probeMilkyHealth(context.Background(), server.Client(), server.URL, "", "QQ:20020"); !errors.Is(err, errMilkyPermanentFailure) {
		t.Fatal("changed account was not rejected as a permanent failure")
	}
}
