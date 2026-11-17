package basesetting_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"

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
	t.Chdir(t.TempDir())

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
		Dice:         d,
		ServeAddress: "127.0.0.1:3211",
	}
	d.Parent = dm

	return NewService(dm), d
}

func TestGetValueIncludesOfficialQQAndTrayTooltipFields(t *testing.T) {
	svc, testDice := newTestBaseSettingService(t)
	testDice.Config.OfficialQQFileSendBase64 = true
	testDice.Config.OfficialQQUseMarkdown = true
	testDice.Config.OfficialQQUseNameCard = true
	testDice.Config.OfficialQQNameCardTmpl = "<b>{card}</b>"

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
	if value, ok := got["officialQQUseNameCard"].(bool); !ok || !value {
		t.Fatalf("officialQQUseNameCard = %#v, want true", got["officialQQUseNameCard"])
	}
	if value, ok := got["officialQQUseNameCardTemplate"].(string); !ok || value != "<b>{card}</b>" {
		t.Fatalf("officialQQUseNameCardTemplate = %#v, want %q", got["officialQQUseNameCardTemplate"], "<b>{card}</b>")
	}
	if value, ok := got["trayTooltip"].(string); !ok || value != "" {
		t.Fatalf("trayTooltip = %#v, want empty string", got["trayTooltip"])
	}
	if value, ok := got["diceRandomGMEnabled"].(bool); !ok || value {
		t.Fatalf("diceRandomGMEnabled = %#v, want false", got["diceRandomGMEnabled"])
	}
}

func TestSetValueAcceptsOfficialQQAndTrayTooltipPatch(t *testing.T) {
	svc, _ := newTestBaseSettingService(t)

	if _, err := svc.SetValue(t.Context(), &BaseSettingUpdateReq{
		Body: BaseSettingPatch{
			OfficialQQFileSendBase64: ptr(true),
			OfficialQQUseMarkdown:    ptr(true),
			OfficialQQUseNameCard:    ptr(true),
			OfficialQQNameCardTmpl:   ptr("<i>{card}</i>"),
			TrayTooltip:              ptr("海豹提示"),
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
	if value, ok := got["officialQQUseNameCard"].(bool); !ok || !value {
		t.Fatalf("officialQQUseNameCard = %#v, want true", got["officialQQUseNameCard"])
	}
	if value, ok := got["officialQQUseNameCardTemplate"].(string); !ok || value != "<i>{card}</i>" {
		t.Fatalf("officialQQUseNameCardTemplate = %#v, want %q", got["officialQQUseNameCardTemplate"], "<i>{card}</i>")
	}
	if value, ok := got["trayTooltip"].(string); !ok || value != "海豹提示" {
		t.Fatalf("trayTooltip = %#v, want 海豹提示", got["trayTooltip"])
	}
}

func TestOfficialQQNameCardRequiresMarkdown(t *testing.T) {
	t.Run("rejects explicit enable without markdown", func(t *testing.T) {
		svc, d := newTestBaseSettingService(t)
		d.Config.OfficialQQUseMarkdown = false
		d.Config.OfficialQQUseNameCard = false

		rec := putSettings(t, svc, `{"officialQQUseMarkdown":false,"officialQQUseNameCard":true}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		if d.Config.OfficialQQUseNameCard {
			t.Fatal("name card should not be enabled without markdown")
		}
	})

	t.Run("disabling markdown also disables name card", func(t *testing.T) {
		svc, d := newTestBaseSettingService(t)
		d.Config.OfficialQQUseMarkdown = true
		d.Config.OfficialQQUseNameCard = true

		rec := putSettings(t, svc, `{"officialQQUseMarkdown":false}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		if d.Config.OfficialQQUseNameCard {
			t.Fatal("name card should be disabled when markdown is disabled")
		}
	})

	t.Run("accepts both enabled", func(t *testing.T) {
		svc, d := newTestBaseSettingService(t)
		d.Config.OfficialQQUseMarkdown = false
		d.Config.OfficialQQUseNameCard = false

		rec := putSettings(t, svc, `{"officialQQUseMarkdown":true,"officialQQUseNameCard":true}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		if !d.Config.OfficialQQUseMarkdown || !d.Config.OfficialQQUseNameCard {
			t.Fatalf("markdown=%t nameCard=%t, want both true",
				d.Config.OfficialQQUseMarkdown, d.Config.OfficialQQUseNameCard)
		}
	})
}

func TestOfficialQQNameCardTemplateRejectsOverlongValue(t *testing.T) {
	svc, d := newTestBaseSettingService(t)
	d.Config.OfficialQQNameCardTmpl = "<b>{card}</b>"

	rec := putSettings(t, svc, `{"officialQQUseNameCardTemplate":"`+
		strings.Repeat("x", dice.MaxOfficialQQNameCardTemplateRunes+1)+`"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if d.Config.OfficialQQNameCardTmpl != "<b>{card}</b>" {
		t.Fatalf("template changed to %q", d.Config.OfficialQQNameCardTmpl)
	}
}

func TestPatchRejectsGMModeWithoutEnable(t *testing.T) {
	svc, d := newTestBaseSettingService(t)
	d.Config.DiceRandomMode = string(dice.DiceRandomModePCG)
	d.Config.DiceRandomGMEnabled = false

	rec := putSettings(t, svc, `{"diceRandomMode":"gm"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if d.Config.DiceRandomMode != string(dice.DiceRandomModePCG) {
		t.Fatalf("DiceRandomMode = %q, want pcg", d.Config.DiceRandomMode)
	}
}

func TestPatchRejectsDisablingGM(t *testing.T) {
	svc, d := newTestBaseSettingService(t)
	d.Config.DiceRandomGMEnabled = true

	rec := putSettings(t, svc, `{"diceRandomGMEnabled":false}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if !d.Config.DiceRandomGMEnabled {
		t.Fatal("rejected patch disabled GM source")
	}
}

func ptr[T any](value T) *T { return &value }

func putSettings(t *testing.T, svc *Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("settings test", "1"))
	svc.RegisterProtectedRoutes(huma.NewGroup(api, "/settings"))
	req := httptest.NewRequest(http.MethodPut, "/settings/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func TestInvalidPatchLeavesSettingsUnchanged(t *testing.T) {
	for _, body := range []string{
		`{"commandPrefix":["!"],"trayTooltip":"` + strings.Repeat("x", dice.MaxTrayTooltipPrefixLength+1) + `"}`,
		`{"trustOnlyMode":"yes"}`,
		`{"body":{"trustOnlyMode":true}}`,
		`{"personalBurst":1.5}`,
		`{"trustOnlyMode":true,"messageDelayRangeStart":4,"messageDelayRangeEnd":2}`,
		`{"trustOnlyMode":true,"aliveNoticeValue":"bad cron"}`,
	} {
		t.Run(body, func(t *testing.T) {
			svc, d := newTestBaseSettingService(t)
			d.CommandPrefix = []string{"."}
			d.Config.TrustOnlyMode = false
			d.Config.MessageDelayRangeStart, d.Config.MessageDelayRangeEnd = 1, 3
			rec := putSettings(t, svc, body)
			if rec.Code < 400 || rec.Code >= 500 {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
			}
			if len(d.CommandPrefix) != 1 || d.CommandPrefix[0] != "." || d.Config.TrustOnlyMode ||
				d.Config.MessageDelayRangeStart != 1 || d.Config.MessageDelayRangeEnd != 3 {
				t.Fatal("rejected patch changed running settings")
			}
		})
	}
}

func TestPatchValidatesNewDelayPair(t *testing.T) {
	t.Chdir(t.TempDir())
	svc, d := newTestBaseSettingService(t)
	d.Config.MessageDelayRangeStart, d.Config.MessageDelayRangeEnd = 5, 6
	rec := putSettings(t, svc, `{"messageDelayRangeStart":1,"messageDelayRangeEnd":2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if d.Config.MessageDelayRangeStart != 1 || d.Config.MessageDelayRangeEnd != 2 {
		t.Fatal("new interval was not applied")
	}
}

func TestPatchDistinguishesMissingFalseAndEmptyAndPreservesPasswords(t *testing.T) {
	t.Chdir(t.TempDir())
	svc, d := newTestBaseSettingService(t)
	d.Config.TrustOnlyMode, d.Config.BotExtFreeSwitch = true, true
	d.Config.MailPassword = "existing-mail-secret"
	d.Parent.UIPasswordHash = "existing-ui-hash"
	d.Parent.AccessTokens.Store("existing-token", true)
	d.CommandPrefix = []string{"."}
	rec := putSettings(t, svc, `{"trustOnlyMode":false,"commandPrefix":[],"mailPassword":"******","uiPassword":"------"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if d.Config.TrustOnlyMode || !d.Config.BotExtFreeSwitch || len(d.CommandPrefix) != 0 {
		t.Fatal("false, empty and omitted fields were not distinguished")
	}
	if d.Config.MailPassword != "existing-mail-secret" || d.Parent.UIPasswordHash != "existing-ui-hash" || !d.Parent.AccessTokens.Exists("existing-token") {
		t.Fatal("masked passwords changed credentials or sessions")
	}
}

func TestPatchRejectsNullExtensionBeforeApplyingAnyFields(t *testing.T) {
	svc, d := newTestBaseSettingService(t)
	d.Config.TrustOnlyMode = false
	rec := putSettings(t, svc, `{"trustOnlyMode":true,"extDefaultSettings":[null]}`)
	if rec.Code < 400 || rec.Code >= 500 || d.Config.TrustOnlyMode {
		t.Fatalf("invalid extension update: status=%d trustOnly=%v", rec.Code, d.Config.TrustOnlyMode)
	}
}
