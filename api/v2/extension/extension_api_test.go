package extension_test

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.uber.org/zap"

	. "sealdice-core/api/v2/extension"
	"sealdice-core/dice"
	"sealdice-core/dice/sealpack"
	"sealdice-core/model/common/request"
)

func TestListRefreshEnableReloadAndConfigRoundTrip(t *testing.T) {
	svc, testDice := newTestExtensionService(t, "")

	const pkgID = "alice/reply-pack"
	archive := createTestSealPack(t, pkgID, "1.0.0", map[string][]string{
		"reply": {"reply/*.yaml"},
	}, map[string]string{
		"reply/main.yaml": "enable: true\nname: package\nitems: []\n",
	}, withStringConfig("mode", "basic"))
	copySealPackToDataPackages(t, archive, pkgID, "1.0.0")

	refreshResp, err := svc.RefreshPackages(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("RefreshPackages returned error: %v", err)
	}
	if !slices.Contains(refreshResp.Body.Item.Added, pkgID) {
		t.Fatalf("Added = %#v, want %q", refreshResp.Body.Item.Added, pkgID)
	}

	listResp, err := svc.ListPackages(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("ListPackages returned error: %v", err)
	}
	if len(listResp.Body.Item.Items) != 1 {
		t.Fatalf("ListPackages item count = %d, want 1", len(listResp.Body.Item.Items))
	}
	item := listResp.Body.Item.Items[0]
	if item.Manifest == nil || item.Manifest.Package.ID != pkgID {
		t.Fatalf("package item = %#v, want %q", item, pkgID)
	}

	enableResp, err := svc.Enable(t.Context(), &IDReq{Body: IDBody{ID: pkgID}})
	if err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if !enableResp.Body.Item.Success || !enableResp.Body.Item.ReloadNeeded {
		t.Fatalf("Enable result = %#v, want success with reload", enableResp.Body.Item)
	}

	reloadResp, err := svc.Reload(t.Context(), &IDReq{Body: IDBody{ID: pkgID}})
	if err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if !reloadResp.Body.Item.Success {
		t.Fatalf("Reload result = %#v, want success", reloadResp.Body.Item)
	}
	if got := reloadResp.Body.Item.ReloadedItems["reply"]; got == "" {
		t.Fatalf("ReloadedItems = %#v, want reply item", reloadResp.Body.Item.ReloadedItems)
	}

	schemaResp, err := svc.GetConfigSchema(t.Context(), &PackageIDQuery{ID: pkgID})
	if err != nil {
		t.Fatalf("GetConfigSchema returned error: %v", err)
	}
	if schemaResp.Body.Item["mode"].Type != "string" {
		t.Fatalf("config schema = %#v, want string mode", schemaResp.Body.Item)
	}

	if _, err = svc.PutConfig(t.Context(), &ConfigReq{
		ID:   pkgID,
		Body: map[string]interface{}{"mode": "custom"},
	}); err != nil {
		t.Fatalf("PutConfig returned error: %v", err)
	}

	configResp, err := svc.GetConfig(t.Context(), &PackageIDQuery{ID: pkgID})
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if got := configResp.Body.Item["mode"]; got != "custom" {
		t.Fatalf("config mode = %#v, want custom", got)
	}

	pkg, ok := testDice.PackageManager.Get(pkgID)
	if !ok || pkg == nil {
		t.Fatalf("PackageManager missing %q after round trip", pkgID)
	}
	if got := pkg.Config["mode"]; got != "custom" {
		t.Fatalf("stored config mode = %#v, want custom", got)
	}
}

func TestStoreInstallListInstallsAndSkipsExactVersion(t *testing.T) {
	const pkgID = "alice/store-pack"
	const version = "1.2.3"

	archive := createTestSealPack(t, pkgID, version, map[string][]string{
		"reply": {"reply/*.yaml"},
	}, map[string]string{
		"reply/main.yaml": "enable: true\nname: store\nitems: []\n",
	})
	archiveData, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", archive, err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("store test request: %s", r.URL.Path)
		switch r.URL.Path {
		case "/dice/api/store/info":
			_, _ = w.Write([]byte(`{"formatVersion":"2.0","name":"Store","protocolVersions":["2.0"]}`))
		case "/dice/api/store/packages/alice/store-pack/1.2.3/store-pack@1.2.3.sealpack":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(archiveData)
		default:
			if strings.Contains(r.URL.Path, "/dice/api/store/packages/") &&
				strings.HasSuffix(r.URL.Path, ".sealpack") {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(archiveData)
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc, testDice := newTestExtensionService(t, server.URL+"/dice/api/store")

	firstResp, err := svc.StoreInstallList(t.Context(), &StoreInstallListReq{
		Body: StoreInstallListBody{
			Packages: []StoreInstallItem{
				{ID: pkgID, Version: version},
			},
		},
	})
	if err != nil {
		t.Fatalf("StoreInstallList first call returned error: %v", err)
	}
	if firstResp.Body.Item.Installed != 1 || firstResp.Body.Item.Skipped != 0 || firstResp.Body.Item.Failed != 0 {
		t.Fatalf("first StoreInstallList summary = %#v", firstResp.Body.Item)
	}
	if len(firstResp.Body.Item.Items) != 1 || firstResp.Body.Item.Items[0].Status != "installed" {
		t.Fatalf("first StoreInstallList items = %#v", firstResp.Body.Item.Items)
	}

	pkg, ok := testDice.PackageManager.Get(pkgID)
	if !ok || pkg == nil || pkg.Manifest == nil || pkg.Manifest.Package.Version != version {
		t.Fatalf("installed package = %#v, want %s@%s", pkg, pkgID, version)
	}

	secondResp, err := svc.StoreInstallList(t.Context(), &StoreInstallListReq{
		Body: StoreInstallListBody{
			Packages: []StoreInstallItem{
				{ID: pkgID, Version: version},
			},
		},
	})
	if err != nil {
		t.Fatalf("StoreInstallList second call returned error: %v", err)
	}
	if secondResp.Body.Item.Installed != 0 || secondResp.Body.Item.Skipped != 1 || secondResp.Body.Item.Failed != 0 {
		t.Fatalf("second StoreInstallList summary = %#v", secondResp.Body.Item)
	}
	if len(secondResp.Body.Item.Items) != 1 || secondResp.Body.Item.Items[0].Status != "skipped" {
		t.Fatalf("second StoreInstallList items = %#v", secondResp.Body.Item.Items)
	}
}

func newTestExtensionService(t *testing.T, backendURL string) (*Service, *dice.Dice) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: "."},
		Config:     dice.NewConfig(nil),
		Logger:     zap.NewNop().Sugar(),
	}
	testDice.Config.DataDir = "."
	if backendURL != "" {
		testDice.Config.BackendUrls = []string{backendURL}
	}
	testDice.ImSession = &dice.IMSession{
		Parent:       testDice,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}

	dm := &dice.DiceManager{Dice: []*dice.Dice{testDice}}
	testDice.Parent = dm

	testDice.PackageManager = dice.NewPackageManager(testDice)
	if err := testDice.PackageManager.Init(); err != nil {
		t.Fatalf("PackageManager.Init() error = %v", err)
	}
	testDice.StoreManager = dice.NewStoreManager(testDice)

	return NewService(dm), testDice
}

func copySealPackToDataPackages(t *testing.T, archivePath, pkgID, version string) {
	t.Helper()
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", archivePath, err)
	}
	namespace, packageName, err := sealpack.ParsePackageID(pkgID)
	if err != nil {
		t.Fatalf("ParsePackageID(%q) error = %v", pkgID, err)
	}
	dir := filepath.Join("data", "packages", namespace)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dir, err)
	}
	filename := filepath.Join(dir, packageName+"@"+version+sealpack.Extension)
	if err = os.WriteFile(filename, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", filename, err)
	}
}

type manifestOption func(*strings.Builder)

func withStringConfig(key string, defaultValue string) manifestOption {
	return func(builder *strings.Builder) {
		builder.WriteString("\n[config.")
		builder.WriteString(key)
		builder.WriteString("]\n")
		builder.WriteString("type = \"string\"\n")
		builder.WriteString("default = ")
		builder.WriteString("\"")
		builder.WriteString(defaultValue)
		builder.WriteString("\"\n")
	}
}

func createTestSealPack(t *testing.T, pkgID, version string, contents map[string][]string, files map[string]string, opts ...manifestOption) string {
	t.Helper()
	tempDir := filepath.Join(".", "temp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", tempDir, err)
	}
	archivePath := filepath.Join(tempDir, strings.ReplaceAll(pkgID, "/", "-")+"-"+version+sealpack.Extension)
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", archivePath, err)
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	infoWriter, err := zw.Create("info.toml")
	if err != nil {
		t.Fatalf("Create(info.toml) error = %v", err)
	}
	if _, err = infoWriter.Write([]byte(buildTestManifest(pkgID, version, contents, opts...))); err != nil {
		t.Fatalf("Write(info.toml) error = %v", err)
	}
	for name, body := range files {
		w, createErr := zw.Create(name)
		if createErr != nil {
			t.Fatalf("Create(%s) error = %v", name, createErr)
		}
		if _, writeErr := w.Write([]byte(body)); writeErr != nil {
			t.Fatalf("Write(%s) error = %v", name, writeErr)
		}
	}
	if err = zw.Close(); err != nil {
		t.Fatalf("Close(zip) error = %v", err)
	}
	return archivePath
}

func buildTestManifest(pkgID, version string, contents map[string][]string, opts ...manifestOption) string {
	var builder strings.Builder
	builder.WriteString("[package]\n")
	builder.WriteString("id = \"" + pkgID + "\"\n")
	builder.WriteString("name = \"Test Package\"\n")
	builder.WriteString("version = \"" + version + "\"\n")
	builder.WriteString("authors = [\"Tester\"]\n")
	builder.WriteString("license = \"MIT\"\n")
	builder.WriteString("description = \"test\"\n")
	if len(contents) > 0 {
		builder.WriteString("\n[contents]\n")
		keys := make([]string, 0, len(contents))
		for key := range contents {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			builder.WriteString(key + " = [")
			for index, pattern := range contents[key] {
				if index > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString("\"" + pattern + "\"")
			}
			builder.WriteString("]\n")
		}
	}
	for _, opt := range opts {
		opt(&builder)
	}
	return builder.String()
}
