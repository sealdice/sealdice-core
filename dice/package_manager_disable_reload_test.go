package dice //nolint:testpackage

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"sealdice-core/dice/sealpkg"
)

func TestPackageManagerDisableDeckPackageReload(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID = "alice/deck-pack"
	archive := createDisableDeckTestPackage(t, pkgID, "1.0.0")
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	pkg, ok := pm.Get(pkgID)
	if !ok || pkg == nil {
		t.Fatalf("expected package %s to exist", pkgID)
	}
	if _, err := os.Stat(pkg.InstallPath); err != nil {
		t.Fatalf("expected install cache to exist: %v", err)
	}

	if _, err := pm.Reload(pkgID); err != nil {
		t.Fatalf("Reload(enabled) error = %v", err)
	}
	assertDisableDeckTestDeckCount(t, testDice, 1)

	if _, err := pm.Disable(pkgID); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if _, err := os.Stat(pkg.InstallPath); !os.IsNotExist(err) {
		t.Fatalf("expected install cache to be removed, stat err = %v", err)
	}

	if _, err := pm.Reload(pkgID); err != nil {
		t.Fatalf("Reload(disabled) error = %v", err)
	}
	assertDisableDeckTestDeckCount(t, testDice, 0)

	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable(after disable) error = %v", err)
	}
	if _, err := os.Stat(pkg.InstallPath); err != nil {
		t.Fatalf("expected install cache to be restored: %v", err)
	}

	if _, err := pm.Reload(pkgID); err != nil {
		t.Fatalf("Reload(re-enabled) error = %v", err)
	}
	assertDisableDeckTestDeckCount(t, testDice, 1)
}

func TestPackageManagerReloadAllAppliesDisabledDeckPackage(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID = "alice/deck-pack"
	archive := createDisableDeckTestPackage(t, pkgID, "1.0.0")
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if _, err := pm.Reload(pkgID); err != nil {
		t.Fatalf("Reload(enabled) error = %v", err)
	}
	assertDisableDeckTestDeckCount(t, testDice, 1)

	pkg, _ := pm.Get(pkgID)
	if _, err := pm.Disable(pkgID); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if _, err := pm.ReloadAll(); err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}

	assertDisableDeckTestDeckCount(t, testDice, 0)
	if len(pkg.PendingReload) != 0 {
		t.Fatalf("expected pending reload to be cleared, got %#v", pkg.PendingReload)
	}
}

func TestPackageManagerReloadClearsPendingReloadAcrossPackagesOfSameKind(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID1 = "alice/deck-pack-a"
	const pkgID2 = "alice/deck-pack-b"
	for _, pkgID := range []string{pkgID1, pkgID2} {
		archive := createDisableDeckTestPackage(t, pkgID, "1.0.0")
		if err := pm.Install(archive); err != nil {
			t.Fatalf("Install(%s) error = %v", pkgID, err)
		}
		if _, err := pm.Enable(pkgID); err != nil {
			t.Fatalf("Enable(%s) error = %v", pkgID, err)
		}
	}

	pkg1, _ := pm.Get(pkgID1)
	pkg2, _ := pm.Get(pkgID2)
	if len(pkg1.PendingReload) == 0 || len(pkg2.PendingReload) == 0 {
		t.Fatalf("expected pending reload hints before reload, got %#v / %#v", pkg1.PendingReload, pkg2.PendingReload)
	}

	if _, err := pm.Reload(pkgID1); err != nil {
		t.Fatalf("Reload(%s) error = %v", pkgID1, err)
	}

	assertDisableDeckTestDeckCount(t, testDice, 2)
	if len(pkg1.PendingReload) != 0 || len(pkg2.PendingReload) != 0 {
		t.Fatalf("expected all deck package pending reload hints to be cleared, got %#v / %#v", pkg1.PendingReload, pkg2.PendingReload)
	}
}

func TestPackageManagerReloadByContentClearsPendingReloadAcrossPackages(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID1 = "alice/deck-pack-a"
	const pkgID2 = "alice/deck-pack-b"
	for _, pkgID := range []string{pkgID1, pkgID2} {
		archive := createDisableDeckTestPackage(t, pkgID, "1.0.0")
		if err := pm.Install(archive); err != nil {
			t.Fatalf("Install(%s) error = %v", pkgID, err)
		}
		if _, err := pm.Enable(pkgID); err != nil {
			t.Fatalf("Enable(%s) error = %v", pkgID, err)
		}
	}

	pkg1, _ := pm.Get(pkgID1)
	pkg2, _ := pm.Get(pkgID2)
	if _, err := pm.ReloadByContent("decks"); err != nil {
		t.Fatalf("ReloadByContent(decks) error = %v", err)
	}

	assertDisableDeckTestDeckCount(t, testDice, 2)
	if len(pkg1.PendingReload) != 0 || len(pkg2.PendingReload) != 0 {
		t.Fatalf("expected all deck package pending reload hints to be cleared, got %#v / %#v", pkg1.PendingReload, pkg2.PendingReload)
	}
}

func assertDisableDeckTestDeckCount(t *testing.T, testDice *Dice, want int) {
	t.Helper()
	if got := len(testDice.DeckList); got != want {
		t.Fatalf("deck count = %d, want %d", got, want)
	}
}

func newDisableDeckTestPackageManager(t *testing.T) (*Dice, *PackageManager) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	if err := os.MkdirAll(filepath.Join("data", "decks"), 0o755); err != nil {
		t.Fatalf("MkdirAll(data/decks) error = %v", err)
	}

	testDice := &Dice{
		BaseConfig: BaseConfig{DataDir: "."},
		Logger:     zap.NewNop().Sugar(),
	}
	pm := NewPackageManager(testDice)
	testDice.PackageManager = pm
	return testDice, pm
}

func createDisableDeckTestPackage(t *testing.T, pkgID, version string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), strings.ReplaceAll(pkgID, "/", "-")+"-"+version+".sealpkg")

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", archivePath, err)
	}
	defer archiveFile.Close()

	zipWriter := zip.NewWriter(archiveFile)
	infoWriter, err := zipWriter.Create("info.toml")
	if err != nil {
		t.Fatalf("Create(info.toml) error = %v", err)
	}
	if _, writeErr := infoWriter.Write([]byte(buildDisableDeckTestManifest(pkgID, version))); writeErr != nil {
		t.Fatalf("Write(info.toml) error = %v", writeErr)
	}

	deckWriter, err := zipWriter.Create("decks/test.json")
	if err != nil {
		t.Fatalf("Create(deck) error = %v", err)
	}
	if _, err := deckWriter.Write([]byte(`{"_title":["Pkg Deck"],"test":["A"]}`)); err != nil {
		t.Fatalf("Write(deck) error = %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close(zip) error = %v", err)
	}
	return archivePath
}

func buildDisableDeckTestManifest(pkgID, version string) string {
	return fmt.Sprintf(`[package]
id = %q
name = "Deck Test Package"
version = %q
authors = ["Tester"]
license = "MIT"
description = "test"

[contents]
decks = ["decks/*.json"]
`, pkgID, version)
}

func TestClearPendingReloadLockedPreservesUnreloadedKinds(t *testing.T) {
	pm := &PackageManager{}
	pkg := &sealpkg.Instance{PendingReload: []string{"scripts", "helpdoc", "templates"}}

	changed := pm.clearPendingReloadLocked(pkg, packageReloadContentFlags{scripts: true, templates: true})
	if !changed {
		t.Fatal("expected pending reload to change")
	}
	if len(pkg.PendingReload) != 1 || pkg.PendingReload[0] != "helpdoc" {
		t.Fatalf("PendingReload = %#v", pkg.PendingReload)
	}
}
