package resource_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	. "sealdice-core/api/v2/resource"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

func newTestService(t *testing.T) *Service {
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
	if err := os.MkdirAll(filepath.Join("data", "images"), 0o755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}

	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "resource-test"
	d.BaseConfig.DataDir = tempDir
	d.Config = dice.NewConfig(d)
	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm
	return NewService(dm)
}

func writeResourceFile(t *testing.T, name string, size int) {
	t.Helper()

	data := bytes.Repeat([]byte("x"), size)
	path := filepath.Join("data", "images", name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write resource file: %v", err)
	}
}

func TestGetListSupportsKeywordSortingAndPagination(t *testing.T) {
	svc := newTestService(t)
	writeResourceFile(t, "beta.gif", 20)
	writeResourceFile(t, "alpha.png", 10)
	writeResourceFile(t, "ignored.txt", 100)

	firstResp, err := svc.GetList(t.Context(), &ListQuery{
		Page:      1,
		PageSize:  1,
		Type:      "image",
		Keyword:   "a",
		SortBy:    "name",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("GetList first page returned error: %v", err)
	}
	if firstResp.Body.Item.Total != 2 {
		t.Fatalf("Total = %d, want 2", firstResp.Body.Item.Total)
	}
	if len(firstResp.Body.Item.List) != 1 {
		t.Fatalf("first page length = %d, want 1", len(firstResp.Body.Item.List))
	}
	if firstResp.Body.Item.List[0].Name != "alpha.png" {
		t.Fatalf("first item = %q, want alpha.png", firstResp.Body.Item.List[0].Name)
	}

	secondResp, err := svc.GetList(t.Context(), &ListQuery{
		Page:      2,
		PageSize:  1,
		Type:      "image",
		Keyword:   "a",
		SortBy:    "name",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("GetList second page returned error: %v", err)
	}
	if secondResp.Body.Item.Page != 2 {
		t.Fatalf("second page = %d, want 2", secondResp.Body.Item.Page)
	}
	if secondResp.Body.Item.List[0].Name != "beta.gif" {
		t.Fatalf("second item = %q, want beta.gif", secondResp.Body.Item.List[0].Name)
	}
}

func TestUploadWritesAllowedImagesAndSanitizesNames(t *testing.T) {
	svc := newTestService(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("files", "nested\\seal.PNG")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err = part.Write([]byte("image data")); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/upload", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err = req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}

	raw := huma.MultipartFormFiles[UploadForm]{}
	raw.Form = req.MultipartForm
	resp, err := svc.Upload(t.Context(), &UploadReq{RawBody: raw})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatal("Upload success = false, want true")
	}
	if _, err = os.Stat(filepath.Join("data", "images", "nested_seal.PNG")); err != nil {
		t.Fatalf("uploaded image not found: %v", err)
	}
}

func TestUploadRejectsUnsupportedFiles(t *testing.T) {
	svc := newTestService(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("files", "bad.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err = part.Write([]byte("not image")); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/upload", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err = req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}

	raw := huma.MultipartFormFiles[UploadForm]{}
	raw.Form = req.MultipartForm
	if _, err = svc.Upload(t.Context(), &UploadReq{RawBody: raw}); err == nil {
		t.Fatal("Upload accepted unsupported file, want error")
	}
}

func TestDeleteRemovesImageAndRejectsTraversal(t *testing.T) {
	svc := newTestService(t)
	writeResourceFile(t, "to-delete.jpg", 12)

	resp, err := svc.Delete(t.Context(), &DeleteReq{
		Body: ResourcePathReqBody{Path: "data/images/to-delete.jpg"},
	})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatal("Delete success = false, want true")
	}
	if _, err = os.Stat(filepath.Join("data", "images", "to-delete.jpg")); !os.IsNotExist(err) {
		t.Fatalf("deleted file stat err = %v, want not exists", err)
	}

	if _, err = svc.Delete(t.Context(), &DeleteReq{
		Body: ResourcePathReqBody{Path: "../outside.png"},
	}); err == nil {
		t.Fatal("Delete accepted traversal path, want error")
	}
}
