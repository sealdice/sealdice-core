//nolint:testpackage
package dice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"

	"sealdice-core/model"
)

func TestMilkyReconnectMessageStillValidatesAccount(t *testing.T) {
	for _, uin := range []int64{10010, 20020} {
		t.Run(strconv.FormatInt(uin, 10), func(t *testing.T) {
			f := newMilkyGatewayFixture(t)
			f.pa.startMilkySession(f.session, 0)
			f.pa.prepareMilkyTransport(f.session)
			connected := make(chan struct{}, 2)
			f.session.OnConnectionChange = func(s *milky.Session, online bool) {
				f.pa.onMilkyTransportChange(s, online)
				if online {
					connected <- struct{}{}
				}
			}
			accepted := make(chan bool, 1)
			f.session.AddHandler(func(s *milky.Session, _ *milky.ReceiveMessage) {
				accepted <- f.pa.onMilkyMessage(s)
			})
			if err := f.pa.refreshMilkyTransport(f.session, newMilkyLoginClient(time.Second)); err != nil {
				t.Fatal(err)
			}
			<-connected
			if !f.pa.onMilkyMessage(f.session) {
				t.Fatal("verified account rejected a message")
			}
			f.uin.Store(uin)
			_ = (<-f.peers).Close()
			select {
			case <-connected:
			case <-time.After(5 * time.Second):
				t.Fatal("SDK did not reconnect")
			}
			peer := <-f.peers
			if err := peer.WriteMessage(websocket.TextMessage, []byte(`{"event_type":"message_receive","data":{"message_scene":"temp","sender_id":12345}}`)); err != nil {
				t.Fatal(err)
			}
			select {
			case handled := <-accepted:
				if handled {
					t.Fatal("unverified account message was accepted")
				}
			case <-time.After(time.Second):
				t.Fatal("message was not delivered")
			}
			// 模拟消息回调先于恢复任务运行；消息不能取消重连后的登录验证。
			assertMilkyState(t, f.pa, StateDisconnected)
			f.startRecovery()
			if uin == 10010 {
				assertMilkyState(t, f.pa, StateConnected)
				if !f.pa.onMilkyMessage(f.session) {
					t.Fatal("account validation did not resume message handling")
				}
			} else {
				assertMilkyState(t, f.pa, StateConnectionFailed)
				if f.pa.onMilkyMessage(f.session) {
					t.Fatal("changed account message was accepted")
				}
			}
			if f.apiCalls.Load() < 2 {
				t.Fatal("reconnected account was never verified")
			}
		})
	}
}

func newMilkyPersistenceDB(t *testing.T) *mockDatabaseOperator {
	t.Helper()
	db, err := newMockDatabaseOperator(filepath.Join(t.TempDir(), "endpoints.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := db.db.AutoMigrate(&model.EndpointInfo{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func readMilkySavedEndpoint(t *testing.T, d *Dice) *EndPointInfo {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		ImSession IMSession `yaml:"imSession"`
	}
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.ImSession.EndPoints) != 1 {
		t.Fatalf("saved %d endpoints, want 1", len(saved.ImSession.EndPoints))
	}
	return saved.ImSession.EndPoints[0]
}

func TestMilkyInitialRecoveryPersistsIdentity(t *testing.T) {
	db := newMilkyPersistenceDB(t)
	f := newMilkyGatewayFixture(t)
	d := f.pa.EndPoint.Session.Parent
	d.DBOperator = db
	d.ImSession.EndPoints = []*EndPointInfo{f.pa.EndPoint}
	f.pa.EndPoint.UserID = ""
	f.pa.startMilkySession(f.session, 0)
	f.pa.prepareMilkyTransport(f.session)
	f.apiHealthy.Store(false)
	if err := f.pa.refreshMilkyTransport(f.session, newMilkyLoginClient(time.Second)); err == nil {
		t.Fatal("initial API outage did not fail")
	}
	// 对齐 Serve 的首次失败保存，后台恢复必须自行保存新取得的账号信息。
	d.MarkModified()
	d.Save(false)
	if ep := readMilkySavedEndpoint(t, d); ep.UserID != "" || d.LastUpdatedTime != 0 {
		t.Fatal("initial failure did not save a clean configuration without identity")
	}
	f.apiHealthy.Store(true)
	f.startRecovery()
	deadline := time.Now().Add(3 * time.Second)
	for {
		data, err := os.ReadFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))
		if err == nil && strings.Contains(string(data), "QQ:10010") && strings.Contains(string(data), "MilkyBot") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background recovery did not save the account identity")
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.pa.stopMilkySession()
	<-f.done
	ep := readMilkySavedEndpoint(t, d)
	if ep.UserID != "QQ:10010" || ep.Nickname != "MilkyBot" || !ep.Enable {
		t.Fatalf("reloaded endpoint has incorrect identity: %+v", ep.EndPointInfoBase)
	}
}

func TestMilkyExpiredQRCodeReportsFailure(t *testing.T) {
	clientSource, err := filepath.Abs("testdata/milky-client/main.go")
	if err != nil {
		t.Fatal(err)
	}
	clientDir := t.TempDir()
	executable := filepath.Join(clientDir, "client")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if output, buildErr := exec.CommandContext(t.Context(), "go", "build", "-o", executable, clientSource).CombinedOutput(); buildErr != nil {
		t.Fatalf("build test client: %v\n%s", buildErr, output)
	}
	clientData, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(clientDir, "milky"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(clientDir)
	for _, mode := range []string{"yogurt", "lagrangeV2"} {
		t.Run(mode, func(t *testing.T) {
			name := mode
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			if err := os.WriteFile(filepath.Join(clientDir, "milky", name), clientData, 0o755); err != nil {
				t.Fatal(err)
			}
			db := newMilkyPersistenceDB(t)
			d, pa := newMilkyStartupTestAdapter(t, mode)
			d.Parent = &DiceManager{}
			d.DBOperator = db
			d.ImSession.EndPoints = []*EndPointInfo{pa.EndPoint}
			ServeMilkyBuiltIn(d, pa.EndPoint)
			t.Cleanup(func() { BuiltinMilkyClientKill(d, pa.EndPoint) })
			deadline := time.Now().Add(5 * time.Second)
			for {
				pa.lifecycleMu.Lock()
				process, done := pa.MilkyProcess, pa.processDone
				pa.lifecycleMu.Unlock()
				if process != nil {
					if _, err := fmt.Fprintln(process.StdIn, "expire"); err != nil {
						t.Fatal(err)
					}
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						t.Fatal("expired client did not exit")
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("test client did not start")
				}
				time.Sleep(time.Millisecond)
			}
			assertMilkyState(t, pa, StateConnectionFailed)
			if pa.BuiltInLoginState != MilkyLoginStateFailed || pa.MilkyProcess != nil || pa.sessionActive {
				t.Fatal("expired client retained login or process state")
			}
			if ep := readMilkySavedEndpoint(t, d); ep.State != StateConnectionFailed {
				t.Fatalf("saved endpoint state = %v, want failed", ep.State)
			}
		})
	}
}
