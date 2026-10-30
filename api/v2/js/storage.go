package js

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/buntdb"

	"sealdice-core/dice"
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
		// 列表以脚本的 @name 展示，而注册表以 seal.ext.new/register 的名字为键，
		// 两者可能不同；通过脚本来源匹配到对应扩展。
		ext = findStorageExtensionByScript(d, name)
		ok = ext != nil
	}
	if !ok || ext == nil {
		return nil, errors.New("未找到扩展：" + name)
	}
	realExt := ext.GetRealExt()
	if realExt == nil {
		return nil, errors.New("扩展不存在：" + name)
	}
	if err := realExt.StorageInit(); err != nil {
		return nil, err
	}
	return &extStorage{d: d, ext: realExt}, nil
}

func findStorageExtensionByScript(d *dice.Dice, name string) *dice.ExtInfo {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for _, script := range d.JsScriptList {
		if script == nil || (script.Name != name && script.Filename != name) {
			continue
		}
		var found *dice.ExtInfo
		d.JsExtRegistry.Range(func(_ string, ext *dice.ExtInfo) bool {
			if ext == nil {
				return true
			}
			realExt := ext.GetRealExt()
			if realExt != nil && realExt.Source == script {
				found = realExt
				return false
			}
			return true
		})
		if found != nil {
			return found
		}
	}
	return nil
}

// listKeys returns a paginated subset of keys matching keyword (glob pattern).
func (s *extStorage) listKeys(page, pageSize int, keyword string) (*DataListResp, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if keyword == "" {
		keyword = "*"
	}

	result := &DataListResp{Keys: []DataKV{}, Page: page, PageSize: pageSize}
	err := s.ext.Storage.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(keyword, func(key, value string) bool {
			// Count all matches, but retain only the requested page. Division
			// avoids overflowing an offset when the client sends a huge page.
			if result.Total/pageSize == page-1 {
				result.Keys = append(result.Keys, DataKV{
					Key: key, Value: value, IsJSON: json.Valid([]byte(value)),
				})
			}
			result.Total++
			return true
		})
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// getValue returns a single key-value pair.
func (s *extStorage) getValue(key string) (*DataKV, error) {
	val, err := s.ext.StorageGet(key)
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil, errors.New("key不存在：" + key)
		}
		return nil, err
	}
	return &DataKV{
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
func (s *extStorage) info() (*DataInfoResp, error) {
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

	return &DataInfoResp{
		KeyCount:  count,
		FileSize:  fileSize,
		CanShrink: canShrink,
	}, nil
}

// shrink compacts the storage database file.
func (s *extStorage) shrink() error {
	return s.ext.Storage.Shrink()
}
