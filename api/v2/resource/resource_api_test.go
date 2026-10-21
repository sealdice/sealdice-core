package resource_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "sealdice-core/api/v2/resource"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	tempDir := t.TempDir()
	t.Chdir(tempDir)
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
		Dice: d,
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
