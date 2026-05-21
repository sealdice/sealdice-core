package ban

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/robfig/cron/v3"

	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/model/common/request"
	"sealdice-core/utils/constant"
	sqliteengine "sealdice-core/utils/dboperator/engine/sqlite"
)

func newTestBanService(t *testing.T) *BanService {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q): %v", tempDir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := os.MkdirAll(filepath.Join("data"), 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}

	operator := &sqliteengine.SQLiteEngine{}
	if err := operator.Init(context.Background()); err != nil {
		t.Fatalf("init sqlite operator: %v", err)
	}
	t.Cleanup(operator.Close)
	if err := operator.GetDataDB(constant.WRITE).AutoMigrate(&model.BanInfo{}); err != nil {
		t.Fatalf("migrate ban table: %v", err)
	}

	d := &dice.Dice{
		Logger:     logger.M(),
		DBOperator: operator,
	}
	d.BaseConfig.Name = "ban-test"
	d.BaseConfig.DataDir = tempDir

	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
		Cron: cron.New(),
	}
	d.Parent = dm
	d.Config = dice.NewConfig(d)
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}

	adapter := &dice.PlatformAdapterHTTP{Session: d.ImSession}
	ep := &dice.EndPointInfo{
		EndPointInfoBase: dice.EndPointInfoBase{
			ID:       "ui-endpoint",
			Nickname: "SealDice",
			UserID:   "UI:1000",
			Platform: "UI",
			Enable:   true,
			Session:  d.ImSession,
		},
		Adapter: adapter,
	}
	adapter.EndPoint = ep
	d.ImSession.EndPoints = []*dice.EndPointInfo{ep}
	d.UIEndpoint = ep

	return NewBanService(dm)
}

func TestGetAndSetConfigRoundTrip(t *testing.T) {
	svc := newTestBanService(t)

	getResp, err := svc.GetConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if !getResp.Body.Item.BanBehaviorRefuseReply {
		t.Fatalf("default BanBehaviorRefuseReply = false, want true")
	}

	_, err = svc.SetConfig(t.Context(), &ConfigReq{
		Body: BanConfig{
			BanBehaviorRefuseReply:                 false,
			BanBehaviorRefuseInvite:                false,
			BanBehaviorQuitLastPlace:               true,
			BanBehaviorQuitPlaceImmediately:        true,
			BanBehaviorQuitIfAdmin:                 true,
			BanBehaviorQuitIfAdminSilentIfNotAdmin: true,
			ThresholdWarn:                          88,
			ThresholdBan:                           188,
			AutoBanMinutes:                         99,
			ScoreReducePerMinute:                   2,
			ScoreGroupMuted:                        33,
			ScoreGroupKicked:                       66,
			ScoreTooManyCommand:                    77,
			JointScorePercentOfGroup:               0.6,
			JointScorePercentOfInviter:             0.4,
		},
	})
	if err != nil {
		t.Fatalf("SetConfig returned error: %v", err)
	}

	nextResp, err := svc.GetConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetConfig after set returned error: %v", err)
	}
	if nextResp.Body.Item.ThresholdWarn != 88 {
		t.Fatalf("ThresholdWarn = %d, want 88", nextResp.Body.Item.ThresholdWarn)
	}
	if nextResp.Body.Item.ScoreTooManyCommand != 77 {
		t.Fatalf("ScoreTooManyCommand = %d, want 77", nextResp.Body.Item.ScoreTooManyCommand)
	}
	if !nextResp.Body.Item.BanBehaviorQuitPlaceImmediately {
		t.Fatal("BanBehaviorQuitPlaceImmediately = false, want true")
	}
}

func TestAddEntryStoresBannedAndTrustedItems(t *testing.T) {
	svc := newTestBanService(t)

	_, err := svc.AddEntry(t.Context(), &AddReq{
		Body: AddReqBody{
			ID:     "UI:1001",
			Rank:   int(dice.BanRankBanned),
			Name:   "测试用户",
			Reason: "人工拉黑",
		},
	})
	if err != nil {
		t.Fatalf("AddEntry(banned) returned error: %v", err)
	}

	_, err = svc.AddEntry(t.Context(), &AddReq{
		Body: AddReqBody{
			ID:     "UI-Group:2001",
			Rank:   int(dice.BanRankTrusted),
			Name:   "信任群",
			Reason: "人工信任",
		},
	})
	if err != nil {
		t.Fatalf("AddEntry(trusted) returned error: %v", err)
	}

	listResp, err := svc.GetBanPage(t.Context(), &BanPageReq{
		Body: BanPageRequest{
			PageInfo: request.PageInfo{Page: 1, PageSize: 20},
		},
	})
	if err != nil {
		t.Fatalf("GetBanPage returned error: %v", err)
	}
	if listResp.Body.Item.Total != 2 {
		t.Fatalf("ban total = %d, want 2", listResp.Body.Item.Total)
	}

	items := listResp.Body.Item.List
	foundBanned := false
	foundTrusted := false
	for _, item := range items {
		if item.ID == "UI:1001" && item.Rank == dice.BanRankBanned && item.Name == "测试用户" {
			foundBanned = true
		}
		if item.ID == "UI-Group:2001" && item.Rank == dice.BanRankTrusted {
			foundTrusted = true
		}
	}
	if !foundBanned || !foundTrusted {
		t.Fatalf("stored items mismatch: %#v", items)
	}
}

func TestImportAddsSuffixAndPersistsItems(t *testing.T) {
	svc := newTestBanService(t)

	payload, err := json.Marshal([]*dice.BanListInfoItem{
		{
			ID:      "UI:3001",
			Name:    "导入用户",
			Rank:    dice.BanRankWarn,
			Score:   120,
			Reasons: []string{"旧原因"},
			Places:  []string{"UI"},
		},
	})
	if err != nil {
		t.Fatalf("marshal import payload: %v", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "ban.json")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/import", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}

	raw := huma.MultipartFormFiles[ImportForm]{}
	raw.Form = req.MultipartForm

	_, err = svc.Import(t.Context(), &ImportReq{
		RawBody: raw,
	})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}

	listResp, err := svc.GetBanPage(t.Context(), &BanPageReq{
		Body: BanPageRequest{
			PageInfo: request.PageInfo{Page: 1, PageSize: 20},
		},
	})
	if err != nil {
		t.Fatalf("GetBanPage after import returned error: %v", err)
	}
	if listResp.Body.Item.Total != 1 {
		t.Fatalf("ban total after import = %d, want 1", listResp.Body.Item.Total)
	}
	item := listResp.Body.Item.List[0]
	if len(item.Reasons) != 1 || item.Reasons[0] != "旧原因（来自导入）" {
		t.Fatalf("imported reasons = %#v, want suffixed reason", item.Reasons)
	}
}
