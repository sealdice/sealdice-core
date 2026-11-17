package dice

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// Each draw leases a catalog until formatting ends. Recursive draws, including
// shallow context copies, inherit the version of their active parent draw.
// Writers publish once and retire the old catalog. No reader holds deckMu while
// fetching cloud data or evaluating a template.
type deckCatalog struct {
	version  uint64
	decks    []*DeckInfo
	commands DeckCommandListItems
	refs     int // guarded by Dice.deckMu
}

// Context copies may keep a scope after its draw returns. The scope then becomes
// inactive and cannot lease its retired catalog again. Each child draw owns its
// own scope so it can finish (and recurse) even after the parent returns. The
// owner can read catalog until release; inheritance and invalidation use deckMu.
type deckDrawScope struct {
	catalog *deckCatalog // nil after the draw returns
}

var deckContentSequence atomic.Uint64

func newDeckInfo() *DeckInfo {
	return &DeckInfo{DeckItems: map[string][]string{}, Command: map[string]bool{}, CloudDeckItemInfos: map[string]*CloudDeckItemInfo{}, contentVersion: deckContentSequence.Add(1)}
}

func (d *Dice) compactDeck(deck *DeckInfo) {
	if !deck.Enable {
		return
	}
	dir := filepath.Join(os.TempDir(), "sealdice-decks")
	var storage *deckBlockStore
	var err error
	if cache := d.ensureDeckDiskCache(); cache != nil {
		storage, err = buildMappedDeck(cache.dir, deck.DeckItems)
		if err == nil {
			cache.register(storage)
		} else {
			d.Logger.Warnf("牌堆 %s 持久缓存写入失败，使用临时存储: %v", deck.Filename, err)
		}
	}
	if storage == nil {
		storage, err = buildMappedDeck(dir, deck.DeckItems)
	}
	if err != nil {
		d.Logger.Warnf("牌堆 %s 压缩存储失败，保留内存牌堆: %v", deck.Filename, err)
		return
	}
	deck.storage = storage
	deck.DeckItems = nil
	deck.RawData = nil
}

func (d *Dice) readDeck(fn, packageID string) *DeckInfo {
	if cache := d.ensureDeckDiskCache(); cache != nil {
		fingerprint, size, err := deckFileFingerprint(fn)
		if err == nil {
			var cached *DeckInfo
			cached, err = cache.restore(fn, packageID, fingerprint, size)
			if err == nil {
				return cached
			}
		}
		d.Logger.Debugf("牌堆 %s: %s", fn, cacheMissDiagnostic(err))
	}
	content, err := os.ReadFile(fn) // #nosec G703 -- fn is selected by deck discovery or the explicit deck loader.
	if err != nil {
		d.Logger.Warnf("读取牌堆 %s: %v", fn, err)
		return nil
	}
	deck := newDeckInfo()
	// hujson may standardize its input in place; fingerprint source bytes first.
	deck.setSourceFingerprint(content)
	_ = parseDeck(d, fn, content, deck)
	deck.Filename, deck.PackageID = fn, packageID
	if deck.Name == "" {
		deck.Name = filepath.Base(fn)
	}
	d.compactDeck(deck)
	d.saveDeckDiskCache(deck)
	return deck
}

func (deck *DeckInfo) setSourceFingerprint(content []byte) {
	digest := sha256.Sum256(content)
	deck.cacheSourceSHA = hex.EncodeToString(digest[:])
	deck.cacheSourceBytes = int64(len(content))
}

func closeDecks(d *Dice, decks []*DeckInfo) {
	for _, deck := range decks {
		if deck.storage != nil {
			if err := deck.storage.close(); err != nil {
				d.Logger.Warnf("释放牌堆缓存: %v", err)
			}
		}
	}
}

func retainDecks(decks []*DeckInfo) {
	for _, deck := range decks {
		if deck.storage != nil {
			deck.storage.retain()
		}
	}
}

// The caller transfers one ownership reference per mapped deck to the catalog.
// All writers serialize on deckLoadMu.
func (d *Dice) publishDecks(decks []*DeckInfo) {
	next := &deckCatalog{decks: decks, refs: 1, version: deckContentSequence.Add(1)}
	for _, deck := range decks {
		for name, visible := range deck.Command {
			next.commands = append(next.commands, &DeckCommandListItem{command: name, deck: deck, visible: visible})
		}
	}
	d.deckMu.Lock()
	old := d.deckCatalog
	d.deckCatalog = next
	d.DeckList = decks
	d.deckCommandItemsList = next.commands
	var retired []*DeckInfo
	if old != nil {
		old.refs--
		if old.refs == 0 {
			retired = old.decks
		}
	}
	d.deckMu.Unlock()
	closeDecks(d, retired)
	if d.StoreManager != nil {
		installed := map[string]bool{}
		for _, deck := range decks {
			if deck.StoreID != "" {
				installed[deck.StoreID] = true
			}
		}
		d.StoreManager.InstalledDecks = installed
	}
	d.MarkModified()
}

func (d *Dice) acquireDeckScope(parent *deckDrawScope) *deckDrawScope {
	d.deckMu.Lock()
	defer d.deckMu.Unlock()
	var catalog *deckCatalog
	if parent != nil && parent.catalog != nil {
		catalog = parent.catalog
	} else {
		if d.deckCatalog == nil {
			d.deckCatalog = &deckCatalog{decks: d.DeckList, refs: 1, version: deckContentSequence.Add(1)}
		}
		catalog = d.deckCatalog
	}
	catalog.refs++
	return &deckDrawScope{catalog: catalog}
}

func (d *Dice) releaseDeckScope(scope *deckDrawScope) {
	d.deckMu.Lock()
	c := scope.catalog
	scope.catalog = nil
	c.refs--
	retired := c.refs == 0
	d.deckMu.Unlock()
	if retired {
		closeDecks(d, c.decks)
	}
}

// DeckListSnapshot returns immutable metadata; it does not lease text mappings.
func (d *Dice) DeckListSnapshot() []*DeckInfo {
	d.deckMu.RLock()
	defer d.deckMu.RUnlock()
	return append([]*DeckInfo(nil), d.DeckList...)
}

// Writers hold deckLoadMu while copying/retaining the current runtime decks.
// After CloseDecks the saved metadata remains, but its mappings cannot be reused.
func (d *Dice) activeDecksSnapshot() []*DeckInfo {
	d.deckMu.RLock()
	defer d.deckMu.RUnlock()
	if d.deckCatalog != nil {
		return append([]*DeckInfo(nil), d.deckCatalog.decks...)
	}
	return append([]*DeckInfo(nil), d.DeckList...)
}

func (d *Dice) deckCommandsSnapshot() DeckCommandListItems {
	d.deckMu.RLock()
	defer d.deckMu.RUnlock()
	return d.deckCommandItemsList
}

func (d *Dice) SetDeckEnabled(filename string, enabled bool) {
	d.deckLoadMu.Lock()
	defer d.deckLoadMu.Unlock()
	decks := d.activeDecksSnapshot()
	for i, deck := range decks {
		if deck.Filename == filename {
			copyDeck := *deck
			copyDeck.Enable = enabled
			decks[i] = &copyDeck
		}
	}
	retainDecks(decks)
	d.publishDecks(decks)
}

// CloseDecks retires runtime resources without changing saved deck metadata or
// installed state. In-flight draws release their mappings on exit.
func (d *Dice) CloseDecks() {
	d.deckLoadMu.Lock()
	defer d.deckLoadMu.Unlock()
	d.deckMu.Lock()
	old := d.deckCatalog
	// Keep an empty runtime catalog so subsequent draws cannot lazily reopen the
	// closed mappings still referenced by the saved metadata.
	d.deckCatalog = &deckCatalog{refs: 1, version: deckContentSequence.Add(1)}
	d.deckCommandItemsList = nil
	var retired []*DeckInfo
	if old != nil {
		old.refs--
		if old.refs == 0 {
			retired = old.decks
		}
	}
	d.deckMu.Unlock()
	closeDecks(d, retired)
	d.closeDeckDiskCache()
}

// Stage in the source directory so a failed write cannot truncate the valid
// source. A successful replacement is followed by catalog publication.
func replaceDeckSource(filename string, data []byte) error {
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".deck-update-*")
	if err != nil {
		return err
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	if err = f.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filename); err != nil {
		return fmt.Errorf("替换牌堆源文件: %w", err)
	}
	return nil
}
