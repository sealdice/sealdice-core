package dice

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestSaveTextBacksUpExistingTemplateBeforeReplace(t *testing.T) {
	dataDir := t.TempDir()
	configDir := filepath.Join(dataDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) error = %v", err)
	}

	templatePath := filepath.Join(configDir, "text-template.yaml")
	backupPath := filepath.Join(configDir, "text-template.yaml.bak")
	oldContent := []byte("核心:\n  旧文案:\n    - - old\n      - 1\n")
	if err := os.WriteFile(templatePath, oldContent, 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) error = %v", err)
	}

	d := newSaveTextTestDice(dataDir, "new")
	if err := d.SaveText(); err != nil {
		t.Fatalf("SaveText() error = %v", err)
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("ReadFile(backupPath) error = %v", err)
	}
	if string(backupContent) != string(oldContent) {
		t.Fatalf("backup content = %q, want %q", string(backupContent), string(oldContent))
	}

	saved := readSavedTextTemplate(t, templatePath)
	if got := saved["核心"]["测试文案"][0][0]; got != "new" {
		t.Fatalf("saved text = %v, want new", got)
	}
}

func TestSaveTextCreatesTemplateWithoutEmptyBackup(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "configs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(configs) error = %v", err)
	}

	templatePath := filepath.Join(dataDir, "configs", "text-template.yaml")
	backupPath := filepath.Join(dataDir, "configs", "text-template.yaml.bak")
	d := newSaveTextTestDice(dataDir, "new")
	if err := d.SaveText(); err != nil {
		t.Fatalf("SaveText() error = %v", err)
	}

	saved := readSavedTextTemplate(t, templatePath)
	if got := saved["核心"]["测试文案"][0][0]; got != "new" {
		t.Fatalf("saved text = %v, want new", got)
	}
	if _, err := os.Stat(backupPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestSaveTextStopsWhenExistingTemplateCannotBeRead(t *testing.T) {
	dataDir := t.TempDir()
	templatePath := filepath.Join(dataDir, "configs", "text-template.yaml")
	backupPath := filepath.Join(dataDir, "configs", "text-template.yaml.bak")
	if err := os.MkdirAll(templatePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(templatePath) error = %v", err)
	}

	d := newSaveTextTestDice(dataDir, "new")
	if err := d.SaveText(); err == nil {
		t.Fatal("SaveText() error = nil, want error")
	}
	if _, err := os.Stat(backupPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestSaveTextDoesNotReplaceTemplateWhenMarshalFails(t *testing.T) {
	dataDir := t.TempDir()
	configDir := filepath.Join(dataDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) error = %v", err)
	}

	templatePath := filepath.Join(configDir, "text-template.yaml")
	oldContent := []byte("核心:\n  旧文案:\n    - - old\n      - 1\n")
	if err := os.WriteFile(templatePath, oldContent, 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) error = %v", err)
	}

	d := &Dice{
		BaseConfig: BaseConfig{DataDir: dataDir},
		Logger:     zap.NewNop().Sugar(),
		TextMapRaw: TextTemplateWithWeightDict{
			"核心": TextTemplateWithWeight{
				"测试文案": []TextTemplateItem{{func() {}, 1}},
			},
		},
	}
	if err := d.SaveText(); err == nil {
		t.Fatal("SaveText() error = nil, want error")
	}

	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("ReadFile(templatePath) error = %v", err)
	}
	if string(content) != string(oldContent) {
		t.Fatalf("template content = %q, want %q", string(content), string(oldContent))
	}
}

func TestWriteFileAtomicallyRemovesTempFileOnReplaceFailure(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.yaml")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(targetPath) error = %v", err)
	}

	if err := writeFileAtomically(targetPath, []byte("new"), 0o644); err == nil {
		t.Fatal("writeFileAtomically() error = nil, want error")
	}

	matches, err := filepath.Glob(filepath.Join(dir, ".target.yaml.tmp-*"))
	if err != nil {
		t.Fatalf("Glob(temp files) error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func newSaveTextTestDice(dataDir, text string) *Dice {
	return &Dice{
		BaseConfig: BaseConfig{DataDir: dataDir},
		Logger:     zap.NewNop().Sugar(),
		TextMapRaw: TextTemplateWithWeightDict{
			"核心": TextTemplateWithWeight{
				"测试文案": []TextTemplateItem{{text, 1}},
			},
		},
	}
}

func readSavedTextTemplate(t *testing.T, path string) TextTemplateWithWeightDict {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	texts := TextTemplateWithWeightDict{}
	if err := yaml.Unmarshal(data, &texts); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
	return texts
}
