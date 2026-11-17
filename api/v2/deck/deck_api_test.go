package deck_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	. "sealdice-core/api/v2/deck"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	deckDir := filepath.Join("data", "decks")
	if err := os.MkdirAll(deckDir, 0o755); err != nil {
		t.Fatalf("mkdir deck dir: %v", err)
	}
	t.Cleanup(func() {
		entries, err := os.ReadDir(deckDir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "codex-v2-deck-") {
				_ = os.RemoveAll(filepath.Join(deckDir, name))
			}
		}
	})

	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "test"
	d.BaseConfig.DataDir = "."
	d.Config = dice.NewConfig(d)
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{
		Dice: d,
	}
	d.Parent = dm
	return NewService(dm)
}

func writeDeckFile(t *testing.T, d *dice.Dice, name string, content string) {
	t.Helper()
	fp := filepath.Join("data", "decks", name)
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatalf("write deck file: %v", err)
	}
	dice.DeckReload(d)
}

func TestGetListSupportsKeywordAndPagination(t *testing.T) {
	svc := newTestService(t)
	writeDeckFile(t, svc.Dice(), "codex-v2-deck-alpha.json", `{"_title":["Alpha"],"_author":["Alice"],"atk":["1"]}`)
	writeDeckFile(t, svc.Dice(), "codex-v2-deck-beta.json", `{"_title":["Beta"],"_author":["Bob"],"heal":["2"]}`)

	resp, err := svc.GetList(t.Context(), &ListQuery{
		Page:      1,
		PageSize:  1,
		Keyword:   "alp",
		SortBy:    "name",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("GetList returned error: %v", err)
	}
	if resp.Body.Item.Total != 1 {
		t.Fatalf("Total = %d, want 1", resp.Body.Item.Total)
	}
	if len(resp.Body.Item.List) != 1 {
		t.Fatalf("list length = %d, want 1", len(resp.Body.Item.List))
	}
	if resp.Body.Item.List[0].Name != "Alpha" {
		t.Fatalf("first item name = %q, want Alpha", resp.Body.Item.List[0].Name)
	}
}

func TestReloadReturnsTestModeWhenJustForTest(t *testing.T) {
	svc := newTestService(t)
	svc.Dice().Parent.JustForTest = true

	resp, err := svc.Reload(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if !resp.Body.Item.TestMode {
		t.Fatalf("TestMode = false, want true")
	}
}

func TestUploadWritesDeckFile(t *testing.T) {
	svc := newTestService(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "sample.json")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, writeErr := part.Write([]byte(`{"_title":["Sample"],"draw":["1"]}`)); writeErr != nil {
		t.Fatalf("write multipart payload: %v", writeErr)
	}
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("writer.Close: %v", closeErr)
	}

	req, err := http.NewRequest(http.MethodPost, "/upload", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if parseErr := req.ParseMultipartForm(1024 * 1024); parseErr != nil {
		t.Fatalf("ParseMultipartForm: %v", parseErr)
	}

	raw := huma.MultipartFormFiles[UploadForm]{}
	raw.Form = req.MultipartForm
	resp, err := svc.Upload(t.Context(), &UploadReq{RawBody: raw})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("Upload success = false, want true")
	}
	if _, err := os.Stat(filepath.Join("data", "decks", "sample.json")); err != nil {
		t.Fatalf("uploaded file not found: %v", err)
	}
	_ = os.Remove(filepath.Join("data", "decks", "sample.json"))
}

func TestDeleteRemovesDeck(t *testing.T) {
	svc := newTestService(t)
	writeDeckFile(t, svc.Dice(), "codex-v2-deck-sample.json", `{"_title":["Sample"],"draw":["1"]}`)

	resp, err := svc.Delete(t.Context(), &FilenameReq{
		Body: FilenameReqBody{Filename: filepath.Join("data", "decks", "codex-v2-deck-sample.json")},
	})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("Delete success = false, want true")
	}
}

func TestCheckUpdateReturnsFailureForMissingDeck(t *testing.T) {
	svc := newTestService(t)

	resp, err := svc.CheckUpdate(t.Context(), &FilenameReq{
		Body: FilenameReqBody{Filename: "missing.json"},
	})
	if err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if resp.Body.Item.Success {
		t.Fatalf("CheckUpdate success = true, want false")
	}
}

func TestFromDeckInfoCarriesPackageID(t *testing.T) {
	item := FromDeckInfo(&dice.DeckInfo{
		Name:      "Pkg Deck",
		Filename:  "pkg.deck",
		PackageID: "author/pkg",
	})

	if item.PackageID != "author/pkg" {
		t.Fatalf("PackageID = %q, want author/pkg", item.PackageID)
	}
}
