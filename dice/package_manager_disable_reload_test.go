package dice //nolint:testpackage

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"sealdice-core/dice/sealpack"
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

func TestPackageManagerReloadAllAppliesUninstalledDeckPackage(t *testing.T) {
	for _, mode := range []sealpack.UninstallMode{
		sealpack.UninstallModeFull,
		sealpack.UninstallModeKeepData,
	} {
		t.Run(string(mode), func(t *testing.T) {
			testDice, pm := newDisableDeckTestPackageManager(t)
			if err := pm.Init(); err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			const pkgID = "alice/deck-pack"
			installAndReloadDeckTestPackage(t, testDice, pm, pkgID, "1.0.0")

			if err := pm.Uninstall(pkgID, mode); err != nil {
				t.Fatalf("Uninstall(%s) error = %v", mode, err)
			}
			if _, ok := pm.Get(pkgID); ok {
				t.Fatalf("expected package %s to be removed", pkgID)
			}

			result, err := pm.ReloadAll()
			if err != nil {
				t.Fatalf("ReloadAll() error = %v", err)
			}
			if _, ok := result.ReloadedItems["decks"]; !ok {
				t.Fatalf("ReloadedItems = %#v, want decks", result.ReloadedItems)
			}
			assertDisableDeckTestDeckCount(t, testDice, 0)

			result, err = pm.ReloadAll()
			if err != nil {
				t.Fatalf("second ReloadAll() error = %v", err)
			}
			if len(result.ReloadedItems) != 0 {
				t.Fatalf("second ReloadedItems = %#v, want empty", result.ReloadedItems)
			}
		})
	}
}

func TestPackageManagerReloadAllAppliesDiskRemovedDeckPackage(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID = "alice/disk-removed-deck-pack"
	installAndReloadDeckTestPackage(t, testDice, pm, pkgID, "1.0.0")

	pkg, ok := pm.Get(pkgID)
	if !ok || pkg == nil {
		t.Fatalf("expected package %s to exist", pkgID)
	}
	if err := os.Remove(pkg.SourcePath); err != nil {
		t.Fatalf("Remove(source) error = %v", err)
	}
	if err := os.RemoveAll(pkg.InstallPath); err != nil {
		t.Fatalf("RemoveAll(install) error = %v", err)
	}

	refreshResult, err := pm.RefreshFromDisk()
	if err != nil {
		t.Fatalf("RefreshFromDisk() error = %v", err)
	}
	if !containsString(refreshResult.Removed, pkgID) {
		t.Fatalf("Removed = %#v, want %s", refreshResult.Removed, pkgID)
	}

	reloadResult, err := pm.ReloadAll()
	if err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}
	if _, ok := reloadResult.ReloadedItems["decks"]; !ok {
		t.Fatalf("ReloadedItems = %#v, want decks", reloadResult.ReloadedItems)
	}
	assertDisableDeckTestDeckCount(t, testDice, 0)
}

func TestPackageManagerReloadAllAppliesPreviousContentsAfterInstallUpgrade(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID = "alice/install-upgrade-deck-pack"
	installAndReloadDeckTestPackage(t, testDice, pm, pkgID, "1.0.0")
	v2 := createTestSealPack(t, "", pkgID, "2.0.0", nil, nil)
	if err := pm.Install(v2); err != nil {
		t.Fatalf("Install(v2) error = %v", err)
	}

	result, err := pm.ReloadAll()
	if err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}
	if _, ok := result.ReloadedItems["decks"]; !ok {
		t.Fatalf("ReloadedItems = %#v, want decks", result.ReloadedItems)
	}
	assertDisableDeckTestDeckCount(t, testDice, 0)
}

func TestPackageManagerReloadAllAppliesPreviousContentsAfterRefreshUpgrade(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	const pkgID = "alice/refresh-upgrade-deck-pack"
	installAndReloadDeckTestPackage(t, testDice, pm, pkgID, "1.0.0")
	v2 := createTestSealPack(t, "", pkgID, "2.0.0", nil, nil)
	copyTestFile(t, v2, filepath.Join("data", "packages", "alice", "refresh-upgrade-deck-pack@2.0.0.sealpack"))

	refreshResult, err := pm.RefreshFromDisk()
	if err != nil {
		t.Fatalf("RefreshFromDisk() error = %v", err)
	}
	if !containsString(refreshResult.Updated, pkgID) {
		t.Fatalf("Updated = %#v, want %s", refreshResult.Updated, pkgID)
	}

	reloadResult, err := pm.ReloadAll()
	if err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}
	if _, ok := reloadResult.ReloadedItems["decks"]; !ok {
		t.Fatalf("ReloadedItems = %#v, want decks", reloadResult.ReloadedItems)
	}
	assertDisableDeckTestDeckCount(t, testDice, 0)
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

func installAndReloadDeckTestPackage(t *testing.T, testDice *Dice, pm *PackageManager, pkgID, version string) {
	t.Helper()
	archive := createDisableDeckTestPackage(t, pkgID, version)
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if _, err := pm.Reload(pkgID); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	assertDisableDeckTestDeckCount(t, testDice, 1)
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
	tempDir := filepath.Join(".", "temp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", tempDir, err)
	}
	archivePath := filepath.Join(tempDir, strings.ReplaceAll(pkgID, "/", "-")+"-"+version+".sealpack")

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
	pkg := &sealpack.Instance{PendingReload: []string{"scripts", "helpdoc", "templates"}}

	changed := pm.clearPendingReloadLocked(pkg, packageReloadContentFlags{scripts: true, templates: true})
	if !changed {
		t.Fatal("expected pending reload to change")
	}
	if len(pkg.PendingReload) != 1 || pkg.PendingReload[0] != "helpdoc" {
		t.Fatalf("PendingReload = %#v", pkg.PendingReload)
	}
}

func TestPackageReloadContentFlagsFromPreviousInstanceCoversAllKinds(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		contents sealpack.Contents
	}{
		{name: "scripts", kind: "scripts", contents: sealpack.Contents{Scripts: []string{"scripts/*.js"}}},
		{name: "decks", kind: "decks", contents: sealpack.Contents{Decks: []string{"decks/*.json"}}},
		{name: "reply", kind: "reply", contents: sealpack.Contents{Reply: []string{"reply/*.yaml"}}},
		{name: "helpdoc", kind: "helpdoc", contents: sealpack.Contents{Helpdoc: []string{"helpdoc/*.json"}}},
		{name: "templates", kind: "templates", contents: sealpack.Contents{Templates: []string{"templates/*.yaml"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := packageReloadContentFlagsFromPreviousInstance(&sealpack.Instance{
				State: sealpack.PackageStateEnabled,
				Manifest: &sealpack.Manifest{
					Contents: test.contents,
				},
			})
			if !flags.contains(test.kind) || flags.count() != 1 {
				t.Fatalf("flags = %#v, want only %s", flags, test.kind)
			}
		})
	}
}

func TestPackageReloadContentFlagsFromPreviousDisabledInstanceUsesPendingHints(t *testing.T) {
	flags := packageReloadContentFlagsFromPreviousInstance(&sealpack.Instance{
		State: sealpack.PackageStateDisabled,
		Manifest: &sealpack.Manifest{
			Contents: sealpack.Contents{
				Scripts: []string{"scripts/*.js"},
				Decks:   []string{"decks/*.json"},
			},
		},
		PendingReload: []string{"牌堆 - 可通过重载接口生效"},
	})
	if !flags.decks || flags.count() != 1 {
		t.Fatalf("flags = %#v, want only decks", flags)
	}
}

func TestClearDetachedReloadScopeLockedPreservesUnreloadedKinds(t *testing.T) {
	pm := &PackageManager{
		detachedReload:    packageReloadContentFlags{scripts: true, helpdoc: true},
		detachedReloadGen: 1,
	}

	pm.clearDetachedReloadScopeLocked(packageReloadContentFlags{scripts: true}, 1)
	if pm.detachedReload.scripts || !pm.detachedReload.helpdoc {
		t.Fatalf("detachedReload = %#v, want only helpdoc", pm.detachedReload)
	}
}

func TestClearDetachedReloadScopeLockedPreservesNewerGeneration(t *testing.T) {
	pm := &PackageManager{
		detachedReload:    packageReloadContentFlags{scripts: true},
		detachedReloadGen: 1,
	}

	startedGeneration := pm.detachedReloadGen
	pm.detachedReload = pm.detachedReload.merge(packageReloadContentFlags{decks: true})
	pm.detachedReloadGen++
	pm.clearDetachedReloadScopeLocked(packageReloadContentFlags{scripts: true}, startedGeneration)

	if !pm.detachedReload.scripts || !pm.detachedReload.decks {
		t.Fatalf("detachedReload = %#v, want scripts and decks", pm.detachedReload)
	}
}

func TestPackageManagerReloadAllPreservesFailedDetachedScope(t *testing.T) {
	testDice, pm := newDisableDeckTestPackageManager(t)
	if testDice.Parent != nil {
		t.Fatal("测试要求 Dice.Parent 为空，以确保帮助文档重载因缺少 DiceManager 而失败")
	}
	// 牌堆应重载成功；帮助文档因缺少 DiceManager 失败，用于验证只保留失败的重载范围。
	pm.detachedReload = packageReloadContentFlags{decks: true, helpdoc: true}
	pm.detachedReloadGen = 1

	result, err := pm.ReloadAll()
	if err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}
	if result.Success {
		t.Fatalf("ReloadAll() Success = true, want false: %#v", result.ReloadedItems)
	}
	if got := result.ReloadedItems["decks"]; got != "牌堆已重载" {
		t.Fatalf("ReloadAll() decks result = %q, want successful reload", got)
	}
	if got := result.ReloadedItems["helpdoc"]; !strings.Contains(got, "help manager is unavailable") {
		t.Fatalf("ReloadAll() helpdoc result = %q, want missing help manager failure", got)
	}
	if pm.detachedReload.decks || !pm.detachedReload.helpdoc {
		t.Fatalf("detachedReload = %#v, want only helpdoc", pm.detachedReload)
	}
}

func TestPackageManagerReloadAllExecutesDetachedScriptScope(t *testing.T) {
	testDice, pm := newScriptReloadTestPackageManager(t)
	pm.detachedReload = packageReloadContentFlags{scripts: true}
	pm.detachedReloadGen = 1

	result, err := pm.ReloadAll()
	if err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}
	if _, ok := result.ReloadedItems["scripts"]; !ok {
		t.Fatalf("ReloadedItems = %#v, want scripts", result.ReloadedItems)
	}
	if pm.detachedReload.scripts {
		t.Fatal("successful script reload should clear detached scope")
	}
	if testDice.ExtLoopManager == nil {
		t.Fatal("script reload should initialize the JS loop manager")
	}
}

func newScriptReloadTestPackageManager(t *testing.T) (*Dice, *PackageManager) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	testDice := &Dice{
		BaseConfig: BaseConfig{DataDir: "."},
		Logger:     zap.NewNop().Sugar(),
		ImSession: &IMSession{
			ServiceAtNew: new(SyncMap[string, *GroupInfo]),
			EndPoints:    []*EndPointInfo{},
		},
		DirtyGroups:        new(SyncMap[string, int64]),
		AttrsManager:       &AttrsManager{},
		JsBuiltinDigestSet: map[string]bool{},
	}
	testDice.Config = NewConfig(testDice)
	testDice.ConfigManager = NewConfigManager(filepath.Join(tmpDir, "plugin-configs.json"))
	pm := NewPackageManager(testDice)
	testDice.PackageManager = pm

	t.Cleanup(func() {
		if testDice.JsScriptCron != nil {
			testDice.JsScriptCron.Stop()
			testDice.JsScriptCron = nil
		}
		if testDice.ExtLoopManager != nil {
			testDice.ExtLoopManager.SetLoop(nil)
		}
	})
	return testDice, pm
}
