package js

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/tidwall/buntdb"

	"sealdice-core/dice"
	jsm "sealdice-core/api/v2/model/js"
)

// extStorage wraps per-extension BuntDB operations.
// It holds a *dice.ExtInfo so we don't look it up on every call.
type extStorage struct {
	d   *dice.Dice
	ext *dice.ExtInfo
}

// resolveStorage looks up the ExtInfo by plugin name and ensures
// its Storage DB is initialised.
func resolveStorage(d *dice.Dice, name string) (*extStorage, error) {
	if d.JsExtRegistry == nil {
		return nil, errors.New("JsExtRegistry 未初始化")
	}
	ext, ok := d.JsExtRegistry.Load(name)
	if !ok || ext == nil {
		return nil, errors.New("未找到扩展：" + name)
	}
	real := ext.GetRealExt()
	if real == nil {
		return nil, errors.New("扩展不存在：" + name)
	}
	if err := real.StorageInit(); err != nil {
		return nil, err
	}
	return &extStorage{d: d, ext: real}, nil
}

// listKeys returns a paginated subset of keys matching keyword (glob pattern).
func (s *extStorage) listKeys(page, pageSize int, keyword string) (*jsm.DataListResp, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if keyword == "" {
		keyword = "*"
	}

	type kv struct {
		Key   string
		Value string
	}

	var all []kv
	err := s.ext.Storage.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(keyword, func(key, value string) bool {
			all = append(all, kv{Key: key, Value: value})
			return true
		})
	})
	if err != nil {
		return nil, err
	}

	total := len(all)
	start := (page - 1) * pageSize
	if start >= total {
		return &jsm.DataListResp{Keys: []jsm.DataKV{}, Total: total, Page: page, PageSize: pageSize}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	keys := make([]jsm.DataKV, 0, end-start)
	for _, item := range all[start:end] {
		keys = append(keys, jsm.DataKV{
			Key:    item.Key,
			Value:  item.Value,
			IsJSON: json.Valid([]byte(item.Value)),
		})
	}

	return &jsm.DataListResp{Keys: keys, Total: total, Page: page, PageSize: pageSize}, nil
}

// getValue returns a single key-value pair.
func (s *extStorage) getValue(key string) (*jsm.DataKV, error) {
	val, err := s.ext.StorageGet(key)
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil, errors.New("key不存在：" + key)
		}
		return nil, err
	}
	return &jsm.DataKV{
		Key:    key,
		Value:  val,
		IsJSON: json.Valid([]byte(val)),
	}, nil
}

// setValue sets a key-value pair.
func (s *extStorage) setValue(key, value string) error {
	return s.ext.StorageSet(key, value)
}

// deleteKeys batch-deletes keys.
func (s *extStorage) deleteKeys(keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.ext.Storage.Update(func(tx *buntdb.Tx) error {
		for _, key := range keys {
			if _, err := tx.Delete(key); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return err
			}
		}
		return nil
	})
}

// info returns DB statistics.
func (s *extStorage) info() (*jsm.DataInfoResp, error) {
	var count int
	var canShrink bool
	var fileSize int64

	dbPath := filepath.Join(s.d.BaseConfig.DataDir, "extensions", s.ext.Name, "storage.db")
	fi, err := os.Stat(dbPath)
	if err == nil {
		fileSize = fi.Size()
		// BuntDB auto-shrink triggers at ~32MB; suggest manual shrink above 10MB
		canShrink = fileSize > 10*1024*1024
	}

	err = s.ext.Storage.View(func(tx *buntdb.Tx) error {
		var lerr error
		count, lerr = tx.Len()
		return lerr
	})
	if err != nil {
		return nil, err
	}

	return &jsm.DataInfoResp{
		KeyCount:  count,
		FileSize:  fileSize,
		CanShrink: canShrink,
	}, nil
}

// shrink compacts the storage database file.
func (s *extStorage) shrink() error {
	return s.ext.Storage.Shrink()
}
