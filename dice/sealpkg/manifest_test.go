package sealpkg //nolint:testpackage

import "testing"

func TestParseManifestWithUnifiedFormat(t *testing.T) {
	data := []byte(`
format_version = "1.0.0"

[package]
id = "木落/奇幻牌堆"
name = "奇幻牌堆合集"
version = "1.2.3"
authors = ["木落"]
license = "MIT"
description = "测试包"

[package.seal]
min_version = "1.5.0"

[dependencies]
"海豹/基础库" = ">=1.0.0"

[contents]
scripts = ["scripts/*.js"]
decks = ["decks/**/*.json"]
reply = ["reply/*.yaml"]
helpdoc = ["helpdoc/*.md"]
templates = ["templates/*.yaml"]

[store]
readme = "README.md"
icon = "assets/icon.png"
screenshots = ["assets/shot-1.png"]

[config.mode]
type = "string"
default = "simple"
`)

	manifest, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if manifest.Package.ID != "木落/奇幻牌堆" {
		t.Fatalf("manifest.Package.ID = %q", manifest.Package.ID)
	}
	if manifest.FormatVersion != CurrentManifestFormatVersion {
		t.Fatalf("manifest.FormatVersion = %q", manifest.FormatVersion)
	}
	if got := len(manifest.Contents.Templates); got != 1 {
		t.Fatalf("manifest.Contents.Templates len = %d", got)
	}
	if manifest.Store.Icon != "assets/icon.png" {
		t.Fatalf("manifest.Store.Icon = %q", manifest.Store.Icon)
	}
}

func TestParseManifestDefaultsFormatVersion(t *testing.T) {
	data := []byte(`
[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"
`)

	manifest, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if manifest.FormatVersion != CurrentManifestFormatVersion {
		t.Fatalf("manifest.FormatVersion = %q", manifest.FormatVersion)
	}
}

func TestParseManifestRejectsUnsupportedFormatVersion(t *testing.T) {
	data := []byte(`
format_version = "999.0.0"

[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want unsupported format_version rejection")
	}
}

func TestParseManifestRejectsInvalidFormatVersion(t *testing.T) {
	data := []byte(`
format_version = "invalid"

[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want invalid format_version rejection")
	}
}

func TestParseManifestRejectsNumericFormatVersion(t *testing.T) {
	data := []byte(`
format_version = 1

[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want numeric format_version rejection")
	}
}

func TestParseManifestRejectsLegacySrcLayout(t *testing.T) {
	data := []byte(`
[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"

[contents]
scripts = ["src/scripts/*.js"]
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want rejection for src/ layout")
	}
}

func TestParseManifestAllowsExtensionFields(t *testing.T) {
	data := []byte(`
[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"
display_name = "Demo Package"

[package.extra]
homepage_name = "Project"

[contents]
scripts = ["scripts/*.js"]

[store]
icon = "assets/icon.png"
summary = "ignored"

[x-sealdice]
note = "ignored"
`)

	manifest, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if manifest.Package.ID != "alice/demo" {
		t.Fatalf("manifest.Package.ID = %q", manifest.Package.ID)
	}
	if manifest.Store.Icon != "assets/icon.png" {
		t.Fatalf("manifest.Store.Icon = %q", manifest.Store.Icon)
	}
}

func TestParseManifestRejectsAbsoluteStorePath(t *testing.T) {
	data := []byte(`
[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"

[store]
icon = "/assets/icon.png"
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want rejection for absolute store path")
	}
}
func TestParseManifestRejectsWindowsAbsoluteStorePath(t *testing.T) {
	data := []byte(`
[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"

[store]
icon = "C:/assets/icon.png"
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want rejection for Windows absolute store path")
	}
}

func TestParseManifestRejectsUnknownContentsKey(t *testing.T) {
	data := []byte(`
[package]
id = "alice/demo"
name = "Demo"
version = "1.0.0"

[contents]
template = ["templates/*.yaml"]
`)

	if _, err := ParseManifest(data); err == nil {
		t.Fatal("ParseManifest() error = nil, want rejection for unknown contents key")
	}
}
