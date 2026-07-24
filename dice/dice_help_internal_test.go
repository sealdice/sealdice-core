package dice

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"sealdice-core/dice/docengine"
)

var errFakeSearchNotImplemented = errors.New("fake search engine: term title lookup not implemented")

//nolint:usetesting // This test changes cwd explicitly so Windows can restore it before TempDir cleanup.
func TestHelpManagerSaveHelpIndexMetaWritesToConfiguredPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Chdir(%q) error = %v", tempDir, err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	meta := &HelpIndexMeta{
		Files: map[string]HelpFileMeta{
			"data/helpdoc/example.json": {
				Hash:  1,
				Size:  2,
				Group: "default",
			},
		},
	}

	manager := &HelpManager{}
	manager.saveHelpIndexMeta(meta)

	data, err := os.ReadFile(helpIndexMetaPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", helpIndexMetaPath, err)
	}

	var got HelpIndexMeta
	err = json.Unmarshal(data, &got)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("saved meta files count = %d, want 1", len(got.Files))
	}
	if got.Files["data/helpdoc/example.json"].Group != "default" {
		t.Fatalf("saved meta group = %q, want %q", got.Files["data/helpdoc/example.json"].Group, "default")
	}
}

type fakeHelpSearchEngine struct {
	docs     map[string]*docengine.HelpTextItem
	pageDocs []*docengine.HelpTextItem
}

func (f *fakeHelpSearchEngine) GetSuffixText() string { return "" }

func (f *fakeHelpSearchEngine) GetPrefixText() string { return "" }

func (f *fakeHelpSearchEngine) GetShowBestRelativeGap() float64 { return 0 }

func (f *fakeHelpSearchEngine) Init() error { return nil }

func (f *fakeHelpSearchEngine) Close() {}

func (f *fakeHelpSearchEngine) AddItem(docengine.HelpTextItem) (string, error) { return "", nil }

func (f *fakeHelpSearchEngine) AddItemApply(bool) error { return nil }

func (f *fakeHelpSearchEngine) Search([]string, string, bool, int, int, string) (*docengine.GeneralSearchResult, int, int, int, error) {
	return nil, 0, 0, 0, nil
}

func (f *fakeHelpSearchEngine) GetHelpTextItemByTermTitle(string) (*docengine.HelpTextItem, error) {
	return nil, errFakeSearchNotImplemented
}

func (f *fakeHelpSearchEngine) GetItemByInternalID(id string) (*docengine.HelpTextItem, error) {
	item, ok := f.docs[id]
	if !ok {
		return nil, errors.New("document not found")
	}
	return item, nil
}

func (f *fakeHelpSearchEngine) PaginateDocuments(int, int, string, string, string) (uint64, []*docengine.HelpTextItem, error) {
	return uint64(len(f.pageDocs)), f.pageDocs, nil
}

func (f *fakeHelpSearchEngine) GetTotalID() uint64 { return uint64(len(f.docs)) }

func (f *fakeHelpSearchEngine) DeleteByFrom(string) error { return nil }

func (f *fakeHelpSearchEngine) DeleteByGroup(string) error { return nil }

func TestHelpManagerGetItemByNumericID_InvalidBounds(t *testing.T) {
	manager := &HelpManager{
		docIDs: []string{"doc-1", "doc-2"},
	}

	tests := []struct {
		name string
		id   int
	}{
		{name: "zero", id: 0},
		{name: "negative", id: -1},
		{name: "too large", id: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := manager.GetItemByNumericID(tt.id)
			if err == nil {
				t.Fatalf("GetItemByNumericID(%d) error = nil, want invalid ID error", tt.id)
			}
			if err.Error() != "无效的帮助条目ID" {
				t.Fatalf("GetItemByNumericID(%d) error = %q, want %q", tt.id, err.Error(), "无效的帮助条目ID")
			}
			if item != nil {
				t.Fatalf("GetItemByNumericID(%d) item = %#v, want nil", tt.id, item)
			}
		})
	}
}

func TestHelpManagerGetItemByNumericIDString_EmptyString(t *testing.T) {
	manager := &HelpManager{
		docIDs: []string{"doc-1"},
	}

	item, err := manager.GetItemByNumericIDString("")
	if err == nil {
		t.Fatalf("GetItemByNumericIDString(\"\") error = nil, want invalid ID error")
	}
	if err.Error() != "无效的帮助条目ID" {
		t.Fatalf("GetItemByNumericIDString(\"\") error = %q, want %q", err.Error(), "无效的帮助条目ID")
	}
	if item != nil {
		t.Fatalf("GetItemByNumericIDString(\"\") item = %#v, want nil", item)
	}
}

func TestHelpManagerGetItemByNumericID_ValidResolvesCorrectDoc(t *testing.T) {
	expected := &docengine.HelpTextItem{
		Title: "example",
	}
	manager := &HelpManager{
		docIDs: []string{"doc-1"},
		searchEngine: &fakeHelpSearchEngine{
			docs: map[string]*docengine.HelpTextItem{
				"doc-1": expected,
			},
		},
	}

	item, err := manager.GetItemByNumericID(1)
	if err != nil {
		t.Fatalf("GetItemByNumericID(1) error = %v", err)
	}
	if item != expected {
		t.Fatalf("GetItemByNumericID(1) item = %#v, want %#v", item, expected)
	}
}

func TestHelpManagerGetHelpItemPageResolvesNumericID(t *testing.T) {
	manager := &HelpManager{
		docIDs: []string{"internal-1"},
		searchEngine: &fakeHelpSearchEngine{docs: map[string]*docengine.HelpTextItem{
			"internal-1": {Title: "example"},
		}},
	}

	total, items := manager.GetHelpItemPage(1, 20, "1", "", "", "")
	if total != 1 || len(items) != 1 {
		t.Fatalf("GetHelpItemPage() total/items = %d/%d, want 1/1", total, len(items))
	}
	if items[0].ID != 1 || items[0].Title != "example" {
		t.Fatalf("GetHelpItemPage() item = %#v, want numeric ID 1 and title example", items[0])
	}
}

func TestHelpManagerGetHelpItemPageMapsPaginationInternalIDs(t *testing.T) {
	manager := &HelpManager{
		docIDs: []string{"internal-1", "internal-2"},
		searchEngine: &fakeHelpSearchEngine{pageDocs: []*docengine.HelpTextItem{
			{InternalID: "internal-2", Title: "second"},
			{InternalID: "internal-1", Title: "first"},
		}},
	}

	total, items := manager.GetHelpItemPage(1, 20, "", "", "", "")
	if total != 2 || len(items) != 2 {
		t.Fatalf("GetHelpItemPage() total/items = %d/%d, want 2/2", total, len(items))
	}
	if items[0].ID != 2 || items[1].ID != 1 {
		t.Fatalf("GetHelpItemPage() IDs = %d/%d, want 2/1", items[0].ID, items[1].ID)
	}
}

func TestReconcileHelpIndexMeta(t *testing.T) {
	existing := &HelpIndexMeta{
		Files: map[string]HelpFileMeta{
			"data/helpdoc/example.json": {
				Hash:  1,
				Size:  2,
				Group: "default",
			},
		},
	}

	t.Run("preserve trusted meta when index reused", func(t *testing.T) {
		gotMeta, gotTrusted := reconcileHelpIndexMeta(existing, true, false)
		if !gotTrusted {
			t.Fatalf("got trusted = false, want true")
		}
		if gotMeta != existing {
			t.Fatalf("got meta %#v, want original %#v", gotMeta, existing)
		}
	})

	t.Run("reset trusted meta when index recreated", func(t *testing.T) {
		gotMeta, gotTrusted := reconcileHelpIndexMeta(existing, true, true)
		if gotTrusted {
			t.Fatalf("got trusted = true, want false")
		}
		if len(gotMeta.Files) != 0 {
			t.Fatalf("got meta files = %d, want 0", len(gotMeta.Files))
		}
	})
}
