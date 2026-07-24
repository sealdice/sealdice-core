package docengine //nolint:testpackage // Tests need access to unexported index helpers.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

func newTestBlugeSearchEngine(t *testing.T) *BlugeSearchEngine {
	t.Helper()

	oldIndexDir := indexDir
	indexDir = filepath.Join(t.TempDir(), "index")
	t.Cleanup(func() {
		indexDir = oldIndexDir
	})

	engine, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatalf("NewBlugeSearchEngine() error = %v", err)
	}
	t.Cleanup(func() {
		engine.Close()
	})

	return engine
}

func addTestHelpItems(t *testing.T, engine *BlugeSearchEngine, count int, group, from, titlePrefix string) {
	t.Helper()

	for i := range count {
		if _, err := engine.AddItem(HelpTextItem{
			Group:       group,
			From:        from,
			Title:       titlePrefix + strconv.Itoa(i),
			Content:     "content",
			PackageName: "pkg",
		}); err != nil {
			t.Fatalf("AddItem() error = %v", err)
		}
	}
}

func TestBlugeSearchEngineDeleteByFromRemovesAllMatches(t *testing.T) {
	engine := newTestBlugeSearchEngine(t)

	const targetCount = deleteSearchBatchSize + 137
	const keepCount = 17
	const targetFrom = "target from/with spaces"

	addTestHelpItems(t, engine, targetCount, "group-a", targetFrom, "target-")
	addTestHelpItems(t, engine, keepCount, "group-b", "keepfrom", "keep-")

	if err := engine.AddItemApply(true); err != nil {
		t.Fatalf("AddItemApply(true) error = %v", err)
	}

	idsBefore, err := engine.ListAllDocumentIDs()
	if err != nil {
		t.Fatalf("ListAllDocumentIDs() before delete error = %v", err)
	}
	if got, want := len(idsBefore), targetCount+keepCount; got != want {
		t.Fatalf("document count before delete = %d, want %d", got, want)
	}

	if deleteErr := engine.DeleteByFrom(targetFrom); deleteErr != nil {
		t.Fatalf("DeleteByFrom() error = %v", deleteErr)
	}

	idsAfter, err := engine.ListAllDocumentIDs()
	if err != nil {
		t.Fatalf("ListAllDocumentIDs() after delete error = %v", err)
	}
	if got, want := len(idsAfter), keepCount; got != want {
		t.Fatalf("document count after delete = %d, want %d", got, want)
	}

	for _, id := range idsAfter {
		item, getErr := engine.GetItemByInternalID(id)
		if getErr != nil {
			t.Fatalf("GetItemByInternalID(%q) error = %v", id, getErr)
		}
		if item.From == targetFrom {
			t.Fatalf("document %q from deleted source still exists", id)
		}
	}
}

func TestBlugeSearchEnginePaginateDocumentsIncludesInternalIDs(t *testing.T) {
	engine := newTestBlugeSearchEngine(t)
	addTestHelpItems(t, engine, 2, "group", "from", "item-")
	if err := engine.AddItemApply(true); err != nil {
		t.Fatal(err)
	}

	ids, err := engine.ListAllDocumentIDs()
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := make(map[string]bool, len(ids))
	for _, id := range ids {
		wantIDs[id] = true
	}

	total, items, err := engine.PaginateDocuments(10, 1, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("PaginateDocuments() total/items = %d/%d, want 2/2", total, len(items))
	}
	for _, item := range items {
		if !wantIDs[item.InternalID] {
			t.Fatalf("PaginateDocuments() returned unexpected internal ID %q", item.InternalID)
		}
	}
}

func TestBlugeSearchEngineNumericIDsRebuildLazily(t *testing.T) {
	engine := newTestBlugeSearchEngine(t)

	addTestHelpItems(t, engine, 1, "remove-me", "from-a", "alpha-")
	addTestHelpItems(t, engine, 2, "keep-me", "from-b", "beta-")

	if err := engine.AddItemApply(true); err != nil {
		t.Fatalf("AddItemApply(true) error = %v", err)
	}

	idsBefore, err := engine.ListAllDocumentIDs()
	if err != nil {
		t.Fatalf("ListAllDocumentIDs() before delete error = %v", err)
	}
	if !sort.StringsAreSorted(idsBefore) {
		t.Fatalf("document IDs are not sorted: %v", idsBefore)
	}

	if deleteErr := engine.DeleteByGroup("remove-me"); deleteErr != nil {
		t.Fatalf("DeleteByGroup() error = %v", deleteErr)
	}

	idsAfter, err := engine.ListAllDocumentIDs()
	if err != nil {
		t.Fatalf("ListAllDocumentIDs() after delete error = %v", err)
	}
	if got, want := len(idsAfter), 2; got != want {
		t.Fatalf("document count after DeleteByGroup = %d, want %d", got, want)
	}

	res, total, _, _, err := engine.Search([]string{"pkg"}, "beta-1", true, 10, 1, "")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if total != 1 || len(res.Hits) != 1 {
		t.Fatalf("Search() total/hits = %d/%d, want 1/1", total, len(res.Hits))
	}
	if _, err := strconv.Atoi(res.Hits[0].ID); err != nil {
		t.Fatalf("Search() returned non-numeric ID %q", res.Hits[0].ID)
	}
}

func TestBlugeSearchEngineGroupFilterIsCaseInsensitive(t *testing.T) {
	engine := newTestBlugeSearchEngine(t)

	for _, item := range []HelpTextItem{
		{Group: "COC", From: "coc.json", Title: "法术", Content: "克苏鲁法术", PackageName: "coc"},
		{Group: "DND", From: "dnd.xlsx", Title: "法术", Content: "龙与地下城法术", PackageName: "dnd"},
	} {
		if _, err := engine.AddItem(item); err != nil {
			t.Fatal(err)
		}
	}
	if err := engine.AddItemApply(true); err != nil {
		t.Fatal(err)
	}

	for _, group := range []string{"coc", "COC", "CoC"} {
		result, total, _, _, err := engine.Search(nil, "法术", false, 10, 1, group)
		if err != nil {
			t.Fatalf("Search(group=%q) error = %v", group, err)
		}
		if total != 1 || len(result.Hits) != 1 {
			t.Fatalf("Search(group=%q) total/hits = %d/%d, want 1/1", group, total, len(result.Hits))
		}
		if got := result.Hits[0].Fields["group"]; got != "COC" {
			t.Fatalf("Search(group=%q) stored group = %v, want COC", group, got)
		}
	}
}

func TestBlugeSearchEngineReopensPersistedIndex(t *testing.T) {
	oldIndexDir := indexDir
	indexDir = filepath.Join(t.TempDir(), "index")
	t.Cleanup(func() { indexDir = oldIndexDir })

	first, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatal(err)
	}
	addTestHelpItems(t, first, 1, "group", "from", "persisted-")
	if applyErr := first.AddItemApply(true); applyErr != nil {
		t.Fatal(applyErr)
	}
	first.Close()

	second, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if second.IndexFreshlyCreated() {
		t.Fatal("persisted Bluge index was treated as newly created")
	}
	ids, err := second.ListAllDocumentIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("reopened document count = %d, want 1", len(ids))
	}
}

func TestBlugeSearchEngineRebuildsIncompatibleIndex(t *testing.T) {
	oldIndexDir := indexDir
	indexDir = filepath.Join(t.TempDir(), "index")
	t.Cleanup(func() { indexDir = oldIndexDir })

	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(indexDir, "index_meta.json"), []byte(`{"store":"scorch"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if !engine.IndexFreshlyCreated() {
		t.Fatal("incompatible index was not rebuilt")
	}
	if _, err := os.Stat(filepath.Join(indexDir, "index_meta.json")); !os.IsNotExist(err) {
		t.Fatalf("incompatible index marker still exists: %v", err)
	}
}

func TestBlugeSearchEngineRebuildsOldSchema(t *testing.T) {
	oldIndexDir := indexDir
	indexDir = filepath.Join(t.TempDir(), "index")
	t.Cleanup(func() { indexDir = oldIndexDir })

	first, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatal(err)
	}
	addTestHelpItems(t, first, 1, "group", "from", "old-schema-")
	if applyErr := first.AddItemApply(true); applyErr != nil {
		t.Fatal(applyErr)
	}
	first.Close()
	if writeErr := os.WriteFile(filepath.Join(indexDir, indexSchemaFile), []byte("1"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	second, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if !second.IndexFreshlyCreated() {
		t.Fatal("old schema index was not rebuilt")
	}
	ids, err := second.ListAllDocumentIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("rebuilt index contains %d old documents", len(ids))
	}
}

func TestBlugeSearchEngineDoesNotDeleteLockedIndex(t *testing.T) {
	oldIndexDir := indexDir
	indexDir = filepath.Join(t.TempDir(), "index")
	t.Cleanup(func() { indexDir = oldIndexDir })

	first, err := NewBlugeSearchEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	sentinel := filepath.Join(indexDir, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if second, err := NewBlugeSearchEngine(); err == nil {
		second.Close()
		t.Fatal("opening a second writer unexpectedly succeeded")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("locked index contents were deleted: %v", err)
	}
}
