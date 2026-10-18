package dice //nolint:testpackage // Exercise the complete help cache loading and metadata reconciliation path.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fy0/bluge"

	"sealdice-core/dice/docengine"
)

func TestHelpManagerRebuildsLegacyBlugeCache(t *testing.T) {
	fixture, err := filepath.Abs("docengine/testdata/bluge-v1/index")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if copyErr := os.CopyFS(docengine.DefaultIndexDir, os.DirFS(fixture)); copyErr != nil {
		t.Fatal(copyErr)
	}

	// This is an actual v1 segment produced by the previous dependency, not a
	// new index with a changed schema marker. The new library cannot read it.
	writer, err := bluge.OpenWriter(bluge.DefaultConfig(docengine.DefaultIndexDir))
	if err == nil {
		_ = writer.Close()
		t.Fatal("new Bluge unexpectedly accepted the legacy index without rebuilding")
	}
	if !strings.Contains(err.Error(), "unsupported version 1 for segment type: zapx-bluge") {
		t.Fatalf("legacy index failed for an unexpected reason: %v", err)
	}
	t.Logf("new Bluge rejects the original index: %v", err)

	const source = "data/helpdoc/upgrade.json"
	const title = "旧缓存重建验证"
	const content = "升级后帮助内容仍可检索"
	if mkdirErr := os.MkdirAll(filepath.Dir(source), 0o755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	help, err := json.Marshal(HelpDocFormat{
		Mod:     "upgrade",
		Helpdoc: map[string]string{title: content},
	})
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(source, help, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	hash, size, err := computeHelpFileHash(source)
	if err != nil {
		t.Fatal(err)
	}
	fileMeta := HelpFileMeta{Hash: hash, Size: size, Group: "default"}
	manager := &HelpManager{EngineType: BlugeSearch}
	// The source has not changed. If Load keeps trusting this metadata after
	// recreating the index, it will skip the file and silently lose its help.
	manager.saveHelpIndexMeta(&HelpIndexMeta{
		ContentParserVersion: helpContentParserVersion,
		Files: map[string]HelpFileMeta{
			source:                      fileMeta,
			"data/helpdoc/removed.json": {Hash: 1, Size: 1, Group: "default"},
		},
	})
	if meta, trusted := manager.loadHelpIndexMeta(); !trusted || meta.Files[source] != fileMeta {
		t.Fatal("test must start with trusted metadata matching the unchanged source")
	}

	var rebuiltID string
	for _, phase := range []string{"upgrade", "restart"} {
		t.Run(phase, func(t *testing.T) {
			manager := &HelpManager{EngineType: BlugeSearch}
			t.Cleanup(func() {
				if manager.searchEngine != nil {
					manager.Close()
				}
			})
			manager.Load(nil, nil, nil)
			engine, ok := manager.searchEngine.(*docengine.BlugeSearchEngine)
			if !ok || !manager.IsAvailable() {
				t.Fatal("help search is unavailable after loading")
			}
			if got, want := engine.IndexFreshlyCreated(), phase == "upgrade"; got != want {
				t.Fatalf("IndexFreshlyCreated() = %v, want %v", got, want)
			}
			result, total, _, _, err := engine.Search(nil, title, true, 10, 1, "default")
			if err != nil {
				t.Fatal(err)
			}
			if total != 1 || len(result.Hits) != 1 {
				t.Fatalf("unchanged help search total/hits = %d/%d, want 1/1", total, len(result.Hits))
			}
			item, err := manager.GetItemByNumericIDString(result.Hits[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			if item.Title != title || item.Content != content || item.From != source || item.PackageName != "upgrade" {
				t.Fatalf("rebuilt help differs from source: %+v", item)
			}
			if phase == "upgrade" {
				if item.InternalID == "legacy-help" {
					t.Fatal("legacy document was reused instead of reindexed from source")
				}
				rebuiltID = item.InternalID
			} else if item.InternalID != rebuiltID {
				t.Fatal("restart reindexed an unchanged help file instead of reusing the new cache")
			}
			_, removed, _, _, err := engine.Search(nil, "已删除旧条目", true, 10, 1, "default")
			if err != nil || removed != 0 {
				t.Fatalf("removed help search count/error = %d/%v, want 0/nil", removed, err)
			}
			meta, trusted := manager.loadHelpIndexMeta()
			if !trusted || len(meta.Files) != 1 || meta.Files[source] != fileMeta {
				t.Fatalf("saved metadata does not match rebuilt sources: %+v, trusted=%v", meta, trusted)
			}
			t.Logf("help is searchable with numeric ID %s and internal ID %s", result.Hits[0].ID, item.InternalID)
		})
	}
}
