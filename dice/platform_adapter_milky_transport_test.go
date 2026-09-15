//nolint:testpackage
package dice

import (
	"context"
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
	wsHealthy  atomic.Bool
	peers      chan *websocket.Conn
	done       chan struct{}
}

func newMilkyHealthFixture(t *testing.T, mode string) *milkyHealthFixture {
	t.Helper()
	f := &milkyHealthFixture{pa: &PlatformAdapterMilky{EndPoint: &EndPointInfo{}, BuiltInMode: mode}, peers: make(chan *websocket.Conn, 16), done: make(chan struct{})}
	f.apiHealthy.Store(true)
	f.wsHealthy.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/get_login_info" {
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !f.apiHealthy.Load() {
				http.Error(w, "gateway unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","retcode":0,"data":{"uin":10010,"nickname":"MilkyBot"}}`))
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
	f.pa.startMilkySession(session, 0)
	f.pa.prepareMilkyTransport(session)
	if session.ShouldReconnectOnError {
		t.Fatal("SDK auto reconnect must be disabled")
	}
	if err := f.pa.openMilkyTransport(session); err != nil {
		server.Close()
		t.Fatal(err)
	}
	assertMilkyState(t, f.pa, StateConnecting)
	f.pa.finishMilkySession(session, &milky.LoginInfo{UIN: 10010, Nickname: "MilkyBot"})
	go func() {
		defer close(f.done)
		f.pa.watchMilkyHealth(f.pa.sessionContext, session, 20*time.Millisecond, 100*time.Millisecond)
	}()
	t.Cleanup(func() {
		f.pa.stopMilkySession()
		select {
		case <-f.done:
		case <-time.After(3 * time.Second):
			t.Error("health monitor did not stop")
		}
		server.Close()
	})
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

func TestMilkyHealthPreservesBotOfflineAndIgnoresStoppedSessions(t *testing.T) {
	f := newMilkyHealthFixture(t, "")
	f.pa.onMilkyBotOffline(f.session, "kicked offline")
	time.Sleep(100 * time.Millisecond)
	assertMilkyState(t, f.pa, StateDisconnected)
	f.pa.onMilkyMessage(f.session)
	assertMilkyState(t, f.pa, StateConnected)
	f.pa.stopMilkySession()
	f.pa.lifecycleMu.Lock()
	f.pa.EndPoint.State = StateDisconnected
	f.pa.EndPoint.Enable = false
	f.pa.lifecycleMu.Unlock()
	f.pa.onMilkyConnectionChange(f.session, true)
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
			defer server.Close()
			if err := probeMilkyHealth(context.Background(), server.Client(), server.URL, "", "QQ:10010"); err == nil {
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
	go func() { done <- probeMilkyHealth(ctx, server.Client(), server.URL, "", "QQ:10010") }()
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
