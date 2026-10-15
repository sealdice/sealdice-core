package uploadgw

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"sealdice-core/dice"
	"sealdice-core/logger"
)

// minimalDM 构造仅够 custom-reply 落盘/重载使用的最小 Dice。
func minimalDM(t *testing.T) *dice.DiceManager {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	d := &dice.Dice{Logger: logger.M()}
	d.BaseConfig.Name = "uploadgw-test"
	d.BaseConfig.DataDir = dir
	d.Config = dice.NewConfig(d)
	dm := &dice.DiceManager{Dice: []*dice.Dice{d}}
	d.Parent = dm
	return dm
}

func TestCustomReplyRejectsExistingFile(t *testing.T) {
	dm := minimalDM(t)
	target := filepath.Join(dm.GetDice().GetExtDataDir("reply"), "existing.yaml")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}

	_, srv := newRealGateway(t, dm)
	resp := createOnly(t, srv, 1, map[string]string{
		MetaModule:   "custom-reply",
		MetaFilename: "existing.yaml",
		MetaSHA256:   sumHex([]byte("x")),
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("create status = %d, want 409", resp.StatusCode)
	}
}

func TestCustomReplyInstallsFile(t *testing.T) {
	dm := minimalDM(t)
	_, srv := newRealGateway(t, dm)

	content := []byte("hello")
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "custom-reply",
		MetaFilename: "team.txt",
		MetaSHA256:   sumHex(content),
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("patch status = %d, want 204", resp.StatusCode)
	}
	got, err := os.ReadFile(dm.GetDice().GetExtConfigFilePath("reply", "team.txt"))
	if err != nil {
		t.Fatalf("read installed reply file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("installed = %q, want %q", got, content)
	}
}
