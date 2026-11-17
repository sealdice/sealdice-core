package dice_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"sealdice-core/dice"
)

func TestDiceManagerGetDiceReturnsManagedDice(t *testing.T) {
	want := &dice.Dice{}
	dm := &dice.DiceManager{Dice: want}

	if got := dm.GetDice(); got != want {
		t.Fatalf("GetDice() = %p, want %p", got, want)
	}
}

func TestDiceManagerLoadDiceUsesFirstCompatibleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatalf("MkdirAll(data) error = %v", err)
	}

	cfg := dice.Configs{
		DiceConfigs: []dice.BaseConfig{
			{Name: "primary"},
			{Name: "legacy-secondary"},
		},
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join("data", "dice.yaml"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(data/dice.yaml) error = %v", err)
	}

	dm := &dice.DiceManager{}
	dm.LoadDice()
	if dm.Cron != nil {
		t.Cleanup(func() {
			dm.Cron.Stop()
		})
	}

	got := dm.GetDice()
	if got == nil {
		t.Fatal("GetDice() = nil, want managed dice")
	}
	if got.BaseConfig.Name != "primary" {
		t.Fatalf("managed dice name = %q, want %q", got.BaseConfig.Name, "primary")
	}
}
