package customreply

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	customreplym "sealdice-core/api/v2/model/customreply"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	dataDir := t.TempDir()
	replyDir := filepath.Join(dataDir, "extensions", "reply")
	if err := os.MkdirAll(replyDir, 0o755); err != nil {
		t.Fatalf("mkdir reply dir: %v", err)
	}

	d := &dice.Dice{
		Logger: logger.M(),
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
	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm
	dice.CustomReplyConfigNew(d, "reply.yaml")
	dice.ReplyReload(d)
	return NewService(dm)
}

func writeReplyFile(t *testing.T, d *dice.Dice, filename string, content string) {
	t.Helper()
	fp := d.GetExtConfigFilePath("reply", filename)
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		t.Fatalf("mkdir reply file dir: %v", err)
	}
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatalf("write reply file: %v", err)
	}
	dice.ReplyReload(d)
}

func TestGetFileListReturnsReplyFiles(t *testing.T) {
	svc := newTestService(t)
	writeReplyFile(t, svc.dice, "reply2.yaml", "enable: false\nitems: []\nconditions: []\n")

	resp, err := svc.GetFileList(context.Background(), &customreplym.FileListQuery{})
	if err != nil {
		t.Fatalf("GetFileList returned error: %v", err)
	}
	if len(resp.Body.Item.List) < 2 {
		t.Fatalf("file list length = %d, want at least 2", len(resp.Body.Item.List))
	}
}

func TestGetFileListSupportsKeywordSortAndPagination(t *testing.T) {
	svc := newTestService(t)
	writeReplyFile(t, svc.dice, "bbb.yaml", "enable: true\nupdateTimestamp: 10\ncreateTimestamp: 9\nitems: []\nconditions: []\n")
	writeReplyFile(t, svc.dice, "aaa.yaml", "enable: true\nupdateTimestamp: 20\ncreateTimestamp: 19\nitems: []\nconditions: []\n")

	resp, err := svc.GetFileList(context.Background(), &customreplym.FileListQuery{
		Page:      1,
		PageSize:  1,
		Keyword:   "a",
		SortBy:    "name",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("GetFileList returned error: %v", err)
	}
	if resp.Body.Item.Total < 1 {
		t.Fatalf("Total = %d, want at least 1", resp.Body.Item.Total)
	}
	if len(resp.Body.Item.List) != 1 {
		t.Fatalf("page size result length = %d, want 1", len(resp.Body.Item.List))
	}
	if resp.Body.Item.List[0].Filename != "aaa.yaml" {
		t.Fatalf("first item filename = %q, want aaa.yaml", resp.Body.Item.List[0].Filename)
	}
}

func TestGetConfigReturnsCurrentReplyConfig(t *testing.T) {
	svc := newTestService(t)
	writeReplyFile(t, svc.dice, "reply.yaml", "enable: true\ninterval: 7\ncreateTimestamp: 11\nupdateTimestamp: 12\nitems: []\nconditions: []\n")

	resp, err := svc.GetConfig(context.Background(), &customreplym.FilenamePath{Filename: "reply.yaml"})
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if resp.Body.Item.Interval != 7 {
		t.Fatalf("Interval = %v, want 7", resp.Body.Item.Interval)
	}
	if resp.Body.Item.Filename != "reply.yaml" {
		t.Fatalf("Filename = %q, want reply.yaml", resp.Body.Item.Filename)
	}
	if resp.Body.Item.ItemCount != 0 {
		t.Fatalf("ItemCount = %d, want 0", resp.Body.Item.ItemCount)
	}
	if resp.Body.Item.UpdateTimestamp != 12 {
		t.Fatalf("UpdateTimestamp = %d, want 12", resp.Body.Item.UpdateTimestamp)
	}
}

func TestGetRulesReturnsPagedRuleItems(t *testing.T) {
	svc := newTestService(t)
	writeReplyFile(t, svc.dice, "reply.yaml", `enable: true
conditions: []
items:
  - enable: true
    conditions:
      - condType: textMatch
        matchType: matchExact
        value: a
    results:
      - resultType: replyToSender
        delay: 0
        message:
          - ["one", 1]
  - enable: true
    conditions:
      - condType: textMatch
        matchType: matchExact
        value: b
    results:
      - resultType: replyToSender
        delay: 0
        message:
          - ["two", 1]
`)

	resp, err := svc.GetRules(context.Background(), &customreplym.RulePageQuery{
		Filename: "reply.yaml",
		Page:     2,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("GetRules returned error: %v", err)
	}
	if resp.Body.Item.Total != 2 {
		t.Fatalf("Total = %d, want 2", resp.Body.Item.Total)
	}
	if len(resp.Body.Item.List) != 1 {
		t.Fatalf("rules page length = %d, want 1", len(resp.Body.Item.List))
	}
	if resp.Body.Item.List[0].Index != 1 {
		t.Fatalf("rule index = %d, want 1", resp.Body.Item.List[0].Index)
	}
}

func TestGetConditionsReturnsPagedConditions(t *testing.T) {
	svc := newTestService(t)
	writeReplyFile(t, svc.dice, "reply.yaml", `enable: true
conditions:
  - condType: textMatch
    matchType: matchExact
    value: first
  - condType: exprTrue
    value: $t1 == 'second'
items: []
`)

	resp, err := svc.GetConditions(context.Background(), &customreplym.ConditionPageQuery{
		Filename: "reply.yaml",
		Page:     2,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("GetConditions returned error: %v", err)
	}
	if resp.Body.Item.Total != 2 {
		t.Fatalf("Total = %d, want 2", resp.Body.Item.Total)
	}
	if len(resp.Body.Item.List) != 1 {
		t.Fatalf("conditions page length = %d, want 1", len(resp.Body.Item.List))
	}
	if resp.Body.Item.List[0].Index != 1 {
		t.Fatalf("condition index = %d, want 1", resp.Body.Item.List[0].Index)
	}
}

func TestSaveConfigWritesReplyFile(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.SaveConfig(context.Background(), &customreplym.SaveReq{
		Filename: "reply.yaml",
		Body: request.RequestWrapper[dice.ReplyConfig]{
			Body: dice.ReplyConfig{
				Enable:   true,
				Interval: 6,
				Items: []*dice.ReplyItem{
					{
						Enable: true,
						Conditions: dice.ReplyConditions{
							&dice.ReplyConditionTextLenLimit{
								CondType: "textLenLimit",
								MatchOp:  "ge",
								Value:    8,
							},
						},
						Results: []dice.ReplyResultBase{
							&dice.ReplyResultReplyToSender{
								ResultType: "replyToSender",
								Delay:      0,
								Message:    dice.TextTemplateItemList{{"你好", 1}},
							},
						},
					},
				},
				Conditions: dice.ReplyConditions{},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	resp, err := svc.GetConfig(context.Background(), &customreplym.FilenamePath{Filename: "reply.yaml"})
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if resp.Body.Item.Filename != "reply.yaml" {
		t.Fatalf("Filename = %q, want reply.yaml", resp.Body.Item.Filename)
	}
	if resp.Body.Item.ItemCount != 1 {
		t.Fatalf("item count = %d, want 1", resp.Body.Item.ItemCount)
	}
	if resp.Body.Item.UpdateTimestamp == 0 {
		t.Fatalf("UpdateTimestamp should be refreshed")
	}
}

func TestCreateFileRejectsExistingName(t *testing.T) {
	svc := newTestService(t)

	resp, err := svc.CreateFile(context.Background(), &customreplym.FileReq{
		Body: request.RequestWrapper[customreplym.FileBody]{
			Body: customreplym.FileBody{Filename: "reply.yaml"},
		},
	})
	if err != nil {
		t.Fatalf("CreateFile returned error: %v", err)
	}
	if resp.Body.Item.Success {
		t.Fatalf("CreateFile should fail for existing file")
	}
}

func TestCreateAndDeleteFile(t *testing.T) {
	svc := newTestService(t)

	createResp, err := svc.CreateFile(context.Background(), &customreplym.FileReq{
		Body: request.RequestWrapper[customreplym.FileBody]{
			Body: customreplym.FileBody{Filename: "reply2.yaml"},
		},
	})
	if err != nil {
		t.Fatalf("CreateFile returned error: %v", err)
	}
	if !createResp.Body.Item.Success {
		t.Fatalf("CreateFile should succeed")
	}

	deleteResp, err := svc.DeleteFile(context.Background(), &customreplym.FilenamePath{Filename: "reply2.yaml"})
	if err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}
	if !deleteResp.Body.Item.Success {
		t.Fatalf("DeleteFile should succeed")
	}
}

func TestDownloadRejectsInvalidName(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Download(context.Background(), &customreplym.FilenamePath{Filename: "../bad.yaml"}); err == nil {
		t.Fatalf("expected invalid filename error")
	}
}

func TestDownloadReturnsStreamForExistingFile(t *testing.T) {
	svc := newTestService(t)
	writeReplyFile(t, svc.dice, "reply.yaml", "enable: true\nitems: []\nconditions: []\n")

	resp, err := svc.Download(context.Background(), &customreplym.FilenamePath{Filename: "reply.yaml"})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if resp == nil || resp.Body == nil {
		t.Fatalf("expected stream response body")
	}
}

func TestDebugModeRoundTrip(t *testing.T) {
	svc := newTestService(t)

	getResp, err := svc.GetDebugMode(context.Background(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetDebugMode returned error: %v", err)
	}
	if getResp.Body.Item.Value {
		t.Fatalf("default debug mode should be false")
	}

	setResp, err := svc.SetDebugMode(context.Background(), &customreplym.DebugModeReq{
		Body: request.RequestWrapper[customreplym.DebugModeResp]{
			Body: customreplym.DebugModeResp{Value: true},
		},
	})
	if err != nil {
		t.Fatalf("SetDebugMode returned error: %v", err)
	}
	if !setResp.Body.Item.Value {
		t.Fatalf("debug mode should be true")
	}
}

func TestUploadFileSavesMultipartPayload(t *testing.T) {
	svc := newTestService(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "reply3.yaml")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("enable: true\nitems: []\nconditions: []\n")); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/upload", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}

	raw := huma.MultipartFormFiles[customreplym.UploadForm]{}
	raw.Form = req.MultipartForm

	_, err = svc.Upload(context.Background(), &customreplym.UploadReq{RawBody: raw})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	itemsResp, err := svc.GetFileList(context.Background(), &customreplym.FileListQuery{})
	if err != nil {
		t.Fatalf("GetFileList returned error: %v", err)
	}
	found := false
	for _, item := range itemsResp.Body.Item.List {
		if item.Filename == "reply3.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("uploaded file not found in file list")
	}
}
