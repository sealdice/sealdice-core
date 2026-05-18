package dice //nolint:testpackage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sealdice-core/dice/sealpack"
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
		"legacy store":       `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"store":{"category":"rules"},"download":{"url":"https://example.com/demo.sealpack"}}`,
		"legacy downloadUrl": `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpack"},"downloadUrl":"https://example.com/demo.sealpack"}`,
		"legacy fullId":      `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpack"},"fullId":"alice/demo@1.2.3"}`,
		"legacy backendId":   `{"id":"alice/demo","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpack"},"backendID":"official"}`,
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

func TestDecodeJSONStrictAllowsStoreFormatVersionMetadata(t *testing.T) {
	infoRaw := `{"formatVersion":"2.0","name":"Official Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`
	var info storeBackendInfoResponse
	if err := decodeJSONStrict([]byte(infoRaw), &info); err != nil {
		t.Fatalf("decode info with formatVersion returned error: %v", err)
	}
	if info.FormatVersion != "2.0" {
		t.Fatalf("FormatVersion = %q, want 2.0", info.FormatVersion)
	}

	pageRaw := `{"formatVersion":"2.0","result":true,"data":{"formatVersion":"2.0","data":[],"pageNum":1,"pageSize":20,"next":false},"err":""}`
	var page storePageResponse
	if err := decodeJSONStrict([]byte(pageRaw), &page); err != nil {
		t.Fatalf("decode page with formatVersion returned error: %v", err)
	}
	if page.FormatVersion != "2.0" || page.Data == nil || page.Data.FormatVersion != "2.0" {
		t.Fatalf("unexpected decoded page metadata: %#v", page)
	}

	recommendRaw := `{"formatVersion":"2.0","result":true,"data":[],"err":""}`
	var recommend storeRecommendResponse
	if err := decodeJSONStrict([]byte(recommendRaw), &recommend); err != nil {
		t.Fatalf("decode recommend with formatVersion returned error: %v", err)
	}
	if recommend.FormatVersion != "2.0" {
		t.Fatalf("FormatVersion = %q, want 2.0", recommend.FormatVersion)
	}

	packageRaw := `{"id":"alice/demo","formatVersion":"1.0.0","version":"1.2.3","name":"Demo","contents":["scripts"],"download":{"url":"https://example.com/demo.sealpack"}}`
	var pkg StorePackage
	if err := decodeJSONStrict([]byte(packageRaw), &pkg); err != nil {
		t.Fatalf("decode package with formatVersion returned error: %v", err)
	}
	if pkg.FormatVersion != "1.0.0" {
		t.Fatalf("FormatVersion = %q, want 1.0.0", pkg.FormatVersion)
	}
}

func TestStorePackageMarshalUsesUnifiedSchema(t *testing.T) {
	data, err := json.Marshal(&StorePackage{
		ID:       "alice/demo",
		Version:  "1.2.3",
		FullID:   "alice/demo@1.2.3",
		Name:     "Demo",
		Contents: []string{"scripts"},
		StoreAssets: sealpack.StoreInfo{
			Category: "rules",
		},
		Download: StorePackageDownload{
			URL: "https://example.com/demo-1.2.3.sealpack",
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
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Official Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		case "/dice/api/store/page":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","result":true,"data":{"formatVersion":"2.0","data":[{"id":"alice/demo","formatVersion":"1.0.0","version":"1.2.3","name":"Demo","authors":["Alice"],"description":"demo","license":"MIT","homepage":"https://example.com","repository":"https://example.com/repo","keywords":["coc"],"contents":["scripts"],"seal":{},"dependencies":{},"storeAssets":{"category":"rules","screenshots":[]},"download":{"url":"https://example.com/demo-1.2.3.sealpack","hash":{"sha256":"abc"},"releaseTime":1,"updateTime":2,"downloadCount":3}}],"pageNum":1,"pageSize":20,"next":false},"err":""}`))
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
	if page.Data[0].FormatVersion != "1.0.0" {
		t.Fatalf("FormatVersion = %q", page.Data[0].FormatVersion)
	}
}

func TestStoreManagerFindPackageMatchesByIDAndVersionAfterRefreshInstalled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Official Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		case "/dice/api/store/page":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","result":true,"data":{"formatVersion":"2.0","data":[{"id":"alice/demo","formatVersion":"1.0.0","version":"1.2.3","name":"Demo","authors":["Alice"],"description":"demo","license":"MIT","homepage":"https://example.com","repository":"https://example.com/repo","keywords":["coc"],"contents":["scripts"],"seal":{},"dependencies":{},"storeAssets":{"category":"rules","screenshots":[]},"download":{"url":"https://example.com/demo-1.2.3.sealpack","hash":{"sha256":"abc"},"releaseTime":1,"updateTime":2,"downloadCount":3}}],"pageNum":1,"pageSize":20,"next":false},"err":""}`))
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
	manager.RefreshInstalled(page.Data)

	pkg, ok := manager.FindPackage("alice/demo", "1.2.3")
	if !ok {
		t.Fatal("expected package to be found in cache")
	}
	if pkg.Download.URL != "https://example.com/demo-1.2.3.sealpack" {
		t.Fatalf("Download.URL = %q", pkg.Download.URL)
	}
}

func TestStoreQueryRecommendCachesBackendInfo(t *testing.T) {
	var infoRequests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			atomic.AddInt32(&infoRequests, 1)
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Official Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		case "/dice/api/store/recommend":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","result":true,"data":[],"err":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldBackendURLs := BackendUrls
	BackendUrls = []string{server.URL}
	defer func() { BackendUrls = oldBackendURLs }()

	manager := NewStoreManager(&Dice{})
	if _, err := manager.StoreQueryRecommend(); err != nil {
		t.Fatalf("StoreQueryRecommend() first error = %v", err)
	}
	if _, err := manager.StoreQueryRecommend(); err != nil {
		t.Fatalf("StoreQueryRecommend() second error = %v", err)
	}
	if got := atomic.LoadInt32(&infoRequests); got != 1 {
		t.Fatalf("info requests = %d, want 1", got)
	}

	manager.lock.Lock()
	manager.backendFetchedAt = time.Now().Add(-storeBackendInfoCacheTTL - time.Second)
	manager.lock.Unlock()

	if _, err := manager.StoreQueryRecommend(); err != nil {
		t.Fatalf("StoreQueryRecommend() after TTL error = %v", err)
	}
	if got := atomic.LoadInt32(&infoRequests); got != 2 {
		t.Fatalf("info requests after TTL = %d, want 2", got)
	}
}

func TestStoreBackendListReturnsEnabledAndDisabledBackends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Custom Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldBackendURLs := BackendUrls
	BackendUrls = []string{server.URL}
	defer func() { BackendUrls = oldBackendURLs }()

	enabledURL := server.URL + "/dice/api/store"
	disabledURL := server.URL + "/disabled/dice/api/store"
	manager := NewStoreManager(&Dice{Config: Config{StoreConfig: StoreConfig{
		BackendUrls:         []string{enabledURL + "/"},
		DisabledBackendUrls: []string{disabledURL + "/"},
	}}})
	backends := manager.StoreBackendList()
	if len(backends) != 3 {
		t.Fatalf("len(backends) = %d", len(backends))
	}
	if backends[0].ID != "official" || !backends[0].Enabled {
		t.Fatalf("official backend = %#v, want enabled official", backends[0])
	}
	if backends[1].Url != enabledURL || !backends[1].Enabled {
		t.Fatalf("enabled backend = %#v", backends[1])
	}
	if backends[2].Url != disabledURL || backends[2].Enabled {
		t.Fatalf("disabled backend = %#v", backends[2])
	}
}

func TestStoreSetBackendEnabledMovesURLBetweenLists(t *testing.T) {
	manager := NewStoreManager(&Dice{Config: Config{StoreConfig: StoreConfig{
		BackendUrls: []string{
			"https://enabled.example/dice/api/store",
			"https://other.example/dice/api/store",
		},
	}}})

	if err := manager.StoreSetBackendEnabled("", "https://enabled.example/dice/api/store", false); err != nil {
		t.Fatalf("StoreSetBackendEnabled(false) error = %v", err)
	}
	if got := manager.parent.Config.BackendUrls; len(got) != 1 || got[0] != "https://other.example/dice/api/store" {
		t.Fatalf("BackendUrls = %#v", got)
	}
	if got := manager.parent.Config.DisabledBackendUrls; len(got) != 1 || got[0] != "https://enabled.example/dice/api/store" {
		t.Fatalf("DisabledBackendUrls = %#v", got)
	}

	if err := manager.StoreSetBackendEnabled("", "https://enabled.example/dice/api/store", true); err != nil {
		t.Fatalf("StoreSetBackendEnabled(true) error = %v", err)
	}
	if got := manager.parent.Config.BackendUrls; len(got) != 2 ||
		got[0] != "https://other.example/dice/api/store" ||
		got[1] != "https://enabled.example/dice/api/store" {
		t.Fatalf("BackendUrls = %#v", got)
	}
	if len(manager.parent.Config.DisabledBackendUrls) != 0 {
		t.Fatalf("DisabledBackendUrls = %#v, want empty", manager.parent.Config.DisabledBackendUrls)
	}
}

func TestStoreSetBackendEnabledRejectsDisablingOnlyCustomBackendWithoutOfficial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Custom Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	customURL := server.URL + "/dice/api/store"

	oldBackendURLs := BackendUrls
	BackendUrls = nil
	defer func() { BackendUrls = oldBackendURLs }()

	manager := NewStoreManager(&Dice{Config: Config{StoreConfig: StoreConfig{
		BackendUrls: []string{customURL},
	}}})

	if err := manager.StoreSetBackendEnabled("", customURL, false); err == nil {
		t.Fatal("expected disabling only custom backend to fail")
	}
	if got := manager.parent.Config.BackendUrls; len(got) != 1 || got[0] != customURL {
		t.Fatalf("BackendUrls = %#v", got)
	}
}

func TestStoreSetBackendEnabledAllowsDisablingOnlyCustomBackendWithOfficial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Custom Store","protocolVersions":["2.0"],"announcement":"ready","sign":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	customURL := server.URL + "/dice/api/store"

	oldBackendURLs := BackendUrls
	BackendUrls = []string{"https://official.example"}
	defer func() { BackendUrls = oldBackendURLs }()

	manager := NewStoreManager(&Dice{Config: Config{StoreConfig: StoreConfig{
		BackendUrls: []string{customURL},
	}}})

	if err := manager.StoreSetBackendEnabled("", customURL, false); err != nil {
		t.Fatalf("StoreSetBackendEnabled(false) error = %v", err)
	}
	if got := manager.parent.Config.BackendUrls; len(got) != 0 {
		t.Fatalf("BackendUrls = %#v, want empty", got)
	}
	if got := manager.parent.Config.DisabledBackendUrls; len(got) != 1 || got[0] != customURL {
		t.Fatalf("DisabledBackendUrls = %#v", got)
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
			URL: "https://example.com/demo-1.2.3.sealpack",
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
			URL: "https://example.com/demo-1.2.3.sealpack",
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
