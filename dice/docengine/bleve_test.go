package docengine //nolint:testpackage // Tests need access to unexported index helpers.

import (
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

func newTestBleveSearchEngine(t *testing.T) *BleveSearchEngine {
	t.Helper()

	oldIndexDir := indexDir
	indexDir = filepath.Join(t.TempDir(), "index")
	t.Cleanup(func() {
		indexDir = oldIndexDir
	})

	engine, err := NewBleveSearchEngine()
	if err != nil {
		t.Fatalf("NewBleveSearchEngine() error = %v", err)
	}
	t.Cleanup(func() {
		engine.Close()
	})

	return engine
}

func addTestHelpItems(t *testing.T, engine *BleveSearchEngine, count int, group, from, titlePrefix string) {
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

func TestBleveSearchEngineDeleteByFromRemovesAllMatches(t *testing.T) {
	engine := newTestBleveSearchEngine(t)

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

	deleteErr := engine.DeleteByFrom(targetFrom)
	if deleteErr != nil {
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
		item, getErr := engine.GetItemByID(id)
		if getErr != nil {
			t.Fatalf("GetItemByID(%q) error = %v", id, getErr)
		}
		if item.From == targetFrom {
			t.Fatalf("document %q from deleted source still exists", id)
		}
	}
}

func TestBleveSearchEngineNumericIDsRebuildLazily(t *testing.T) {
	engine := newTestBleveSearchEngine(t)

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

	deleteErr := engine.DeleteByGroup("remove-me")
	if deleteErr != nil {
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
	_, parseErr := strconv.Atoi(res.Hits[0].ID)
	if parseErr != nil {
		t.Fatalf("Search() returned non-numeric ID %q", res.Hits[0].ID)
	}
}
