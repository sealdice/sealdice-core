package config_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	. "sealdice-core/api/v2/config"

	"gorm.io/gorm"

	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/model/common/request"
	"sealdice-core/utils/constant"
	sqliteengine "sealdice-core/utils/dboperator/engine/sqlite"
	"sealdice-core/utils/public_dice"

	"github.com/robfig/cron/v3"
)

type configTestDatabaseOperator struct {
	db *gorm.DB
}

func newConfigTestDatabaseOperator(t *testing.T) *configTestDatabaseOperator {
	t.Helper()

	db, err := sqliteengine.SQLiteDBInit(filepath.Join(t.TempDir(), "test.db"), false)
	if err != nil {
		t.Fatalf("open test sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EndpointInfo{}); err != nil {
		t.Fatalf("migrate endpoint table: %v", err)
	}
	return &configTestDatabaseOperator{db: db}
}

func (o *configTestDatabaseOperator) Init(context.Context) error             { return nil }
func (o *configTestDatabaseOperator) Type() string                           { return "test-sqlite" }
func (o *configTestDatabaseOperator) DBCheck()                               {}
func (o *configTestDatabaseOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.db }
func (o *configTestDatabaseOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.db }
func (o *configTestDatabaseOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.db }
func (o *configTestDatabaseOperator) Close() {
	sqlDB, err := o.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	db := newConfigTestDatabaseOperator(t)
	t.Cleanup(db.Close)

	d := &dice.Dice{
		Logger:     logger.M(),
		Cron:       cron.New(),
		DBOperator: db,
	}
	d.BaseConfig.Name = "test"
	d.BaseConfig.DataDir = dataDir
	d.Config = dice.NewConfig(d)
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	d.AttrsManager = &dice.AttrsManager{}
	d.AttrsManager.Init(d)
	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm
	return NewService(dm)
}

func installPublicDiceTestServer(t *testing.T, svc *Service, registeredID string) map[string]int {
	t.Helper()

	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/dice/api/public-dice/register":
			var body struct {
				Name  string `json:"name"`
				Brief string `json:"brief"`
				Note  string `json:"note"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode register body: %v", err)
			}
			if body.Name != "公骰" || body.Brief != "简介" || body.Note != "留言" {
				t.Fatalf("unexpected register body: %+v", body)
			}
			_, _ = w.Write([]byte(`{"item":{"id":"` + registeredID + `"}}`))
		case "/dice/api/public-dice/endpoint-update":
			_, _ = w.Write([]byte(`{}`))
		case "/dice/api/public-dice/tick-update":
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	svc.Dice().PublicDice = public_dice.NewClient(server.URL, "")
	return calls
}

func TestReplyConfigRoundTrip(t *testing.T) {
	svc := newTestService(t)
	defer svc.Dice().AttrsManager.Stop()

	getResp, err := svc.GetReplyConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetReplyConfig returned error: %v", err)
	}
	if getResp.Body.Item.CustomReplyConfigEnable {
		t.Fatalf("default custom reply switch should be false")
	}

	setResp, err := svc.SetReplyConfig(t.Context(), &ReplyConfigReq{
		Body: ReplyModuleConfig{CustomReplyConfigEnable: true},
	})
	if err != nil {
		t.Fatalf("SetReplyConfig returned error: %v", err)
	}
	if !setResp.Body.Item.CustomReplyConfigEnable {
		t.Fatalf("custom reply switch should be true")
	}
}

func TestAdvancedConfigRoundTrip(t *testing.T) {
	svc := newTestService(t)
	defer svc.Dice().AttrsManager.Stop()

	_, err := svc.SetAdvancedConfig(t.Context(), &AdvancedConfigReq{
		Body: dice.AdvancedConfig{
			Show:                 true,
			Enable:               true,
			StoryLogBackendUrl:   "https://example.com",
			StoryLogApiVersion:   "v2",
			StoryLogBackendToken: "token",
		},
	})
	if err != nil {
		t.Fatalf("SetAdvancedConfig returned error: %v", err)
	}

	getResp, err := svc.GetAdvancedConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetAdvancedConfig returned error: %v", err)
	}
	if !getResp.Body.Item.Show || !getResp.Body.Item.Enable {
		t.Fatalf("advanced config flags not persisted")
	}
	if getResp.Body.Item.StoryLogBackendUrl != "https://example.com" {
		t.Fatalf("StoryLogBackendUrl = %q, want https://example.com", getResp.Body.Item.StoryLogBackendUrl)
	}
}

func TestPublicDiceConfigGetReturnsConfigAndSimplifiedEndpoints(t *testing.T) {
	svc := newTestService(t)
	defer svc.Dice().AttrsManager.Stop()

	svc.Dice().Config.PublicDiceConfig = dice.PublicDiceConfig{
		Enable: true,
		ID:     "dice-id",
		Name:   "公骰",
		Brief:  "简介",
		Note:   "留言",
		Avatar: "https://example.com/avatar.png",
	}
	svc.Dice().ImSession.EndPoints = []*dice.EndPointInfo{
		{
			EndPointInfoBase: dice.EndPointInfoBase{
				ID:           "ep-1",
				UserID:       "1001",
				Platform:     "QQ",
				ProtocolType: "onebot",
				State:        dice.StateConnected,
				IsPublic:     true,
			},
		},
		{
			EndPointInfoBase: dice.EndPointInfoBase{
				ID:           "ep-2",
				UserID:       "1002",
				Platform:     "QQ",
				ProtocolType: "milky",
				State:        dice.StateDisconnected,
				IsPublic:     false,
			},
		},
	}

	resp, err := svc.GetPublicDiceConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetPublicDiceConfig returned error: %v", err)
	}

	item := resp.Body.Item
	if item.Config.PublicDiceID != "dice-id" || !item.Config.PublicDiceEnable {
		t.Fatalf("unexpected config: %+v", item.Config)
	}
	if len(item.Endpoints) != 2 {
		t.Fatalf("endpoints len = %d, want 2", len(item.Endpoints))
	}
	if item.Endpoints[0].ID != "ep-1" || item.Endpoints[0].UserID != "1001" || !item.Endpoints[0].IsPublic {
		t.Fatalf("unexpected first endpoint: %+v", item.Endpoints[0])
	}
	if item.Endpoints[1].ProtocolType != "milky" || item.Endpoints[1].State != dice.StateDisconnected {
		t.Fatalf("unexpected second endpoint: %+v", item.Endpoints[1])
	}
}

func TestPublicDiceConfigUpdateSavesConfigAndSelectedEndpoints(t *testing.T) {
	svc := newTestService(t)
	defer svc.Dice().AttrsManager.Stop()
	calls := installPublicDiceTestServer(t, svc, "registered-id")

	svc.Dice().ImSession.EndPoints = []*dice.EndPointInfo{
		{
			EndPointInfoBase: dice.EndPointInfoBase{
				ID:           "ep-1",
				UserID:       "1001",
				Platform:     "QQ",
				ProtocolType: "onebot",
				State:        dice.StateConnected,
				IsPublic:     false,
			},
		},
		{
			EndPointInfoBase: dice.EndPointInfoBase{
				ID:           "ep-2",
				UserID:       "1002",
				Platform:     "QQ",
				ProtocolType: "milky",
				State:        dice.StateDisconnected,
				IsPublic:     true,
			},
		},
	}

	resp, err := svc.SetPublicDiceConfig(t.Context(), &PublicDiceUpdateReq{
		Body: PublicDiceUpdateBody{
			Config: PublicDiceConfig{
				PublicDiceEnable: true,
				PublicDiceID:     "",
				PublicDiceName:   "公骰",
				PublicDiceBrief:  "简介",
				PublicDiceNote:   "留言",
				PublicDiceAvatar: "https://example.com/avatar.png",
			},
			SelectedEndpointIDs: []string{"ep-1"},
		},
	})
	if err != nil {
		t.Fatalf("SetPublicDiceConfig returned error: %v", err)
	}

	if resp.Body.Item.Config.PublicDiceID != "registered-id" {
		t.Fatalf("public dice id = %q, want registered-id", resp.Body.Item.Config.PublicDiceID)
	}
	if !svc.Dice().ImSession.EndPoints[0].IsPublic {
		t.Fatalf("ep-1 should be public")
	}
	if svc.Dice().ImSession.EndPoints[1].IsPublic {
		t.Fatalf("ep-2 should not be public")
	}
	if calls["/dice/api/public-dice/register"] != 1 {
		t.Fatalf("register calls = %d, want 1", calls["/dice/api/public-dice/register"])
	}
	if calls["/dice/api/public-dice/endpoint-update"] != 1 {
		t.Fatalf("endpoint-update calls = %d, want 1", calls["/dice/api/public-dice/endpoint-update"])
	}
}
