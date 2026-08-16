package basesetting_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"github.com/robfig/cron/v3"

	. "sealdice-core/api/v2/basesetting"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
	"sealdice-core/utils/constant"
	sqliteengine "sealdice-core/utils/dboperator/engine/sqlite"
)

type testDatabaseOperator struct {
	db *gorm.DB
}

func newTestDatabaseOperator(t *testing.T) *testDatabaseOperator {
	t.Helper()

	db, err := sqliteengine.SQLiteDBInit(filepath.Join(t.TempDir(), "test.db"), false)
	if err != nil {
		t.Fatalf("open test sqlite: %v", err)
	}
	return &testDatabaseOperator{db: db}
}

func (o *testDatabaseOperator) Init(context.Context) error             { return nil }
func (o *testDatabaseOperator) Type() string                           { return "test-sqlite" }
func (o *testDatabaseOperator) DBCheck()                               {}
func (o *testDatabaseOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.db }
func (o *testDatabaseOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.db }
func (o *testDatabaseOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.db }
func (o *testDatabaseOperator) Close() {
	sqlDB, err := o.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func newTestBaseSettingService(t *testing.T) (*Service, *dice.Dice) {
	t.Helper()

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	db := newTestDatabaseOperator(t)
	t.Cleanup(db.Close)

	d := &dice.Dice{
		Logger:     logger.M(),
		Cron:       cron.New(),
		DBOperator: db,
	}
	d.BaseConfig.Name = "test"
	d.BaseConfig.DataDir = dataDir
	d.Config = dice.NewConfig(d)

	dm := &dice.DiceManager{
		Dice:         []*dice.Dice{d},
		ServeAddress: "127.0.0.1:3211",
	}
	d.Parent = dm

	return NewService(dm), d
}

func TestGetValueIncludesOfficialQQAndTrayTooltipFields(t *testing.T) {
	svc, testDice := newTestBaseSettingService(t)
	testDice.Config.OfficialQQFileSendBase64 = true
	testDice.Config.OfficialQQUseMarkdown = true

	resp, err := svc.GetValue(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetValue returned error: %v", err)
	}

	data, err := json.Marshal(resp.Body.Item)
	if err != nil {
		t.Fatalf("marshal value resp: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal value resp: %v", err)
	}

	if value, ok := got["officialQQFileSendBase64"].(bool); !ok || !value {
		t.Fatalf("officialQQFileSendBase64 = %#v, want true", got["officialQQFileSendBase64"])
	}
	if value, ok := got["officialQQUseMarkdown"].(bool); !ok || !value {
		t.Fatalf("officialQQUseMarkdown = %#v, want true", got["officialQQUseMarkdown"])
	}
	if value, ok := got["trayTooltip"].(string); !ok || value != "" {
		t.Fatalf("trayTooltip = %#v, want empty string", got["trayTooltip"])
	}
}

func TestSetValueAcceptsOfficialQQAndTrayTooltipPatch(t *testing.T) {
	svc, _ := newTestBaseSettingService(t)

	if _, err := svc.SetValue(t.Context(), &BaseSettingUpdateReq{
		Body: map[string]any{
			"officialQQFileSendBase64": true,
			"officialQQUseMarkdown":    true,
			"trayTooltip":              "海豹提示",
		},
	}); err != nil {
		t.Fatalf("SetValue returned error: %v", err)
	}

	resp, err := svc.GetValue(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetValue returned error: %v", err)
	}

	data, err := json.Marshal(resp.Body.Item)
	if err != nil {
		t.Fatalf("marshal value resp: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal value resp: %v", err)
	}
	if value, ok := got["officialQQFileSendBase64"].(bool); !ok || !value {
		t.Fatalf("officialQQFileSendBase64 = %#v, want true", got["officialQQFileSendBase64"])
	}
	if value, ok := got["officialQQUseMarkdown"].(bool); !ok || !value {
		t.Fatalf("officialQQUseMarkdown = %#v, want true", got["officialQQUseMarkdown"])
	}
	if value, ok := got["trayTooltip"].(string); !ok || value != "海豹提示" {
		t.Fatalf("trayTooltip = %#v, want 海豹提示", got["trayTooltip"])
	}
}
