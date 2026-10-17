package dice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	mmap "github.com/blevesearch/mmap-go"
	"github.com/gofrs/flock"
)

const deckCacheFormat = 3

const deckCacheCodec = "zstd-fastest-16KiB-v1"

// Bump when any source parser, weight interpretation or export rule changes.
const deckCacheParser = "510cee1-deck-parser-1"

type deckCacheManifest struct {
	Format       int                           `json:"cache_format"`
	Parser       string                        `json:"parser_version"`
	WordBits     int                           `json:"word_bits"`
	SourceID     string                        `json:"source_id"`
	Source       string                        `json:"source"`
	PackageID    string                        `json:"package_id"`
	SourceSHA256 string                        `json:"source_sha256"`
	SourceBytes  int64                         `json:"source_bytes"`
	File         string                        `json:"data_file"`
	SHA256       string                        `json:"data_sha256"`
	Bytes        int64                         `json:"data_bytes"`
	Created      string                        `json:"created_utc"`
	Codec        string                        `json:"codec"`
	BlockEntries int                           `json:"block_entries"`
	BlockBytes   int                           `json:"block_bytes"`
	IndexHash    string                        `json:"index_hash"`
	Groups       int                           `json:"groups"`
	Entries      int64                         `json:"entries"`
	Blocks       int64                         `json:"blocks"`
	Metadata     DeckInfo                      `json:"metadata"`
	Info         []string                      `json:"info"`
	CloudGroups  map[string]*CloudDeckItemInfo `json:"cloud_groups"`
}

// The OS lock is held until the owner and all in-flight mapped readers release.
// A second process/Dice using the same cache directory falls back to temporary
// storage. It cannot clean files still being used by the first owner.
type deckDiskCache struct {
	mu    sync.Mutex
	dir   string
	lock  *flock.Flock
	refs  int
	live  map[string]int
	heads map[string]string
}

func deckSourceIdentity(filename, packageID string) (id, source string, err error) {
	source, err = filepath.Abs(filename)
	if err != nil {
		return "", "", err
	}
	source = filepath.ToSlash(filepath.Clean(source))
	if runtime.GOOS == "windows" {
		source = strings.ToLower(source)
	}
	digest := sha256.Sum256([]byte(source + "\x00" + packageID))
	return hex.EncodeToString(digest[:]), source, nil
}

func deckFileFingerprint(filename string) (string, int64, error) {
	f, err := os.Open(filename) // #nosec G703 -- the loader selects the source path; cache metadata does not supply it.
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 64<<10)
	n, err := io.CopyBuffer(h, f, buf)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func openDeckDiskCache(dir string) (*deckDiskCache, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(dir, "cache.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, errors.New("deck cache directory is already in use")
	}
	return &deckDiskCache{dir: dir, lock: lock, refs: 1, live: map[string]int{}, heads: map[string]string{}}, nil
}

func (d *Dice) ensureDeckDiskCache() *deckDiskCache {
	if !d.deckCacheAttempted {
		d.deckCacheAttempted = true // Initialization policy only; callers hold deckLoadMu.
		base := d.BaseConfig.DataDir
		if base == "" {
			base = "data"
		}
		dir := filepath.Join(base, "cache", "decks")
		var err error
		d.deckDiskCache, err = openDeckDiskCache(dir)
		if err != nil {
			d.Logger.Warnf("牌堆持久缓存不可用，使用临时存储: %v", err)
		}
	}
	return d.deckDiskCache
}

func (c *deckDiskCache) register(s *deckBlockStore) {
	c.mu.Lock()
	c.refs++
	c.live[s.filename]++
	c.mu.Unlock()
	s.cache = c
}

func (c *deckDiskCache) kept(filename string) bool {
	for _, head := range c.heads {
		if head == filename {
			return true
		}
	}
	return false
}

func (c *deckDiskCache) releaseFile(filename string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if filename != "" {
		c.live[filename]--
		if c.live[filename] == 0 {
			delete(c.live, filename)
			if !c.kept(filename) {
				err = removeDeckCacheFile(filename)
			}
		}
	}
	c.refs--
	if c.refs == 0 {
		err = errors.Join(err, c.lock.Unlock())
	}
	return err
}

func removeDeckCacheFile(filename string) error {
	err := os.Remove(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (c *deckDiskCache) manifestPath(id string) string { return filepath.Join(c.dir, id+".meta.json") }

func readDeckCacheManifest(filename string) (*deckCacheManifest, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var manifest deckCacheManifest
	err = json.NewDecoder(f).Decode(&manifest)
	return &manifest, err
}

func validDeckCacheFilename(name string) bool {
	return name == filepath.Base(name) && strings.HasPrefix(name, "deck-") && strings.HasSuffix(name, ".blocks")
}

func (c *deckDiskCache) restore(filename, packageID, fingerprint string, size int64) (*DeckInfo, error) {
	id, source, err := deckSourceIdentity(filename, packageID)
	if err != nil {
		return nil, err
	}
	meta, err := readDeckCacheManifest(c.manifestPath(id))
	if err != nil {
		return nil, err
	}
	if meta.Format != deckCacheFormat || meta.Parser != deckCacheParser || meta.WordBits != strconv.IntSize ||
		meta.Codec != deckCacheCodec || meta.BlockEntries != deckBlockEntries || meta.BlockBytes != deckBlockBytes ||
		meta.SourceID != id || meta.Source != source || meta.PackageID != packageID || meta.SourceSHA256 != fingerprint || meta.SourceBytes != size {
		return nil, errors.New("cache source/content/parser version mismatch")
	}
	if !validDeckCacheFilename(meta.File) || !meta.Metadata.Enable || meta.Metadata.Command == nil {
		return nil, errors.New("invalid cache metadata")
	}
	path := filepath.Join(c.dir, meta.File)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() != meta.Bytes || info.Size() < 33 {
		return nil, errors.New("cache data length mismatch")
	}
	data, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}
	s := &deckBlockStore{filename: path, data: data, references: 1, persistent: true}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != meta.SHA256 {
		_ = data.Unmap()
		return nil, errors.New("cache data checksum mismatch")
	}
	if err = s.loadMappedIndex(); err != nil {
		_ = data.Unmap()
		return nil, err
	}
	if s.groupCount() != meta.Groups {
		_ = data.Unmap()
		return nil, errors.New("cache group count mismatch")
	}
	d := meta.Metadata
	d.Filename, d.PackageID = filename, packageID
	d.Info, d.CloudDeckItemInfos = meta.Info, meta.CloudGroups
	if d.CloudDeckItemInfos == nil {
		d.CloudDeckItemInfos = map[string]*CloudDeckItemInfo{}
	}
	d.storage = s
	d.contentVersion = deckContentSequence.Add(1)
	d.cacheSourceSHA, d.cacheSourceBytes, d.cacheHit = fingerprint, size, true
	d.sourceStoreID, d.sourceEtag = d.StoreID, d.Etag
	c.mu.Lock()
	c.heads[id] = path
	c.mu.Unlock()
	c.register(s)
	return &d, nil
}

// Publish the manifest only after the source update succeeds. Data files are
// immutable and unique; the old manifest stays valid until atomic replacement.
func (c *deckDiskCache) save(deck *DeckInfo) error {
	s := deck.storage
	if s == nil || s.cache != c || deck.cacheSourceSHA == "" {
		return nil
	}
	id, source, err := deckSourceIdentity(deck.Filename, deck.PackageID)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.filename, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	err = f.Sync()
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	digest := sha256.Sum256(s.data)
	metadata := *deck
	if metadata.Format == "Dice!" {
		metadata.StoreID, metadata.Etag = deck.sourceStoreID, deck.sourceEtag
	}
	meta := deckCacheManifest{Format: deckCacheFormat, Parser: deckCacheParser, WordBits: strconv.IntSize,
		SourceID: id, Source: source, PackageID: deck.PackageID, SourceSHA256: deck.cacheSourceSHA, SourceBytes: deck.cacheSourceBytes,
		File: filepath.Base(s.filename), SHA256: hex.EncodeToString(digest[:]), Bytes: int64(len(s.data)),
		Created: time.Now().UTC().Format(time.RFC3339Nano), Codec: deckCacheCodec, IndexHash: "xxhash64+original-key",
		BlockEntries: deckBlockEntries, BlockBytes: deckBlockBytes,
		Groups: s.groupCount(), Metadata: metadata, Info: deck.Info, CloudGroups: deck.CloudDeckItemInfos}
	for _, entry := range s.hashes {
		g, _ := s.mappedGroup(entry.offset)
		meta.Entries += int64(g.count)
		meta.Blocks += int64(g.blockCount())
	}
	tmp, err := os.CreateTemp(c.dir, ".deck-meta-*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = tmp.Close(); _ = os.Remove(tmp.Name()) }()
	if err = json.NewEncoder(tmp).Encode(&meta); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp.Name(), c.manifestPath(id)); err != nil {
		return err
	}
	c.mu.Lock()
	c.heads[id] = s.filename
	c.mu.Unlock()
	s.persistent = true
	return nil
}

// Only a complete reload establishes which sources remain. Shutdown does not
// prune manifests. Live old versions are removed by releaseFile after readers
// finish; orphaned builds are removed on the next reload, including after crash.
func (c *deckDiskCache) prune(decks []*DeckInfo) error {
	active := map[string]bool{}
	for _, deck := range decks {
		id, _, err := deckSourceIdentity(deck.Filename, deck.PackageID)
		if err != nil {
			return err
		}
		active[id] = true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	heads := map[string]string{}
	var errs []error
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".meta.json") {
			continue
		}
		id := strings.TrimSuffix(name, ".meta.json")
		if len(id) != 64 {
			continue
		}
		if _, err := hex.DecodeString(id); err != nil {
			continue
		}
		path := filepath.Join(c.dir, name)
		if !active[id] {
			errs = append(errs, removeDeckCacheFile(path))
			continue
		}
		meta, err := readDeckCacheManifest(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if meta.SourceID == id && validDeckCacheFilename(meta.File) {
			heads[id] = filepath.Join(c.dir, meta.File)
		}
	}
	c.heads = heads
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		isTemp := strings.HasPrefix(name, ".deck-meta-") && strings.HasSuffix(name, ".tmp")
		if !validDeckCacheFilename(name) && !isTemp {
			continue
		}
		path := filepath.Join(c.dir, name)
		if c.live[path] == 0 && !c.kept(path) {
			errs = append(errs, removeDeckCacheFile(path))
		}
	}
	return errors.Join(errs...)
}

func (d *Dice) saveDeckDiskCache(deck *DeckInfo) {
	if deck.storage != nil && deck.storage.cache != nil {
		if err := deck.storage.cache.save(deck); err != nil {
			d.Logger.Warnf("牌堆 %s 缓存元数据未提交，将在下次重新导入: %v", deck.Filename, err)
		}
	}
}

func (d *Dice) pruneDeckDiskCache(decks []*DeckInfo) {
	if d.deckDiskCache != nil {
		if err := d.deckDiskCache.prune(decks); err != nil {
			d.Logger.Warnf("清理牌堆缓存: %v", err)
		}
	}
}

func (d *Dice) closeDeckDiskCache() {
	if d.deckDiskCache != nil {
		if err := d.deckDiskCache.releaseFile(""); err != nil {
			d.Logger.Warnf("关闭牌堆缓存: %v", err)
		}
		d.deckDiskCache = nil
	}
	d.deckCacheAttempted = false
}

func cacheMissDiagnostic(err error) string {
	if errors.Is(err, os.ErrNotExist) {
		return "cache not present"
	}
	return fmt.Sprintf("cache invalid: %v", err)
}
