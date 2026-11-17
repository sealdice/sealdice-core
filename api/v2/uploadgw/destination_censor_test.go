package uploadgw //nolint:testpackage // 复用包内网关夹具验证词库落盘。

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"sealdice-core/dice"
)

func TestCensorDestinationInstallsWordFile(t *testing.T) {
	t.Chdir(t.TempDir())
	_, srv := newRealGateway(t, &dice.DiceManager{})

	content := []byte("badword\n")
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "censor",
		MetaFilename: "words.txt",
		MetaSHA256:   sumHex(content),
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("patch status = %d, want 204", resp.StatusCode)
	}
	got, err := os.ReadFile(filepath.Join("data", "censor", "words.txt"))
	if err != nil {
		t.Fatalf("read installed word file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("installed = %q, want %q", got, content)
	}
}

func TestCensorDestinationRejectsWrongExt(t *testing.T) {
	t.Chdir(t.TempDir())
	_, srv := newRealGateway(t, &dice.DiceManager{})

	resp := createOnly(t, srv, 1, map[string]string{
		MetaModule:   "censor",
		MetaFilename: "words.json",
		MetaSHA256:   sumHex([]byte("x")),
	})
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("create status = %d, want >= 400 for wrong ext", resp.StatusCode)
	}
}

func TestCensorDestinationRejectsOversize(t *testing.T) {
	t.Chdir(t.TempDir())
	_, srv := newRealGateway(t, &dice.DiceManager{})

	resp := createOnly(t, srv, int(maxCensorUploadBytes)+1, map[string]string{
		MetaModule:   "censor",
		MetaFilename: "words.txt",
		MetaSHA256:   sumHex([]byte("x")),
	})
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("create status = %d, want >= 400 for oversize", resp.StatusCode)
	}
}
