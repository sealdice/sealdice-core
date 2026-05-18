package customtext_test

import (
	"context"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/api/v2/customtext"
	customtextm "sealdice-core/api/v2/model/customtext"
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
func (o *testDatabaseOperator) Close()                                 { _ = closeGormDB(o.db) }

func closeGormDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func newTestService(t *testing.T) (*customtext.Service, func()) {
	t.Helper()

	db := newTestDatabaseOperator(t)
	d := &dice.Dice{
		Logger:            logger.M(),
		DBOperator:        db,
		ExtRegistry:       new(dice.SyncMap[string, *dice.ExtInfo]),
		GameSystemMap:     new(dice.SyncMap[string, *dice.GameSystemTemplate]),
		DirtyGroups:       new(dice.SyncMap[string, int64]),
		TextMapRaw:        dice.TextTemplateWithWeightDict{},
		TextMapHelpInfo:   dice.TextTemplateWithHelpDict{},
		TextMapCompatible: dice.TextTemplateCompatibleDict{},
	}
	d.Config = dice.NewConfig(d)
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	d.AttrsManager = &dice.AttrsManager{}
	d.AttrsManager.Init(d)
	d.UIEndpoint = &dice.EndPointInfo{}
	d.UIEndpoint.UserID = "UI:1001"
	d.UIEndpoint.Platform = "UI"
	d.UIEndpoint.Session = d.ImSession
	dm := &dice.DiceManager{Dice: []*dice.Dice{d}}
	d.Parent = dm
	return customtext.NewServiceWithAutoSave(dm, false), func() {
		d.AttrsManager.Stop()
		db.Close()
	}
}

func TestGetTextReturnsTextsHelpInfoAndPreviewInfo(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	svc.Dice().TextMapRaw["核心"] = dice.TextTemplateWithWeight{
		"骰子名字": {{"海豹", 1}},
	}
	svc.Dice().TextMapHelpInfo["核心"] = dice.TextTemplateHelpGroup{
		"骰子名字": &dice.TextTemplateHelpItem{SubType: "基础", Vars: []string{"$t玩家"}},
	}
	inner := &dice.SyncMap[string, dice.TextItemCompatibleInfo]{}
	inner.Store("海豹", dice.TextItemCompatibleInfo{Version: "v2", TextV2: "海豹"})
	svc.Dice().TextMapCompatible.Store("核心:骰子名字", inner)

	resp, err := svc.GetText(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetText returned error: %v", err)
	}
	item := resp.Body.Item
	if got := item.Texts["核心"]["骰子名字"][0][0]; got != "海豹" {
		t.Fatalf("text = %v, want 海豹", got)
	}
	if got := item.HelpInfo["核心"]["骰子名字"].SubType; got != "基础" {
		t.Fatalf("subType = %q, want 基础", got)
	}
	if _, ok := item.PreviewInfo["核心:骰子名字"]["海豹"]; !ok {
		t.Fatalf("preview info missing")
	}
}

func TestSaveCategoryTrimsTextAndUpdatesRawMap(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.SaveCategory(t.Context(), &customtextm.SaveCategoryReq{
		Category: "核心",
		Body: request.RequestWrapper[customtextm.SaveCategoryBody]{
			Body: customtextm.SaveCategoryBody{
				Data: dice.TextTemplateWithWeight{
					"骰子名字": {{"  海豹  ", 2.0}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveCategory returned error: %v", err)
	}
	got := svc.Dice().TextMapRaw["核心"]["骰子名字"][0]
	if got[0] != "海豹" {
		t.Fatalf("text = %v, want trimmed 海豹", got[0])
	}
	if got[1] != 2 {
		t.Fatalf("weight = %#v, want int 2", got[1])
	}
	if _, ok := svc.Dice().TextMapCompatible.Load("核心:骰子名字"); !ok {
		t.Fatalf("compatible preview was not refreshed")
	}
}
