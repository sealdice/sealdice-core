package uploadgw //nolint:testpackage // 与包内网关夹具共享未导出的落盘实现。

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

// newRealGateway 用真实注册的 destinations 挂一个网关（区别于 newFakeGateway 的注入式）。
func newRealGateway(t *testing.T, dm *dice.DiceManager) (*Gateway, *httptest.Server) {
	t.Helper()
	gw, err := New(dm, t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e := echo.New()
	gw.Mount(e, func(next echo.HandlerFunc) echo.HandlerFunc { return next })
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return gw, srv
}

func createOnly(t *testing.T, srv *httptest.Server, length int, meta map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+BasePath, nil)
	if err != nil {
		t.Fatalf("new create request: %v", err)
	}
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Length", strconv.Itoa(length))
	req.Header.Set("Upload-Metadata", metadataHeader(meta))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	return resp
}

func TestResourceDestinationInstallsImage(t *testing.T) {
	t.Chdir(t.TempDir())
	_, srv := newRealGateway(t, &dice.DiceManager{})

	content := []byte("png-bytes")
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "resource",
		MetaFilename: "a.png",
		MetaSHA256:   sumHex(content),
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("patch status = %d, want 204", resp.StatusCode)
	}
	got, err := os.ReadFile(filepath.Join("data", "images", "a.png"))
	if err != nil {
		t.Fatalf("read installed image: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("installed = %q, want %q", got, content)
	}
}

func TestResourceDestinationRejectsWrongExt(t *testing.T) {
	t.Chdir(t.TempDir())
	_, srv := newRealGateway(t, &dice.DiceManager{})

	content := []byte("x")
	resp := createOnly(t, srv, len(content), map[string]string{
		MetaModule:   "resource",
		MetaFilename: "a.txt",
		MetaSHA256:   sumHex(content),
	})
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("create status = %d, want >= 400 for wrong ext", resp.StatusCode)
	}
}

func TestResourceDestinationRejectsOversize(t *testing.T) {
	t.Chdir(t.TempDir())
	_, srv := newRealGateway(t, &dice.DiceManager{})

	resp := createOnly(t, srv, int(maxResourceUploadBytes)+1, map[string]string{
		MetaModule:   "resource",
		MetaFilename: "big.png",
		MetaSHA256:   sumHex([]byte("x")),
	})
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("create status = %d, want >= 400 for oversize", resp.StatusCode)
	}
}
