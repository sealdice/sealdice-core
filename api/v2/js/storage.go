package js

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"

	"sealdice-core/dice"
)

// extStorage wraps per-extension KV operations.
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

// allKeys walks the namespace in ascending key order using the cursor-based
// List API, so a big plugin KV never has to be materialized all at once.
func (s *extStorage) allKeys(ctx context.Context) ([]string, error) {
	var keys []string
	cursor := ""
	for {
		result, err := s.ext.Storage.List(ctx, store.ListOptions{Limit: 1000, Cursor: cursor})
		if err != nil {
			return nil, err
		}
		for _, item := range result.Keys {
			keys = append(keys, item.Name)
		}
		if result.ListComplete || result.Cursor == "" || result.Cursor == cursor {
			return keys, nil
		}
		cursor = result.Cursor
	}
}

// compileGlob translates a BuntDB-style key pattern (* and ?) into a regexp.
// Unlike path.Match it lets * span any character, including '/', which matches
// how plugin keys are addressed.
func compileGlob(pattern string) *regexp.Regexp {
	var builder strings.Builder
	builder.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			builder.WriteString(".*")
		case '?':
			builder.WriteString(".")
		default:
			builder.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	builder.WriteString("$")
	re, err := regexp.Compile(builder.String())
	if err != nil {
		return nil
	}
	return re
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

	all, err := s.allKeys(context.Background())
	if err != nil {
		return nil, err
	}
	matcher := compileGlob(keyword)
	matched := make([]string, 0, len(all))
	for _, key := range all {
		if matcher == nil {
			if key == keyword {
				matched = append(matched, key)
			}
			continue
		}
		if matcher.MatchString(key) {
			matched = append(matched, key)
		}
	}

	result := &DataListResp{Keys: []DataKV{}, Page: page, PageSize: pageSize, Total: len(matched)}
	// Guard against overflowing (page-1)*pageSize when the client sends a huge
	// page number; anything past the last page is simply empty.
	if page > 1 && page > (len(matched)+pageSize-1)/pageSize {
		return result, nil
	}
	start := (page - 1) * pageSize
	if start >= len(matched) {
		return result, nil
	}
	end := start + pageSize
	if end > len(matched) {
		end = len(matched)
	}

	for _, key := range matched[start:end] {
		record, found, err := s.ext.Storage.Get(context.Background(), key)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		value := string(record.Value)
		result.Keys = append(result.Keys, DataKV{
			Key: key, Value: value, IsJSON: json.Valid(record.Value),
		})
	}
	return result, nil
}

// getValue returns a single key-value pair.
func (s *extStorage) getValue(key string) (*DataKV, error) {
	record, found, err := s.ext.Storage.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("key不存在：" + key)
	}
	return &DataKV{
		Key:    key,
		Value:  string(record.Value),
		IsJSON: json.Valid(record.Value),
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
	for _, key := range keys {
		if err := s.ext.Storage.Delete(context.Background(), key); err != nil {
			return err
		}
	}
	return nil
}

// info returns the key count and the on-disk size of this plugin's large-value
// bodies. Small values live in the shared plugin database, so their size is not
// attributable to a single extension.
func (s *extStorage) info() (*DataInfoResp, error) {
	keys, err := s.allKeys(context.Background())
	if err != nil {
		return nil, err
	}

	// 大值正文是 plugin-files/<ext>/<hash>.bin 的扁平布局，读一层目录即可。
	var bodyBytes int64
	if entries, readErr := os.ReadDir(filepath.Join(s.d.BaseConfig.DataDir, "plugin-files", s.ext.Name)); readErr == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if info, statErr := entry.Info(); statErr == nil {
				bodyBytes += info.Size()
			}
		}
	}

	return &DataInfoResp{
		KeyCount:  len(keys),
		FileSize:  bodyBytes,
		CanShrink: false,
	}, nil
}

// shrink is a no-op: the plugin database runs with incremental auto-vacuum, so
// there is no per-extension file to compact anymore.
func (s *extStorage) shrink() error {
	return nil
}
