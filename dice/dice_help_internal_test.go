package dice

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHelpManagerSaveHelpIndexMetaWritesToConfiguredPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
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
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("saved meta files count = %d, want 1", len(got.Files))
	}
	if got.Files["data/helpdoc/example.json"].Group != "default" {
		t.Fatalf("saved meta group = %q, want %q", got.Files["data/helpdoc/example.json"].Group, "default")
	}
}
