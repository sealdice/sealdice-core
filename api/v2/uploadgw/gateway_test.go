package uploadgw //nolint:testpackage // 通过私有目标注册表注入可控落盘实现。

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

type fakeDestination struct {
	module    string
	installed []byte
	meta      map[string]string
	fail      bool
	gotSize   int64
	maxSize   int64
}

func (f *fakeDestination) Validate(meta map[string]string, size int64) error {
	f.gotSize = size
	if strings.TrimSpace(meta[MetaFilename]) == "" {
		return errors.New("filename required")
	}
	if f.maxSize > 0 && size > f.maxSize {
		return errors.New("file too large")
	}
	return nil
}

func (f *fakeDestination) Install(_ context.Context, stagedPath string, meta map[string]string) error {
	if f.fail {
		return errors.New("install failed")
	}
	data, err := os.ReadFile(stagedPath)
	if err != nil {
		return err
	}
	f.installed = data
	f.meta = meta
	return nil
}

func metadataHeader(meta map[string]string) string {
	parts := make([]string, 0, len(meta))
	for k, v := range meta {
		parts = append(parts, k+" "+base64.StdEncoding.EncodeToString([]byte(v)))
	}
	return strings.Join(parts, ",")
}

func newFakeGateway(t *testing.T, dst *fakeDestination) (*Gateway, *httptest.Server) {
	t.Helper()
	gw, err := New(&dice.DiceManager{}, t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	gw.dests = map[string]Destination{dst.module: dst}
	e := echo.New()
	// 与生产一致：通过 Echo 挂载，覆盖路由前缀与 Stripping 逻辑。
	gw.Mount(e, func(next echo.HandlerFunc) echo.HandlerFunc { return next })
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return gw, srv
}

// createUpload 走一次完整的 tus 流程（创建 + 单次 PATCH），返回最终响应。
func createUpload(t *testing.T, srv *httptest.Server, content []byte, meta map[string]string) *http.Response {
	t.Helper()

	createReq, err := http.NewRequest(http.MethodPost, srv.URL+BasePath, nil)
	if err != nil {
		t.Fatalf("new create request: %v", err)
	}
	createReq.Header.Set("Tus-Resumable", "1.0.0")
	createReq.Header.Set("Upload-Length", strconv.Itoa(len(content)))
	createReq.Header.Set("Upload-Metadata", metadataHeader(meta))

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("create status = %d, want 201; body=%s", createResp.StatusCode, body)
	}
	location := createResp.Header.Get("Location")
	if location == "" {
		t.Fatal("missing Location header")
	}

	patchReq, err := http.NewRequest(http.MethodPatch, location, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("new patch request: %v", err)
	}
	patchReq.Header.Set("Tus-Resumable", "1.0.0")
	patchReq.Header.Set("Upload-Offset", "0")
	patchReq.Header.Set("Content-Type", "application/offset+octet-stream")

	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	defer patchResp.Body.Close()
	return patchResp
}

func TestGatewayInstallsCompletedUpload(t *testing.T) {
	dst := &fakeDestination{module: "test"}
	_, srv := newFakeGateway(t, dst)

	content := []byte("console.log('hello');")
	sum := sha256.Sum256(content)
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "test",
		MetaFilename: "hello.js",
		MetaSHA256:   hex.EncodeToString(sum[:]),
	})
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("patch status = %d, want 204; body=%s", resp.StatusCode, body)
	}
	if string(dst.installed) != string(content) {
		t.Fatalf("installed = %q, want %q", dst.installed, content)
	}
	if dst.meta[MetaFilename] != "hello.js" {
		t.Fatalf("installed filename = %q, want hello.js", dst.meta[MetaFilename])
	}
}

func TestGatewayRejectsChecksumMismatch(t *testing.T) {
	dst := &fakeDestination{module: "test"}
	_, srv := newFakeGateway(t, dst)

	content := []byte("payload")
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "test",
		MetaFilename: "hello.js",
		MetaSHA256:   strings.Repeat("0", 64), // 错误的校验值
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("patch status = %d, want 400", resp.StatusCode)
	}
	if dst.installed != nil {
		t.Fatal("destination should not be installed on checksum mismatch")
	}
}

func TestGatewayRejectsUnknownModule(t *testing.T) {
	dst := &fakeDestination{module: "test"}
	_, srv := newFakeGateway(t, dst)

	content := []byte("payload")
	sum := sha256.Sum256(content)
	createReq, err := http.NewRequest(http.MethodPost, srv.URL+BasePath, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	createReq.Header.Set("Tus-Resumable", "1.0.0")
	createReq.Header.Set("Upload-Length", strconv.Itoa(len(content)))
	createReq.Header.Set("Upload-Metadata", metadataHeader(map[string]string{
		MetaModule:   "nope",
		MetaFilename: "x.js",
		MetaSHA256:   hex.EncodeToString(sum[:]),
	}))

	resp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create status = %d, want 400", resp.StatusCode)
	}
}

func sumHex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func TestGatewayPassesUploadSizeToDestination(t *testing.T) {
	dst := &fakeDestination{module: "test", maxSize: 5}
	_, srv := newFakeGateway(t, dst)

	content := []byte("12345")
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "test",
		MetaFilename: "a.bin",
		MetaSHA256:   sumHex(content),
	})
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("patch status = %d, want 204; body=%s", resp.StatusCode, body)
	}
	if dst.gotSize != int64(len(content)) {
		t.Fatalf("destination saw size = %d, want %d", dst.gotSize, len(content))
	}
}

func TestGatewayRejectsOversizeAtCreation(t *testing.T) {
	dst := &fakeDestination{module: "test", maxSize: 2}
	_, srv := newFakeGateway(t, dst)

	content := []byte("12345")
	createReq, err := http.NewRequest(http.MethodPost, srv.URL+BasePath, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	createReq.Header.Set("Tus-Resumable", "1.0.0")
	createReq.Header.Set("Upload-Length", strconv.Itoa(len(content)))
	createReq.Header.Set("Upload-Metadata", metadataHeader(map[string]string{
		MetaModule:   "test",
		MetaFilename: "a.bin",
		MetaSHA256:   sumHex(content),
	}))

	resp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("create status = %d, want >= 400 for oversize", resp.StatusCode)
	}
}
