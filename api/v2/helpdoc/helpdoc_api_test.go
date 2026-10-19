package helpdoc_test

import (
	"os"
	"path/filepath"
	"testing"

	. "sealdice-core/api/v2/helpdoc"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestHelpDocService(t *testing.T) *Service {
	t.Helper()

	tempDir := t.TempDir()
	t.Chdir(tempDir)
	if err := os.MkdirAll(filepath.Join("data", "helpdoc"), 0o755); err != nil {
		t.Fatalf("mkdir helpdoc dir: %v", err)
	}

	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "helpdoc-test"
	d.BaseConfig.DataDir = tempDir
	d.Config = dice.NewConfig(d)
	dm := &dice.DiceManager{
		Dice: d,
		Help: &dice.HelpManager{
			HelpDocTree: []*dice.HelpDoc{
				{
					Key:   "file-1",
					Name:  "sample.json",
					Path:  "data/helpdoc/sample.json",
					Group: "default",
					Type:  ".json",
				},
			},
			Config: &dice.HelpConfig{Aliases: map[string][]string{
				"default": {"d"},
			}},
		},
	}
	d.Parent = dm
	return NewService(dm)
}

func TestGetTreeReturnsHelpDocTree(t *testing.T) {
	svc := newTestHelpDocService(t)

	resp, err := svc.GetTree(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetTree returned error: %v", err)
	}
	if len(resp.Body.Item.Data) != 1 {
		t.Fatalf("tree length = %d, want 1", len(resp.Body.Item.Data))
	}
	if resp.Body.Item.Data[0].Name != "sample.json" {
		t.Fatalf("tree first name = %q, want sample.json", resp.Body.Item.Data[0].Name)
	}
}

func TestSaveConfigPersistsAliases(t *testing.T) {
	svc := newTestHelpDocService(t)

	resp, err := svc.SetConfig(t.Context(), &ConfigReq{
		Body: HelpConfigBody{
			Aliases: map[string][]string{"default": {"d", "main"}},
		},
	})
	if err != nil {
		t.Fatalf("SetConfig returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("SetConfig success = false, want true")
	}

	data, err := os.ReadFile(filepath.Join("data", "helpdoc", dice.HelpConfigFilename))
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if string(data) == "" {
		t.Fatal("saved config is empty")
	}
}
