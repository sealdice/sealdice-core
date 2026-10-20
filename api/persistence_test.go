package api //nolint:testpackage

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/dice"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
	sqliteengine "sealdice-core/utils/dboperator/engine/sqlite"
)

func persistenceDice(t *testing.T) *dice.Dice {
	t.Helper()
	previousDice, previousDM := myDice, dm
	t.Cleanup(func() { myDice, dm = previousDice, previousDM })
	d := &dice.Dice{
		BaseConfig:      dice.BaseConfig{DataDir: t.TempDir()},
		Logger:          zap.NewNop().Sugar(),
		TextMapRaw:      dice.TextTemplateWithWeightDict{},
		TextMapHelpInfo: dice.TextTemplateWithHelpDict{},
	}
	d.Config.BanList = &dice.BanListInfo{Map: new(dice.SyncMap[string, *dice.BanListInfoItem])}
	dm = &dice.DiceManager{Dice: d}
	dm.AccessTokens.Store("persistence-test", true)
	d.Parent, myDice = dm, d
	return d
}

func persistenceRequest(handler echo.HandlerFunc, body string) *httptest.ResponseRecorder {
	e := echo.New()
	e.POST("/write", handler)
	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Token", "persistence-test")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestCustomTextSaveReportsPersistenceFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "write-failure"}[fail], func(t *testing.T) {
			d := persistenceDice(t)
			want := http.StatusOK
			if fail {
				if err := os.WriteFile(filepath.Join(d.BaseConfig.DataDir, "configs"), []byte("blocked"), 0o600); err != nil {
					t.Fatal(err)
				}
				want = http.StatusInternalServerError
			}
			rec := persistenceRequest(customTextSave, `{"category":"test","data":{}}`)
			if rec.Code != want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, want, rec.Body.String())
			}
		})
	}
}

type persistenceDB struct {
	engine.DatabaseOperator
	db *gorm.DB
}

func (db persistenceDB) GetDataDB(constant.DBMode) *gorm.DB { return db.db }

func TestBanDeleteReportsDatabaseFailureAndPreservesMemory(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "database-failure"}[fail], func(t *testing.T) {
			d := persistenceDice(t)
			db, err := sqliteengine.SQLiteDBInit(filepath.Join(t.TempDir(), "ban.db"), false)
			if err != nil {
				t.Fatal(err)
			}
			conn, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Close() })
			if !fail {
				if err := db.AutoMigrate(&model.BanInfo{}); err != nil {
					t.Fatal(err)
				}
			}
			d.DBOperator = persistenceDB{db: db}
			d.Config.BanList.Map.Store("QQ:1", &dice.BanListInfoItem{ID: "QQ:1"})
			rec := persistenceRequest(banMapDeleteOne, `{"id":"QQ:1"}`)
			want := http.StatusOK
			if fail {
				want = http.StatusInternalServerError
			}
			if rec.Code != want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, want, rec.Body.String())
			}
			if _, kept := d.Config.BanList.Map.Load("QQ:1"); kept != fail {
				t.Fatalf("entry kept = %v, want %v", kept, fail)
			}
		})
	}
}
