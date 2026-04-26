package dice

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestMatchPackageContentPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "direct file", pattern: "scripts/main.js", path: "scripts/main.js", want: true},
		{name: "star", pattern: "scripts/*.js", path: "scripts/main.js", want: true},
		{name: "double star nested", pattern: "helpdoc/**/*.md", path: "helpdoc/group/topic/file.md", want: true},
		{name: "double star top level", pattern: "helpdoc/**/*.md", path: "helpdoc/file.md", want: true},
		{name: "mismatch dir", pattern: "reply/*.yaml", path: "scripts/main.js", want: false},
		{name: "mismatch ext", pattern: "templates/*.yaml", path: "templates/demo.json", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPackageContentPattern(tt.pattern, tt.path); got != tt.want {
				t.Fatalf("matchPackageContentPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestPackageManagerInitSelectsHighestVersion(t *testing.T) {
	testDice, pm := newTestPackageManager(t)
	_ = testDice

	pkgID := "alice/demo"
	v1 := createTestSealPkg(t, "", pkgID, "1.0.0", map[string][]string{
		"scripts": {"scripts/*.js"},
	}, map[string]string{
		"scripts/main.js": "// v1",
	})
	v2 := createTestSealPkg(t, "", pkgID, "2.0.0", map[string][]string{
		"scripts": {"scripts/*.js"},
	}, map[string]string{
		"scripts/main.js": "// v2",
	})

	destDir := filepath.Join("data", "packages", "alice", "demo")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	copyTestFile(t, v1, filepath.Join(destDir, "1.0.0.sealpkg"))
	copyTestFile(t, v2, filepath.Join(destDir, "2.0.0.sealpkg"))

	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	pkg, ok := pm.Get(pkgID)
	if !ok || pkg == nil || pkg.Manifest == nil {
		t.Fatalf("expected package %s to be loaded", pkgID)
	}
	if got := pkg.Manifest.Package.Version; got != "2.0.0" {
		t.Fatalf("loaded version = %q, want 2.0.0", got)
	}
	if !strings.HasSuffix(filepath.ToSlash(pkg.SourcePath), "data/packages/alice/demo/2.0.0.sealpkg") {
		t.Fatalf("SourcePath = %q", pkg.SourcePath)
	}
	if _, err := os.Stat(filepath.Join("cache", "packages", "alice", "demo", "info.toml")); err != nil {
		t.Fatalf("expected extracted cache manifest: %v", err)
	}
}

func TestPackageManagerInstallUpgradePreservesConfigAndUserData(t *testing.T) {
	_, pm := newTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	pkgID := "alice/demo"
	v1 := createTestSealPkg(t, "", pkgID, "1.0.0", map[string][]string{
		"scripts": {"scripts/*.js"},
	}, map[string]string{
		"scripts/main.js": "// v1",
	}, withConfigMode())
	v2 := createTestSealPkg(t, "", pkgID, "2.0.0", map[string][]string{
		"scripts": {"scripts/*.js"},
	}, map[string]string{
		"scripts/main.js": "// v2",
	}, withConfigMode())

	if err := pm.Install(v1); err != nil {
		t.Fatalf("Install(v1) error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if err := pm.SetConfig(pkgID, map[string]interface{}{"mode": "custom"}); err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}

	pkg, _ := pm.Get(pkgID)
	markerPath := filepath.Join(pkg.UserDataPath, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("userdata"), 0o644); err != nil {
		t.Fatalf("WriteFile(marker): %v", err)
	}
	oldSourcePath := pkg.SourcePath

	if err := pm.Install(v2); err != nil {
		t.Fatalf("Install(v2) error = %v", err)
	}
	pkg, _ = pm.Get(pkgID)
	if got := pkg.Manifest.Package.Version; got != "2.0.0" {
		t.Fatalf("upgraded version = %q, want 2.0.0", got)
	}
	if pkg.State != PackageStateEnabled {
		t.Fatalf("State = %q, want %q", pkg.State, PackageStateEnabled)
	}
	cfg, err := pm.GetConfig(pkgID)
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if got := cfg["mode"]; got != "custom" {
		t.Fatalf("config mode = %#v, want %q", got, "custom")
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected userdata marker to remain: %v", err)
	}
	if _, err := os.Stat(oldSourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected old source artifact to be removed, stat err = %v", err)
	}
	if err := pm.Install(v1); err == nil {
		t.Fatal("expected lower version install to fail")
	}
	if err := pm.Install(v2); err == nil {
		t.Fatal("expected same version install to fail")
	}
}

func TestPackageManagerGetEnabledContentFilesUsesDeclaredPatterns(t *testing.T) {
	_, pm := newTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	pkgID := "alice/content-pack"
	archive := createTestSealPkg(t, "", pkgID, "1.0.0", map[string][]string{
		"scripts": {"scripts/*.js"},
		"decks":   {"decks/**/*.json"},
		"reply":   {"reply/*.yaml"},
	}, map[string]string{
		"scripts/main.js":          "// keep",
		"scripts/ignored.ts":       "// skip",
		"scripts/nested/hidden.js": "// skip",
		"decks/main.json":          "{}",
		"decks/nested/extra.json":  "{}",
		"reply/main.yaml":          "replies: []",
		"helpdoc/readme.md":        "# not declared",
	})
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	scriptFiles := pm.GetEnabledContentFiles("scripts")
	assertContentFiles(t, scriptFiles, []string{"scripts/main.js"})

	deckFiles := pm.GetEnabledContentFiles("decks")
	assertContentFiles(t, deckFiles, []string{"decks/main.json", "decks/nested/extra.json"})

	replyFiles := pm.GetEnabledContentFiles("reply")
	assertContentFiles(t, replyFiles, []string{"reply/main.yaml"})

	helpFiles := pm.GetEnabledContentFiles("helpdoc")
	if len(helpFiles) != 0 {
		t.Fatalf("helpdoc files = %#v, want none", helpFiles)
	}
}

func TestPackageManagerReloadAllLoadsTemplateFilesFromEnabledPackages(t *testing.T) {
	testDice, pm := newTestPackageManager(t)
	testDice.GameSystemMap = new(SyncMap[string, *GameSystemTemplate])
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	pkgID := "alice/template-pack"
	archive := createTestSealPkg(t, "", pkgID, "1.0.0", map[string][]string{
		"templates": {"templates/*.yaml"},
	}, map[string]string{
		"templates/pkgtest.yaml": loadTemplateFixture(t, "pkgtest"),
	})
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	if _, exists := testDice.GameSystemMap.Load("pkgtest"); exists {
		t.Fatal("pkgtest should not be loaded before reload")
	}
	if _, err := pm.ReloadAll(); err != nil {
		t.Fatalf("ReloadAll() error = %v", err)
	}
	tmpl, exists := testDice.GameSystemMap.Load("pkgtest")
	if !exists {
		t.Fatal("expected pkgtest template to be loaded from enabled package files")
	}
	if got := strings.Join(tmpl.SetConfig.Keys, ","); got != "pkgtest,pkgtest-rule" {
		t.Fatalf("pkgtest set keys = %q, want %q", got, "pkgtest,pkgtest-rule")
	}
}

func TestPackageManagerListReturnsStablePackageOrder(t *testing.T) {
	_, pm := newTestPackageManager(t)
	if err := pm.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	archives := []struct {
		id      string
		version string
	}{
		{id: "zeta/pkg", version: "1.0.0"},
		{id: "alpha/pkg", version: "1.0.0"},
		{id: "middle/pkg", version: "1.0.0"},
	}
	for _, item := range archives {
		archive := createTestSealPkg(t, "", item.id, item.version, map[string][]string{
			"scripts": {"scripts/*.js"},
		}, map[string]string{
			"scripts/main.js": "// test",
		})
		if err := pm.Install(archive); err != nil {
			t.Fatalf("Install(%s) error = %v", item.id, err)
		}
	}

	list := pm.List()
	got := make([]string, 0, len(list))
	for _, pkg := range list {
		got = append(got, pkg.Manifest.Package.ID)
	}
	want := []string{"alpha/pkg", "middle/pkg", "zeta/pkg"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("List() order = %#v, want %#v", got, want)
	}
}

func TestVerifyDownloadedPackageHash(t *testing.T) {
	data := []byte("demo package")
	sum := sha256.Sum256(data)
	sha256Text := hex.EncodeToString(sum[:])

	tests := []struct {
		name    string
		hashes  map[string]string
		wantErr bool
	}{
		{name: "no hash", hashes: nil, wantErr: false},
		{name: "valid sha256", hashes: map[string]string{"sha256": sha256Text}, wantErr: false},
		{name: "valid sha256 case insensitive", hashes: map[string]string{"SHA256": strings.ToUpper(sha256Text)}, wantErr: false},
		{name: "invalid sha256", hashes: map[string]string{"sha256": strings.Repeat("0", 64)}, wantErr: true},
		{name: "blank sha256", hashes: map[string]string{"sha256": "   "}, wantErr: true},
		{name: "unsupported only", hashes: map[string]string{"sha512": "abc"}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyDownloadedPackageHash(data, tt.hashes)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUnsupportedPackageHashAlgorithms(t *testing.T) {
	got := unsupportedPackageHashAlgorithms(map[string]string{
		"sha512": "a",
		"SHA256": "b",
		"md5":    "c",
	})
	if strings.Join(got, ",") != "md5,sha512" {
		t.Fatalf("unsupportedPackageHashAlgorithms() = %#v", got)
	}
}

type manifestOption func(*strings.Builder)

func withConfigMode() manifestOption {
	return func(b *strings.Builder) {
		b.WriteString("\n[config.mode]\n")
		b.WriteString("type = \"string\"\n")
		b.WriteString("default = \"basic\"\n")
	}
}

func newTestPackageManager(t *testing.T) (*Dice, *PackageManager) {
	t.Helper()
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	d := &Dice{
		BaseConfig: BaseConfig{DataDir: "."},
		Logger:     zap.NewNop().Sugar(),
	}
	pm := NewPackageManager(d)
	d.PackageManager = pm
	return d, pm
}

func createTestSealPkg(t *testing.T, dir, pkgID, version string, contents map[string][]string, files map[string]string, opts ...manifestOption) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	archivePath := filepath.Join(dir, strings.ReplaceAll(pkgID, "/", "-")+"-"+version+".sealpkg")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", archivePath, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	infoWriter, err := zw.Create("info.toml")
	if err != nil {
		t.Fatalf("Create(info.toml) error = %v", err)
	}
	manifest := buildTestManifest(pkgID, version, contents, opts...)
	if _, err := infoWriter.Write([]byte(manifest)); err != nil {
		t.Fatalf("Write(info.toml) error = %v", err)
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("Write(%s) error = %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close(zip) error = %v", err)
	}
	return archivePath
}

func buildTestManifest(pkgID, version string, contents map[string][]string, opts ...manifestOption) string {
	var b strings.Builder
	b.WriteString("[package]\n")
	b.WriteString(fmt.Sprintf("id = %q\n", pkgID))
	b.WriteString("name = \"Test Package\"\n")
	b.WriteString(fmt.Sprintf("version = %q\n", version))
	b.WriteString("authors = [\"Tester\"]\n")
	b.WriteString("license = \"MIT\"\n")
	b.WriteString("description = \"test\"\n")
	if len(contents) > 0 {
		b.WriteString("\n[contents]\n")
		keys := make([]string, 0, len(contents))
		for key := range contents {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			b.WriteString(key)
			b.WriteString(" = [")
			for i, pattern := range contents[key] {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(fmt.Sprintf("%q", pattern))
			}
			b.WriteString("]\n")
		}
	}
	for _, opt := range opts {
		opt(&b)
	}
	return b.String()
}

func copyTestFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", dst, err)
	}
}

func assertContentFiles(t *testing.T, files []PackageContentFile, want []string) {
	t.Helper()
	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, filepath.ToSlash(file.PackagePath))
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("content files = %#v, want %#v", got, want)
	}
}

func loadTemplateFixture(t *testing.T, name string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "templates", "coc7.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(template fixture) error = %v", err)
	}
	content := strings.Replace(string(data), "name: coc7", "name: "+name, 1)
	content = strings.Replace(content, "fullName:", "fullName: package fixture\n#", 1)
	content = strings.Replace(content, "      - coc\n      - coc7", "      - "+name+"\n      - "+name+"-rule", 1)
	return content
}
