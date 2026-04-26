package dice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"sealdice-core/dice/sealpkg"
)

func TestParseStorePackageFullID(t *testing.T) {
	pkgID, version, err := ParseStorePackageFullID("alice/demo@1.2.3")
	if err != nil {
		t.Fatalf("ParseStorePackageFullID returned error: %v", err)
	}
	if pkgID != "alice/demo" {
		t.Fatalf("pkgID = %q", pkgID)
	}
	if version != "1.2.3" {
		t.Fatalf("version = %q", version)
	}
}

func TestDecodeJSONStrictRejectsLegacyStorePackageFields(t *testing.T) {
	cases := map[string]string{
		"legacy store":       `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"store":{"category":"rules"},"download":{"url":"https://example.com/demo.sealpkg"}}`,
		"legacy downloadUrl": `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpkg"},"downloadUrl":"https://example.com/demo.sealpkg"}`,
		"legacy fullId":      `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpkg"},"fullId":"alice/demo@1.2.3"}`,
		"legacy backendId":   `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpkg"},"backendID":"official"}`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			var pkg StorePackage
			err := decodeJSONStrict([]byte(raw), &pkg)
			if err == nil {
				t.Fatal("expected legacy field to be rejected")
			}
		})
	}
}

func TestStorePackageMarshalUsesUnifiedSchema(t *testing.T) {
	data, err := json.Marshal(&StorePackage{
		ID:       "alice/demo",
		Version:  "1.2.3",
		FullID:   "alice/demo@1.2.3",
		Name:     "Demo",
		Contents: []string{"scripts"},
		StoreAssets: sealpkg.StoreInfo{
			Category: "rules",
		},
		Download: StorePackageDownload{
			URL: "https://example.com/demo-1.2.3.sealpkg",
		},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	jsonText := string(data)
	if !strings.Contains(jsonText, `"storeAssets"`) {
		t.Fatalf("expected storeAssets in %s", jsonText)
	}
	if !strings.Contains(jsonText, `"download"`) {
		t.Fatalf("expected download in %s", jsonText)
	}
	if strings.Contains(jsonText, `"fullId"`) {
		t.Fatalf("did not expect fullId in %s", jsonText)
	}
	if strings.Contains(jsonText, `"backendID"`) {
		t.Fatalf("did not expect backendID in %s", jsonText)
	}
	if strings.Contains(jsonText, `"downloadUrl"`) {
		t.Fatalf("did not expect downloadUrl in %s", jsonText)
	}
	if strings.Contains(jsonText, `"store":`) {
		t.Fatalf("did not expect legacy store field in %s", jsonText)
	}
}

func TestNormalizeConfiguredStoreBackendURLKeepsFirstNonEmpty(t *testing.T) {
	got := normalizeConfiguredStoreBackendURL([]string{"  ", "https://first.example/store/", "https://second.example/store"})
	if got != "https://first.example/store" {
		t.Fatalf("normalizeConfiguredStoreBackendURL() = %q", got)
	}
}

func TestFindPackageUsesIDAndVersion(t *testing.T) {
	target := &StorePackage{ID: "alice/demo", Version: "1.2.3"}
	manager := &StoreManager{
		lock: new(sync.RWMutex),
		packageCache: map[string]*StorePackage{
			BuildStorePackageFullID(target.ID, target.Version): target,
		},
	}

	pkg, ok := manager.FindPackage(" alice/demo ", " 1.2.3 ")
	if !ok {
		t.Fatal("expected package to be found")
	}
	if pkg != target {
		t.Fatalf("FindPackage returned unexpected package: %#v", pkg)
	}
}

func TestFindPackageReturnsMissingWhenVersionDiffers(t *testing.T) {
	manager := &StoreManager{
		lock: new(sync.RWMutex),
		packageCache: map[string]*StorePackage{
			BuildStorePackageFullID("alice/demo", "1.2.3"): {ID: "alice/demo", Version: "1.2.3"},
		},
	}

	if _, ok := manager.FindPackage("alice/demo", "1.2.4"); ok {
		t.Fatal("expected version mismatch to miss cache")
	}
}

func TestStoreQueryPageUsesSingleResolvedBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"name":"Official Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		case "/dice/api/store/page":
			_, _ = w.Write([]byte(`{"result":true,"data":{"data":[{"id":"alice/demo","version":"1.2.3","name":"Demo","authors":["Alice"],"description":"demo","license":"MIT","homepage":"https://example.com","repository":"https://example.com/repo","keywords":["coc"],"contents":["scripts"],"seal":{},"dependencies":{},"storeAssets":{"category":"rules","screenshots":[]},"download":{"url":"https://example.com/demo-1.2.3.sealpkg","hash":{"sha256":"abc"},"releaseTime":1,"updateTime":2,"downloadCount":3}}],"pageNum":1,"pageSize":20,"next":false},"err":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldBackendURLs := BackendUrls
	BackendUrls = []string{server.URL}
	defer func() { BackendUrls = oldBackendURLs }()

	manager := NewStoreManager(&Dice{})
	page, err := manager.StoreQueryPage(StoreQueryPageParams{PageNum: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("StoreQueryPage() error = %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("len(page.Data) = %d", len(page.Data))
	}
	if page.Data[0].FullID != "alice/demo@1.2.3" {
		t.Fatalf("FullID = %q", page.Data[0].FullID)
	}
}

func TestStoreBackendListReturnsSingleCurrentBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"name":"Custom Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	manager := NewStoreManager(&Dice{Config: Config{StoreConfig: StoreConfig{BackendUrls: []string{server.URL + "/dice/api/store/"}}}})
	backends := manager.StoreBackendList()
	if len(backends) != 1 {
		t.Fatalf("len(backends) = %d", len(backends))
	}
	if backends[0].Url != server.URL+"/dice/api/store" {
		t.Fatalf("Url = %q", backends[0].Url)
	}
}

func TestSanitizeStorePackageRejectsMismatchedFullID(t *testing.T) {
	_, err := sanitizeStorePackage(&StorePackage{
		ID:       "alice/demo",
		Version:  "1.2.3",
		FullID:   "alice/demo@1.2.4",
		Name:     "Demo",
		Contents: []string{"scripts"},
		Download: StorePackageDownload{
			URL: "https://example.com/demo-1.2.3.sealpkg",
		},
	})
	if err == nil {
		t.Fatal("expected error for mismatched fullId")
	}
}

func TestSanitizeStorePackageMarksCanonicalFields(t *testing.T) {
	pkg, err := sanitizeStorePackage(&StorePackage{
		ID:       "alice/demo",
		Version:  "1.2.3",
		Name:     "Demo",
		Contents: []string{"scripts", "scripts", "decks"},
		Download: StorePackageDownload{
			URL: "https://example.com/demo-1.2.3.sealpkg",
		},
	})
	if err != nil {
		t.Fatalf("sanitizeStorePackage returned error: %v", err)
	}
	if pkg.FullID != "alice/demo@1.2.3" {
		t.Fatalf("FullID = %q", pkg.FullID)
	}
	if len(pkg.Contents) != 2 || pkg.Contents[0] != "scripts" || pkg.Contents[1] != "decks" {
		t.Fatalf("Contents = %#v", pkg.Contents)
	}
	if pkg.Download.Hash == nil {
		t.Fatal("expected Download.Hash to be initialized")
	}
	if pkg.StoreAssets.Screenshots == nil {
		t.Fatal("expected StoreAssets.Screenshots to be initialized")
	}
}
