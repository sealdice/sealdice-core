package api //nolint:testpackage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"sealdice-core/dice"
)

func TestResolvePackageAssetPathRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := resolvePackageAssetPath(root, "../secret.png"); err == nil {
		t.Fatal("resolvePackageAssetPath() error = nil, want escape rejection")
	}
}

func TestPackageAssetServesInstalledFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: "."},
		Logger:     zap.NewNop().Sugar(),
	}
	pm := dice.NewPackageManager(testDice)
	testDice.PackageManager = pm
	if err := pm.Init(); err != nil {
		t.Fatalf("PackageManager.Init() error = %v", err)
	}
	manager := &dice.DiceManager{Dice: []*dice.Dice{testDice}}
	token := "test-token"
	manager.AccessTokens.Store(token, true)
	testDice.Parent = manager
	myDice = testDice
	dm = manager
	t.Cleanup(func() {
		myDice = nil
		dm = nil
	})

	const pkgID = "alice/icon-pack"
	archive := createReplyAPITestSealPack(t, pkgID, "1.0.0", map[string]string{
		"reply/main.yaml": "items: []",
		"assets/icon.png": "png-data",
	})
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/sd-api/package/asset?id=alice/icon-pack&path=assets/icon.png&token="+token, nil)
	rec := httptest.NewRecorder()
	if err := packageAsset(e.NewContext(req, rec)); err != nil {
		t.Fatalf("packageAsset() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "png-data" {
		t.Fatalf("body = %q, want png-data", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "sandbox") {
		t.Fatalf("Content-Security-Policy = %q", got)
	}
}
