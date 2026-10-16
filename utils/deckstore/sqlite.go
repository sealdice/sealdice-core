package deckstore

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"

	"sealdice-core/utils/dboperator/engine/sqlite"
)

// schemaDDL 是三张表的建表语句（照规格 §4）。
var schemaDDL = []string{
	`CREATE TABLE IF NOT EXISTS deck_meta (
		deck_key TEXT NOT NULL,
		k        TEXT NOT NULL,
		v        BLOB NOT NULL,
		PRIMARY KEY (deck_key, k)
	)`,
	`CREATE TABLE IF NOT EXISTS deck_group (
		deck_key     TEXT NOT NULL,
		group_name   TEXT NOT NULL,
		item_count   INTEGER NOT NULL,
		total_weight INTEGER NOT NULL,
		weighted     INTEGER NOT NULL,
		chunk_count  INTEGER NOT NULL,
		chunk_starts BLOB NOT NULL,
		weight_blob  BLOB,
		PRIMARY KEY (deck_key, group_name)
	)`,
	`CREATE TABLE IF NOT EXISTS deck_chunk (
		deck_key   TEXT NOT NULL,
		group_name TEXT NOT NULL,
		chunk_no   INTEGER NOT NULL,
		blob       BLOB NOT NULL,
		PRIMARY KEY (deck_key, group_name, chunk_no)
	)`,
}

// sqliteStore 是 Store 的 SQLite 实现。
type sqliteStore struct {
	reader *gorm.DB
	writer *gorm.DB
	log    Logger

	mu    sync.RWMutex
	index map[string]GroupMeta // 键 = Key(deckID, group)

	chunkLRU  *byteLRU[[]string]
	weightLRU *byteLRU[[]uint32]
}

func openSQLite(path string, log Logger) (Store, error) {
	reader, writer, err := sqlite.SQLiteDBRWInit(path)
	if err != nil {
		return nil, err
	}
	s := &sqliteStore{
		reader:    reader,
		writer:    writer,
		log:       log,
		index:     make(map[string]GroupMeta),
		chunkLRU:  newByteLRU[[]string](ChunkCacheCap),
		weightLRU: newByteLRU[[]uint32](WeightCacheCap),
	}
	if err := s.ensureSchema(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func openSQLiteMemory(log Logger) (Store, error) {
	db, err := sqlite.SQLiteDBInit(":memory:", false)
	if err != nil {
		return nil, err
	}
	// :memory: 下每个连接各自为库，必须钉在单连接上。
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)

	s := &sqliteStore{
		reader:    db,
		writer:    db,
		log:       log,
		index:     make(map[string]GroupMeta),
		chunkLRU:  newByteLRU[[]string](ChunkCacheCap),
		weightLRU: newByteLRU[[]uint32](WeightCacheCap),
	}
	if err := s.ensureSchema(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// ensureSchema 建表 + 关闭 mmap + 校验 user_version（不匹配则重建），最后置版本号。
func (s *sqliteStore) ensureSchema() error {
	if err := s.writer.Exec("PRAGMA mmap_size=0").Error; err != nil {
		return err
	}

	var ver int
	if err := s.writer.Raw("PRAGMA user_version").Row().Scan(&ver); err != nil {
		return err
	}
	if ver > 0 && ver != SchemaVersion {
		for _, t := range []string{"deck_chunk", "deck_group", "deck_meta"} {
			if err := s.writer.Exec("DROP TABLE IF EXISTS " + t).Error; err != nil {
				return err
			}
		}
	}

	for _, ddl := range schemaDDL {
		if err := s.writer.Exec(ddl).Error; err != nil {
			return err
		}
	}

	// PRAGMA 不支持参数绑定，版本号来自常量 SchemaVersion。
	return s.writer.Exec(fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion)).Error
}

func (s *sqliteStore) logf(format string, args ...any) {
	if s.log != nil {
		s.log.Errorf(format, args...)
	}
}

func (s *sqliteStore) BeginImport(deckID string) (Import, error) {
	tx := s.writer.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	for _, table := range []string{"deck_meta", "deck_group", "deck_chunk"} {
		if err := tx.Exec("DELETE FROM "+table+" WHERE deck_key = ?", deckID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	return &sqliteImport{
		s:      s,
		tx:     tx,
		deckID: deckID,
		groups: make(map[string]GroupMeta),
	}, nil
}

func (s *sqliteStore) GroupMeta(deckID, group string) (GroupMeta, bool) {
	s.mu.RLock()
	gm, ok := s.index[Key(deckID, group)]
	s.mu.RUnlock()
	return gm, ok
}

func (s *sqliteStore) Item(deckID, group string, idx int) (string, error) {
	gm, ok := s.GroupMeta(deckID, group)
	if !ok {
		return "", ErrNotFound
	}
	chunkNo := ChunkForIndex(gm.ChunkStarts, idx)
	if chunkNo < 0 {
		return "", ErrNotFound
	}
	start := gm.ChunkStarts[chunkNo]
	local := idx - start

	key := chunkCacheKey(deckID, group, chunkNo)
	if items, hit := s.chunkLRU.get(key); hit {
		if local >= len(items) {
			return "", ErrNotFound
		}
		return items[local], nil
	}

	blob, err := s.readChunkBlob(deckID, group, chunkNo)
	if err != nil {
		return "", err
	}
	items, err := decodeChunk(blob)
	if err != nil {
		return "", err
	}
	s.chunkLRU.put(key, items, chunkByteSize(items))
	if local >= len(items) {
		return "", ErrNotFound
	}
	return items[local], nil
}

func (s *sqliteStore) readChunkBlob(deckID, group string, chunkNo int) ([]byte, error) {
	row := s.reader.Raw(
		"SELECT blob FROM deck_chunk WHERE deck_key = ? AND group_name = ? AND chunk_no = ?",
		deckID, group, chunkNo,
	).Row()
	var blob []byte
	if err := row.Scan(&blob); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return blob, nil
}

func (s *sqliteStore) Weights(deckID, group string) ([]uint32, error) {
	gm, ok := s.GroupMeta(deckID, group)
	if !ok {
		return nil, ErrNotFound
	}
	if !gm.Weighted {
		return nil, nil
	}

	key := Key(deckID, group)
	if w, hit := s.weightLRU.get(key); hit {
		return w, nil
	}

	row := s.reader.Raw(
		"SELECT weight_blob FROM deck_group WHERE deck_key = ? AND group_name = ?",
		deckID, group,
	).Row()
	var blob []byte
	if err := row.Scan(&blob); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	w, err := decodeWeights(blob)
	if err != nil {
		return nil, err
	}
	s.weightLRU.put(key, w, weightByteSize(w))
	return w, nil
}

func (s *sqliteStore) GroupItems(deckID, group string) []string {
	gm, ok := s.GroupMeta(deckID, group)
	if !ok {
		return nil
	}
	out := make([]string, 0, gm.ItemCount)
	for i := 0; i < gm.ItemCount; i++ {
		it, err := s.Item(deckID, group, i)
		if err != nil {
			s.logf("deckstore: GroupItems(%s, %s) at %d: %v", deckID, group, i, err)
			return nil
		}
		out = append(out, it)
	}
	return out
}

func (s *sqliteStore) Meta(deckID, key string) ([]byte, bool) {
	row := s.reader.Raw("SELECT v FROM deck_meta WHERE deck_key = ? AND k = ?", deckID, key).Row()
	var v []byte
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false
		}
		s.logf("deckstore: read meta(%s, %s): %v", deckID, key, err)
		return nil, false
	}
	return v, true
}

func (s *sqliteStore) DeckIDs() ([]string, error) {
	rows, err := s.reader.Raw(
		"SELECT deck_key FROM deck_meta UNION SELECT deck_key FROM deck_group",
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *sqliteStore) DeleteDeck(deckID string) error {
	tx := s.writer.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	for _, table := range []string{"deck_meta", "deck_group", "deck_chunk"} {
		if err := tx.Exec("DELETE FROM "+table+" WHERE deck_key = ?", deckID).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	prefix := deckID + "\x00"
	s.mu.Lock()
	for k := range s.index {
		if strings.HasPrefix(k, prefix) {
			delete(s.index, k)
		}
	}
	s.mu.Unlock()
	s.purgeDeckCaches(deckID)
	return nil
}

// purgeDeckCaches 清除该 deck 的块与权重缓存（导入提交 / 删除时调用）。
// 否则重导同一 deck 后，chunk 键不变的旧条目会滞留在 LRU，Item 命中过期内容。
func (s *sqliteStore) purgeDeckCaches(deckID string) {
	prefix := deckID + "\x00"
	s.chunkLRU.removePrefix(prefix)
	s.weightLRU.removePrefix(prefix)
}

func (s *sqliteStore) ReplaceIndex(m map[string]GroupMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index = make(map[string]GroupMeta, len(m))
	for k, v := range m {
		s.index[k] = v
	}
}

// indexReplace 在提交后更新索引：先移除该 deck 的旧键，再写入新组。
func (s *sqliteStore) indexReplace(deckID string, groups map[string]GroupMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := deckID + "\x00"
	for k := range s.index {
		if strings.HasPrefix(k, prefix) {
			delete(s.index, k)
		}
	}
	for k, v := range groups {
		s.index[k] = v
	}
}

func (s *sqliteStore) Close() error {
	var firstErr error
	if s.reader != nil {
		if db, err := s.reader.DB(); err == nil {
			firstErr = db.Close()
		}
	}
	if s.writer != nil && s.writer != s.reader {
		if db, err := s.writer.DB(); err == nil {
			if cerr := db.Close(); cerr != nil && firstErr == nil {
				firstErr = cerr
			}
		}
	}
	return firstErr
}

// sqliteImport 是一次导入写事务。
type sqliteImport struct {
	s      *sqliteStore
	tx     *gorm.DB
	deckID string
	groups map[string]GroupMeta
	err    error
	done   bool
}

func (im *sqliteImport) BeginGroup(name string) GroupWriter {
	return newGroupWriter(im, name)
}

func (im *sqliteImport) SetMeta(key string, value []byte) {
	if im.err != nil || im.done {
		return
	}
	if err := im.tx.Exec(
		"INSERT OR REPLACE INTO deck_meta (deck_key, k, v) VALUES (?, ?, ?)",
		im.deckID, key, value,
	).Error; err != nil {
		im.err = err
	}
}

func (im *sqliteImport) Commit() error {
	if im.done {
		return nil
	}
	im.done = true
	if im.err != nil {
		im.tx.Rollback()
		return im.err
	}
	if err := im.tx.Commit().Error; err != nil {
		return err
	}
	im.s.indexReplace(im.deckID, im.groups)
	im.s.purgeDeckCaches(im.deckID)
	return nil
}

func (im *sqliteImport) Abort() {
	if im.done {
		return
	}
	im.done = true
	im.tx.Rollback()
}
