package story

import (
	"context"
	"testing"
	"time"

	storym "sealdice-core/api/v2/model/story"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/model/common/request"
	"sealdice-core/utils/constant"
	engine "sealdice-core/utils/dboperator/engine"
	sqliteengine "sealdice-core/utils/dboperator/engine/sqlite"
)

func newTestStoryService(t *testing.T) *Service {
	t.Helper()

	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	operator := &sqliteengine.SQLiteEngine{}
	if err := operator.Init(context.Background()); err != nil {
		t.Fatalf("init sqlite operator: %v", err)
	}
	t.Cleanup(operator.Close)
	if err := operator.GetLogDB(constant.WRITE).AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatalf("migrate log tables: %v", err)
	}

	d := &dice.Dice{
		Logger:     logger.M(),
		DBOperator: operator,
	}
	d.BaseConfig.Name = "story-test"
	d.BaseConfig.DataDir = dataDir
	d.Config = dice.NewConfig(d)
	d.UIEndpoint = &dice.EndPointInfo{EndPointInfoBase: dice.EndPointInfoBase{UserID: "UI:1000"}}

	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm

	return NewService(dm)
}

func insertStoryLogFixture(t *testing.T, svc *Service, logInfo model.LogInfo, items []model.LogOneItem) {
	t.Helper()

	db := svc.dice.DBOperator.GetLogDB(constant.WRITE)
	if err := db.Create(&logInfo).Error; err != nil {
		t.Fatalf("create log info: %v", err)
	}
	for i := range items {
		items[i].LogID = logInfo.ID
		if err := db.Create(&items[i]).Error; err != nil {
			t.Fatalf("create log item: %v", err)
		}
	}
}

func TestGetItemPageSupportsLogIDAndFallbackQuery(t *testing.T) {
	svc := newTestStoryService(t)
	logInfo := model.LogInfo{
		ID:        101,
		Name:      "alpha",
		GroupID:   "QQ-Group:1",
		CreatedAt: time.Now().Add(-48 * time.Hour).Unix(),
		UpdatedAt: time.Now().Add(-24 * time.Hour).Unix(),
	}
	insertStoryLogFixture(t, svc, logInfo, []model.LogOneItem{
		{ID: 1001, GroupID: logInfo.GroupID, Nickname: "Alice", IMUserID: "u1", Time: logInfo.CreatedAt, Message: "first"},
		{ID: 1002, GroupID: logInfo.GroupID, Nickname: "Bob", IMUserID: "u2", Time: logInfo.CreatedAt + 1, Message: "second"},
	})

	respByID, err := svc.GetItemPage(context.Background(), &storym.ItemPageQuery{
		LogID:    logInfo.ID,
		PageNum:  1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("GetItemPage by id returned error: %v", err)
	}
	if len(respByID.Body.Item) != 2 {
		t.Fatalf("GetItemPage by id returned %d items, want 2", len(respByID.Body.Item))
	}

	respByLegacy, err := svc.GetItemPage(context.Background(), &storym.ItemPageQuery{
		GroupID:  logInfo.GroupID,
		LogName:  logInfo.Name,
		PageNum:  1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("GetItemPage by legacy query returned error: %v", err)
	}
	if len(respByLegacy.Body.Item) != 2 {
		t.Fatalf("GetItemPage by legacy query returned %d items, want 2", len(respByLegacy.Body.Item))
	}
}

func TestDeleteLogDeletesByIDOnly(t *testing.T) {
	svc := newTestStoryService(t)
	logInfo := model.LogInfo{
		ID:        201,
		Name:      "beta",
		GroupID:   "QQ-Group:2",
		CreatedAt: time.Now().Add(-48 * time.Hour).Unix(),
		UpdatedAt: time.Now().Add(-24 * time.Hour).Unix(),
	}
	insertStoryLogFixture(t, svc, logInfo, []model.LogOneItem{
		{ID: 2001, GroupID: logInfo.GroupID, Nickname: "Alice", IMUserID: "u1", Time: logInfo.CreatedAt, Message: "first"},
	})

	resp, err := svc.DeleteLog(context.Background(), &storym.DeleteLogReq{
		Body: request.RequestWrapper[storym.DeleteLogReqBody]{
			Body: storym.DeleteLogReqBody{ID: logInfo.ID},
		},
	})
	if err != nil {
		t.Fatalf("DeleteLog returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatal("DeleteLog success = false, want true")
	}

	db := svc.dice.DBOperator.GetLogDB(constant.READ)
	var logsCount int64
	if err := db.Model(&model.LogInfo{}).Where("id = ?", logInfo.ID).Count(&logsCount).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logsCount != 0 {
		t.Fatalf("remaining logs count = %d, want 0", logsCount)
	}

	var itemsCount int64
	if err := db.Model(&model.LogOneItem{}).Where("log_id = ?", logInfo.ID).Count(&itemsCount).Error; err != nil {
		t.Fatalf("count log items: %v", err)
	}
	if itemsCount != 0 {
		t.Fatalf("remaining log items count = %d, want 0", itemsCount)
	}
}

func TestGetLogPageIncludesLinkState(t *testing.T) {
	svc := newTestStoryService(t)
	now := time.Now().Unix()
	insertStoryLogFixture(t, svc, model.LogInfo{
		ID:         301,
		Name:       "none",
		GroupID:    "QQ-Group:3",
		CreatedAt:  now - 300,
		UpdatedAt:  now - 200,
		UploadURL:  "",
		UploadTime: 0,
	}, nil)
	insertStoryLogFixture(t, svc, model.LogInfo{
		ID:         302,
		Name:       "fresh",
		GroupID:    "QQ-Group:3",
		CreatedAt:  now - 300,
		UpdatedAt:  now - 200,
		UploadURL:  "https://example.com/fresh",
		UploadTime: int(now - 100),
	}, nil)
	insertStoryLogFixture(t, svc, model.LogInfo{
		ID:         303,
		Name:       "stale",
		GroupID:    "QQ-Group:3",
		CreatedAt:  now - 300,
		UpdatedAt:  now - 100,
		UploadURL:  "https://example.com/stale",
		UploadTime: int(now - 200),
	}, nil)

	resp, err := svc.GetLogPage(context.Background(), &storym.LogPageQuery{
		PageNum:  1,
		PageSize: 10,
		GroupID:  "QQ-Group:3",
	})
	if err != nil {
		t.Fatalf("GetLogPage returned error: %v", err)
	}
	if len(resp.Body.Item.Data) != 3 {
		t.Fatalf("GetLogPage returned %d items, want 3", len(resp.Body.Item.Data))
	}

	got := map[uint64]string{}
	for _, item := range resp.Body.Item.Data {
		got[item.ID] = item.LinkState
	}
	if got[301] != "none" || got[302] != "fresh" || got[303] != "stale" {
		t.Fatalf("unexpected link states: %#v", got)
	}
}

func TestCleanupPreviewAndExecute(t *testing.T) {
	svc := newTestStoryService(t)
	now := time.Now()
	oldUpdatedAt := now.AddDate(0, -3, 0).Unix()
	newUpdatedAt := now.AddDate(0, -1, 0).Unix()

	insertStoryLogFixture(t, svc, model.LogInfo{
		ID:        401,
		Name:      "old-log",
		GroupID:   "QQ-Group:4",
		CreatedAt: oldUpdatedAt - 3600,
		UpdatedAt: oldUpdatedAt,
	}, []model.LogOneItem{
		{ID: 4001, GroupID: "QQ-Group:4", Nickname: "Old", IMUserID: "u-old", Time: oldUpdatedAt, Message: "old-1"},
		{ID: 4002, GroupID: "QQ-Group:4", Nickname: "Old", IMUserID: "u-old", Time: oldUpdatedAt + 1, Message: "old-2"},
	})
	insertStoryLogFixture(t, svc, model.LogInfo{
		ID:        402,
		Name:      "new-log",
		GroupID:   "QQ-Group:4",
		CreatedAt: newUpdatedAt - 3600,
		UpdatedAt: newUpdatedAt,
	}, []model.LogOneItem{
		{ID: 4003, GroupID: "QQ-Group:4", Nickname: "New", IMUserID: "u-new", Time: newUpdatedAt, Message: "new"},
	})

	preview, err := svc.PreviewCleanup(context.Background(), &storym.CleanupPreviewQuery{Months: 2})
	if err != nil {
		t.Fatalf("PreviewCleanup returned error: %v", err)
	}
	if preview.Body.Item.Logs != 1 {
		t.Fatalf("PreviewCleanup logs = %d, want 1", preview.Body.Item.Logs)
	}
	if preview.Body.Item.Items != 2 {
		t.Fatalf("PreviewCleanup items = %d, want 2", preview.Body.Item.Items)
	}

	execResp, err := svc.Cleanup(context.Background(), &storym.CleanupReq{
		Body: request.RequestWrapper[storym.CleanupReqBody]{
			Body: storym.CleanupReqBody{Months: 2, Vacuum: false},
		},
	})
	if err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if execResp.Body.Item.Logs != 1 || execResp.Body.Item.Items != 2 {
		t.Fatalf("Cleanup result = %#v, want 1 log and 2 items", execResp.Body.Item)
	}

	db := svc.dice.DBOperator.GetLogDB(constant.READ)
	var count int64
	if err := db.Model(&model.LogInfo{}).Where("id = ?", 401).Count(&count).Error; err != nil {
		t.Fatalf("count old logs: %v", err)
	}
	if count != 0 {
		t.Fatalf("old log count = %d, want 0", count)
	}
	if err := db.Model(&model.LogInfo{}).Where("id = ?", 402).Count(&count).Error; err != nil {
		t.Fatalf("count new logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("new log count = %d, want 1", count)
	}
}

func TestUploadLogRejectsMissingID(t *testing.T) {
	svc := newTestStoryService(t)
	_, err := svc.UploadLog(context.Background(), &storym.UploadLogReq{
		Body: request.RequestWrapper[storym.UploadLogReqBody]{
			Body: storym.UploadLogReqBody{ID: 0, Force: false},
		},
	})
	if err == nil {
		t.Fatal("UploadLog unexpectedly accepted empty id")
	}
}

type _ interface {
	engine.DatabaseOperator
}
