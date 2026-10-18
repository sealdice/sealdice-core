//nolint:testpackage // Test cache lifecycle through real loading/updating/drawing.
package dice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func cacheTestDice(t *testing.T) *Dice {
	t.Helper()
	d := &Dice{Logger: zap.NewNop().Sugar()}
	d.Config = NewConfig(d)
	t.Cleanup(d.CloseDecks)
	return d
}

func cacheTestManifest(t *testing.T, d *Dice, deck *DeckInfo) (string, *deckCacheManifest) {
	t.Helper()
	id, _, err := deckSourceIdentity(deck.Filename, deck.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	path := d.deckDiskCache.manifestPath(id)
	meta, err := readDeckCacheManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, meta
}

func cacheTestWriteMeta(t *testing.T, path string, meta *deckCacheManifest) {
	t.Helper()
	body, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	mmapWrite(t, path, string(body))
}

func cacheTestFileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

func cacheTestAssertUnlocked(t *testing.T, dir string) {
	t.Helper()
	cache, err := openDeckDiskCache(dir)
	if err != nil {
		t.Fatal("closed mapping still holds the cache lock", err)
	}
	if err := cache.releaseFile(""); err != nil {
		t.Fatal(err)
	}
}

func TestDeckDiskCacheClosePreservesConfiguration(t *testing.T) {
	d := mmapTestDice(t)
	d.BaseConfig.DataDir = t.TempDir()
	d.ImSession = &IMSession{Parent: d}
	db, err := newMockDatabaseOperator(filepath.Join(d.BaseConfig.DataDir, "save.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	d.AttrsManager = &AttrsManager{db: db}
	d.StoreManager = &StoreManager{InstalledDecks: map[string]bool{}}
	mmapWrite(t, "data/decks/a.json", `{"_storeID":["installed-deck"],"g":["A"]}`)
	DeckReload(d)
	old := d.DeckListSnapshot()[0].storage
	cacheDir := d.deckDiskCache.dir
	readSavedDecks := func() []*DeckInfo {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			DeckList []*DeckInfo `yaml:"deckList"`
		}
		if err := yaml.Unmarshal(body, &saved); err != nil {
			t.Fatal(err)
		}
		return saved.DeckList
	}
	d.Save(true)
	want := readSavedDecks()
	if len(want) != 1 || want[0].StoreID != "installed-deck" {
		t.Fatalf("invalid saved fixture: %+v", want)
	}
	d.CloseDecks()
	d.CloseDecks() // Runtime cleanup must be idempotent.
	if d.LastUpdatedTime != 0 {
		t.Error("closing runtime storage marked the configuration as modified")
	}
	if !d.StoreManager.InstalledDecks["installed-deck"] {
		t.Error("closing runtime storage cleared the installed state")
	}
	if !old.closed || len(d.deckCommandsSnapshot()) != 0 {
		t.Error("runtime resources were not closed")
	}
	cacheTestAssertUnlocked(t, cacheDir)
	// A later save, including one triggered by an unrelated configuration edit,
	// must still serialize the deck metadata.
	d.MarkModified()
	d.Save(true)
	if got := readSavedDecks(); !reflect.DeepEqual(got, want) {
		t.Fatalf("shutdown changed saved deck metadata: got %+v, want %+v", got, want)
	}
	if ok, _, err := deckDraw(&MsgContext{Dice: d}, "g", false); err != nil || ok {
		t.Fatalf("closed catalog was reactivated: exists=%v err=%v", ok, err)
	}
	DeckReload(d)
	if !d.DeckListSnapshot()[0].cacheHit || mmapDraw(t, &MsgContext{Dice: d}, "g", false) != "A" {
		t.Fatal("reload after close did not restore the reusable cache")
	}
}

func TestDeckDiskCacheInvalidReloadAfterClose(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/a.json"
	mmapWrite(t, filename, `{"g":["A"]}`)
	DeckReload(d)
	old := d.DeckListSnapshot()[0].storage
	cacheDir := d.deckDiskCache.dir
	d.CloseDecks()
	cacheTestAssertUnlocked(t, cacheDir)
	mmapWrite(t, filename, `{"g":`)
	DeckReload(d)
	deck := d.DeckListSnapshot()[0]
	if !old.closed || deck.Enable || deck.ErrText == "" || deck.storage != nil {
		t.Fatal("invalid reload revived a closed mapping from saved metadata")
	}
	if ok, text, err := deckDraw(&MsgContext{Dice: d}, "g", false); ok || err != nil {
		t.Fatalf("invalid deck is drawable: exists=%v text=%q err=%v", ok, text, err)
	}
	mmapWrite(t, filename, `{"g":["B"]}`)
	DeckReload(d)
	if got := mmapDraw(t, &MsgContext{Dice: d}, "g", false); got != "B" {
		t.Fatalf("valid reload failed: %q", got)
	}
}

func TestDeckDiskCacheRestartAndInvalidation(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/untitled.json"
	mmapWrite(t, filename, `{"_keys":["root"],"root":["{_leaf}"],"_leaf":["::0::BAD","::9::A"]}`)
	DeckReload(d)
	first := d.DeckListSnapshot()[0]
	file := first.storage.filename
	if first.cacheHit || !first.storage.persistent {
		t.Fatal("first import did not create a durable cache")
	}
	path, meta := cacheTestManifest(t, d, first)
	if meta.SourceSHA256 == "" || meta.SHA256 == "" || meta.Groups < 2 || meta.Blocks < 2 || meta.Entries < 3 {
		t.Fatalf("incomplete manifest: %+v", meta)
	}
	DeckReload(d)
	if deck := d.DeckListSnapshot()[0]; !deck.cacheHit || deck.storage.filename != file {
		t.Fatal("unchanged reload reimported")
	}
	d.CloseDecks()
	if _, err := os.Stat(file); err != nil {
		t.Fatal("shutdown removed reusable data", err)
	}
	d = cacheTestDice(t)
	DeckReload(d)
	deck := d.DeckListSnapshot()[0]
	if !deck.cacheHit || deck.Name != "untitled.json" || !reflect.DeepEqual(first.Command, deck.Command) {
		t.Fatal("restart metadata differs")
	}
	ctx := &MsgContext{Dice: d}
	for range 10 {
		if got := mmapDraw(t, ctx, "root", true); got != "A" {
			t.Fatal(got)
		}
	}
	// Same length and mtime must still invalidate by content, not timestamps.
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	mmapWrite(t, filename, `{"_keys":["root"],"root":["{_leaf}"],"_leaf":["::0::BAD","::9::B"]}`)
	if err := os.Chtimes(filename, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	DeckReload(d)
	if d.DeckListSnapshot()[0].cacheHit || mmapDraw(t, ctx, "root", true) != "B" {
		t.Fatal("stale fingerprint/pool")
	}
	owner := t
	for _, field := range []string{"parser", "format", "checksum", "bounds", "json", "disabled", "missing-commands"} {
		t.Run(field, func(t *testing.T) {
			deck := d.DeckListSnapshot()[0]
			path, meta = cacheTestManifest(t, d, deck)
			d.CloseDecks()
			switch field {
			case "disabled":
				meta.Metadata.Enable = false
			case "missing-commands":
				meta.Metadata.Command = nil
			case "parser":
				meta.Parser = "older-parser"
			case "format":
				meta.Format--
			case "checksum":
				meta.SHA256 = strings.Repeat("0", 64)
			case "bounds":
				// Valid checksum, invalid directory bounds: exercise structural validation.
				dataPath := filepath.Join(filepath.Dir(path), meta.File)
				data, err := os.ReadFile(dataPath)
				if err != nil {
					t.Fatal(err)
				}
				for i := len(data) - 16; i < len(data); i++ {
					data[i] = 255
				}
				if err = os.WriteFile(dataPath, data, 0o600); err != nil {
					t.Fatal(err)
				}
				meta.SHA256, _, err = deckFileFingerprint(dataPath)
				if err != nil {
					t.Fatal(err)
				}
			}
			cacheTestWriteMeta(t, path, meta)
			if field == "json" {
				mmapWrite(t, path, "{")
			}
			d = cacheTestDice(owner)
			DeckReload(d)
			if d.DeckListSnapshot()[0].cacheHit || mmapDraw(t, &MsgContext{Dice: d}, "root", false) != "B" {
				t.Fatal("invalid cache was not rebuilt")
			}
		})
	}
}

func TestDeckDiskCacheMetadataMatchesSource(t *testing.T) {
	d := mmapTestDice(t)
	fixtures := map[string]string{
		"plain":   "\ufeff" + `{"extension":{"normal":1},"g":["a",2,null,{},"b"],"_storeID":["store"],"_etag":["etag"]}`,
		"x.jsonc": `{/* comment */"g":["a","b",],}`,
		"x.yaml":  "name: Y\ncommand: g\ndefault: [a, b]\ninfo: [hello, world]\nprivate: [p]\n",
		"x.toml":  "[meta]\ntitle='T'\n[decks]\n__private=['p']\n[g]\nexport=true\nvisible=false\ncloud_extra=true\ndistinct=true\noptions=['local']\noptions_urls=['https://example.invalid/deck']\n",
	}
	for filename, body := range fixtures {
		mmapWrite(t, filepath.Join("data/decks", filename), body)
	}
	DeckReload(d)
	old := d.DeckListSnapshot()
	d.CloseDecks()
	d = cacheTestDice(t)
	DeckReload(d)
	for i, next := range d.DeckListSnapshot() {
		a, _ := json.Marshal(old[i])
		b, _ := json.Marshal(next)
		if !next.cacheHit || string(a) != string(b) || !reflect.DeepEqual(old[i].Info, next.Info) || !reflect.DeepEqual(old[i].CloudDeckItemInfos, next.CloudDeckItemInfos) {
			t.Fatalf("metadata differs for %s: %s / %s; hit=%v Info=%#v/%#v Cloud=%#v/%#v", next.Filename, a, b, next.cacheHit, old[i].Info, next.Info, old[i].CloudDeckItemInfos, next.CloudDeckItemInfos)
		}
		body, err := os.ReadFile(next.Filename)
		if err != nil {
			t.Fatal(err)
		}
		legacy := newDeckInfo()
		if !parseDeck(d, next.Filename, body, legacy) {
			t.Fatal("legacy rejection")
		}
		for key, want := range legacy.DeckItems {
			got, err := next.storage.materialize(key)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("restored %s/%s mismatch: %v", next.Filename, key, err)
			}
		}
	}
	// Package ownership is part of source identity, not borrowed from another cache.
	fn := "data/decks/plain"
	d.deckLoadMu.Lock()
	packaged := d.readDeck(fn, "package-a")
	d.deckLoadMu.Unlock()
	if packaged.cacheHit || packaged.PackageID != "package-a" {
		t.Fatal("cross-package cache reuse")
	}
	closeDecks(d, []*DeckInfo{packaged})
}

func TestDeckDiskCacheUpdateFailureAndRestart(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/a.json"
	mmapWrite(t, filename, `{"_storeID":["id"],"g":["A"]}`)
	DeckReload(d)
	first := d.DeckListSnapshot()[0]
	path, original := cacheTestManifest(t, d, first)
	// Deterministic source-write failure on both Windows and Unix.
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	mmapWrite(t, "update.tmp", `{"g":["B"]}`)
	if err := d.DeckUpdate(first, "update.tmp"); err == nil {
		t.Fatal("missing source replacement succeeded")
	}
	_, current := cacheTestManifest(t, d, first)
	originalJSON, _ := json.Marshal(original)
	currentJSON, _ := json.Marshal(current)
	if string(originalJSON) != string(currentJSON) || mmapDraw(t, &MsgContext{Dice: d}, "g", false) != "A" {
		t.Fatal("failed update changed committed version")
	}
	mmapWrite(t, filename, `{"_storeID":["id"],"g":["A"]}`)
	mmapWrite(t, "update.tmp", `{"g":["B"]}`)
	if err := d.DeckUpdate(first, "update.tmp"); err != nil {
		t.Fatal(err)
	}
	if d.DeckListSnapshot()[0].StoreID != "id" {
		t.Fatal("live omitted StoreID behavior changed")
	}
	meta, err := readDeckCacheManifest(path)
	if err != nil || meta.File == original.File {
		t.Fatal("update manifest not committed", err)
	}
	d.CloseDecks()
	d = cacheTestDice(t)
	DeckReload(d)
	if !d.DeckListSnapshot()[0].cacheHit || d.DeckListSnapshot()[0].StoreID != "" || mmapDraw(t, &MsgContext{Dice: d}, "g", false) != "B" {
		t.Fatal("restart differs from cold source parsing")
	}
}

func TestDeckDiskCacheRetirementDeletionAndLock(t *testing.T) {
	d := mmapTestDice(t)
	mmapWrite(t, "data/decks/a.json", `{"g":["A"]}`)
	mmapWrite(t, "data/decks/b.json", `{"h":["H"]}`)
	DeckReload(d)
	lease := d.acquireDeckScope(nil)
	defer func() {
		if lease.catalog != nil {
			d.releaseDeckScope(lease)
		}
	}()
	oldFile := lease.catalog.decks[0].storage.filename
	cacheDir := d.deckDiskCache.dir
	mmapWrite(t, "data/decks/a.json", `{"g":["B"]}`)
	DeckReload(d)
	if _, err := os.Stat(oldFile); err != nil {
		t.Fatal("active old mapping deleted", err)
	}
	ctx := &MsgContext{Dice: d, deckSnapshot: lease}
	if mmapDraw(t, ctx, "g", false) != "A" || mmapDraw(t, &MsgContext{Dice: d}, "g", false) != "B" {
		t.Fatal("mixed versions")
	}
	d.CloseDecks()
	other := cacheTestDice(t)
	DeckReload(other)
	if other.deckDiskCache != nil || other.DeckListSnapshot()[0].storage == nil || other.DeckListSnapshot()[0].storage.persistent {
		t.Fatal("cache lock not held by old reader / fallback failed")
	}
	var temporary []string
	for _, deck := range other.DeckListSnapshot() {
		if deck.storage == nil || deck.storage.persistent || !cacheTestFileExists(t, deck.storage.filename) {
			t.Fatal("temporary fallback mapping missing")
		}
		temporary = append(temporary, deck.storage.filename)
	}
	if len(temporary) != 2 {
		t.Fatalf("expected two temporary mappings, got %d", len(temporary))
	}
	other.CloseDecks()
	for _, path := range temporary {
		if cacheTestFileExists(t, path) {
			t.Fatal("temporary fallback mapping leaked")
		}
	}
	d.releaseDeckScope(lease)
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("retired data not removed", err)
	}
	DeckReload(other)
	if !other.DeckListSnapshot()[0].cacheHit {
		t.Fatal("cache lock not released")
	}
	DeckDelete(other, other.DeckListSnapshot()[0])
	if len(other.DeckListSnapshot()) != 1 {
		t.Fatal("partial deletion")
	}
	DeckDelete(other, other.DeckListSnapshot()[0])
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "cache.lock" {
			t.Fatal("last deletion left cache", entry.Name())
		}
	}
}

func TestDeckDiskCacheUnavailableUsesTemporaryMapping(t *testing.T) {
	d := mmapTestDice(t)
	blocked := filepath.Join(t.TempDir(), "file")
	mmapWrite(t, blocked, "not a directory")
	d.BaseConfig.DataDir = blocked
	mmapWrite(t, "data/decks/a.json", `{"g":["A"]}`)
	DeckReload(d)
	deck := d.DeckListSnapshot()[0]
	if deck.storage == nil || deck.storage.persistent || mmapDraw(t, &MsgContext{Dice: d}, "g", false) != "A" {
		t.Fatal("temporary fallback failed")
	}
	file := deck.storage.filename
	d.CloseDecks()
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("temporary file leaked", err)
	}
}

func TestDeckDiskCacheFailedReloadPreservesVersionAndPrunesOrphans(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/a.json"
	mmapWrite(t, filename, `{"g":["A"]}`)
	DeckReload(d)
	first := d.DeckListSnapshot()[0]
	path, _ := cacheTestManifest(t, d, first)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(d.deckDiskCache.dir, "deck-orphan.blocks")
	tempMeta := filepath.Join(d.deckDiskCache.dir, ".deck-meta-orphan.tmp")
	mmapWrite(t, orphan, "incomplete build")
	mmapWrite(t, tempMeta, "{")
	mmapWrite(t, filename, "{")
	DeckReload(d)
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) || d.DeckListSnapshot()[0] != first || mmapDraw(t, &MsgContext{Dice: d}, "g", false) != "A" {
		t.Fatal("failed reload lost valid version", err)
	}
	for _, file := range []string{orphan, tempMeta} {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Fatal("orphan not pruned", file, err)
		}
	}
	mmapWrite(t, filename, `{"g":["A"]}`)
	mmapWrite(t, "data/decks/empty.json", `{}`)
	DeckReload(d)
	if !d.DeckListSnapshot()[0].cacheHit {
		t.Fatal("valid preserved cache was not reused")
	}
	d.CloseDecks()
	d = cacheTestDice(t)
	DeckReload(d)
	for _, deck := range d.DeckListSnapshot() {
		if !deck.cacheHit {
			t.Fatal("empty or restored deck missed", deck.Filename)
		}
	}
}
