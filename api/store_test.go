package api //nolint:testpackage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func TestStorePackageFilePreviewSecuresUpstreamResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Store","protocolVersions":["2.0"]}`))
		case "/dice/api/store/file/alice/demo/1.2.3":
			w.Header().Set("Content-Type", "text/html")
			if r.URL.Query().Get("path") == "missing.html" {
				w.WriteHeader(http.StatusNotFound)
			}
			_, _ = w.Write([]byte(`<script>remote()</script>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldBackendURLs := dice.BackendUrls
	dice.BackendUrls = []string{server.URL}
	defer func() { dice.BackendUrls = oldBackendURLs }()

	testDice := &dice.Dice{}
	testDice.StoreManager = dice.NewStoreManager(testDice)
	previousDice := myDice
	myDice = testDice
	defer func() { myDice = previousDice }()

	request := func(path string) *httptest.ResponseRecorder {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/sd-api/store/file/alice/demo/1.2.3?path="+path, nil)
		rec := httptest.NewRecorder()
		ctx := e.NewContext(req, rec)
		ctx.SetParamNames("namespace", "package", "version")
		ctx.SetParamValues("alice", "demo", "1.2.3")
		if err := storePackageFilePreview(ctx); err != nil {
			t.Fatalf("storePackageFilePreview() error = %v", err)
		}
		return rec
	}

	success := request("preview.html")
	if success.Code != http.StatusOK || !strings.Contains(success.Body.String(), "remote()") {
		t.Fatalf("success response = %d %q", success.Code, success.Body.String())
	}
	if got := success.Header().Get("Content-Security-Policy"); !strings.Contains(got, "sandbox") {
		t.Fatalf("Content-Security-Policy = %q", got)
	}
	if got := success.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}

	notFound := request("missing.html")
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d", notFound.Code)
	}
	if strings.Contains(notFound.Body.String(), "remote()") {
		t.Fatalf("upstream error body was forwarded: %q", notFound.Body.String())
	}
}
