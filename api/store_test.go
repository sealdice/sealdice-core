package api //nolint:testpackage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func TestStorePackageInfoListAllowsMultipleVersionsOfSamePackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Store","protocolVersions":["2.0"]}`))
		case "/dice/api/store/file/alice/demo/1.0.0":
			_, _ = w.Write([]byte("format_version = \"1.0.0\"\n[package]\nid = \"alice/demo\"\nname = \"Demo v1\"\nversion = \"1.0.0\"\n"))
		case "/dice/api/store/file/alice/demo/2.0.0":
			_, _ = w.Write([]byte("format_version = \"1.0.0\"\n[package]\nid = \"alice/demo\"\nname = \"Demo v2\"\nversion = \"2.0.0\"\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	token := "test-token"
	manager := &dice.DiceManager{}
	manager.AccessTokens.Store(token, true)
	testDice := &dice.Dice{
		Parent: manager,
		Config: dice.Config{StoreConfig: dice.StoreConfig{
			BackendUrls: []string{server.URL + "/dice/api/store"},
		}},
	}
	testDice.PackageManager = dice.NewPackageManager(testDice)
	testDice.StoreManager = dice.NewStoreManager(testDice)
	previousDice := myDice
	myDice = testDice
	t.Cleanup(func() { myDice = previousDice })

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sd-api/store/package-info-list", strings.NewReader(
		`{"packages":[{"id":"alice/demo","version":"1.0.0"},{"id":"alice/demo","version":"2.0.0"}]}`,
	))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Token", token)
	rec := httptest.NewRecorder()
	if err := storePackageInfoList(e.NewContext(req, rec)); err != nil {
		t.Fatalf("storePackageInfoList() error = %v", err)
	}

	var response struct {
		Result bool                         `json:"result"`
		Data   []storePackageInfoItemResult `json:"data"`
		Err    string                       `json:"err"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Result || response.Err != "" {
		t.Fatalf("response = %#v", response)
	}
	if len(response.Data) != 2 || response.Data[0].Name != "Demo v1" || response.Data[1].Name != "Demo v2" {
		t.Fatalf("response data = %#v", response.Data)
	}
}

func TestSameStorePackageVersion(t *testing.T) {
	if !sameStorePackageVersion("v1.2.3", "1.2.3") {
		t.Fatal("equivalent semantic versions did not match")
	}
	if sameStorePackageVersion("1.2.3", "1.2.4") {
		t.Fatal("different semantic versions matched")
	}
	if sameStorePackageVersion("invalid", "invalid") {
		t.Fatal("invalid versions matched")
	}
}

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

	testDice := &dice.Dice{Config: dice.Config{StoreConfig: dice.StoreConfig{
		BackendUrls: []string{server.URL + "/dice/api/store"},
	}}}
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
