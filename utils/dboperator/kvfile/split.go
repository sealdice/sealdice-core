package kvfile

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"gorm.io/gorm"
)

// Ref is the pointer row kept in the database for every file-backed (large)
// value. The database row is the small, portable source of truth for the key:
// it carries size, metadata and expiration, while the actual body lives in the
// folder-based FileStore. Because refs live in the same database as small
// values, list / getWithMetadata work uniformly and the whole index migrates
// with the database (SQLite → PostgreSQL / MySQL).
type Ref struct {
	ID           uint       `gorm:"primaryKey"`
	Namespace    string     `gorm:"column:namespace;size:255;not null;uniqueIndex:idx_kvfile_ns_key,priority:1"`
	Key          string     `gorm:"column:key;size:512;not null;uniqueIndex:idx_kvfile_ns_key,priority:2"`
	Size         int64      `gorm:"column:size;not null"`
	FileName     string     `gorm:"column:file_name;size:40;not null"`
	MetadataJSON []byte     `gorm:"column:metadata_json"`
	ExpiresAt    *time.Time `gorm:"column:expires_at;index"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;not null"`
}

// TableName fixes the ref table name.
func (Ref) TableName() string { return "plugin_kv_file_refs" }

// Backend is the folder-based large-value backend. It owns the ref table and
// the FileStore and produces per-namespace SplitStores that also route small
// values into an in-database NamespaceStore.
type Backend struct {
	readDB    *gorm.DB
	writeDB   *gorm.DB
	files     *FileStore
	threshold int64
}

// NewBackend creates a kvfile backend. readDB/writeDB own the pointer-row
// table (the same database instance as the small-value backend so the index is
// portable). AutoMigrate of the ref table is performed lazily on first use to
// avoid schema writes outside an initialized database; callers that manage
// schema explicitly should call AutoMigrate themselves.
func NewBackend(readDB, writeDB *gorm.DB, fileRoot string, threshold int64) (*Backend, error) {
	if readDB == nil {
		return nil, errors.New("kvfile: read db is required")
	}
	if writeDB == nil {
		return nil, errors.New("kvfile: write db is required")
	}
	files, err := NewFileStore(fileRoot)
	if err != nil {
		return nil, err
	}
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	return &Backend{
		readDB:    readDB,
		writeDB:   writeDB,
		files:     files,
		threshold: threshold,
	}, nil
}

// AutoMigrate creates the pointer-row table. Call once during startup on the
// write database.
func (b *Backend) AutoMigrate() error {
	return b.writeDB.AutoMigrate(&Ref{})
}

// Namespace returns a SplitStore scoped to namespace. small is the in-database
// store that holds small values for the same namespace (e.g. a gormkv
// Repository).
func (b *Backend) Namespace(small store.NamespaceStore, namespace string) (*SplitStore, error) {
	ns, err := sanitizeNamespace(namespace)
	if err != nil {
		return nil, err
	}
	if small == nil {
		return nil, errors.New("kvfile: small store is required")
	}
	return &SplitStore{
		small:     small,
		readDB:    b.readDB,
		writeDB:   b.writeDB,
		files:     b.files,
		ns:        ns,
		threshold: b.threshold,
		now:       time.Now,
	}, nil
}

// PurgeExpired removes expired file-backed values (body files and ref rows)
// across all namespaces, at most limit rows. It mirrors the small-value
// PurgeExpired lifecycle and is intended to run on the same cleanup cron.
func (b *Backend) PurgeExpired(ctx context.Context, limit int) (PurgeExpiredResult, error) {
	if limit <= 0 {
		limit = 100
	}
	var refs []Ref
	err := b.readDB.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).
		Limit(limit).
		Find(&refs).Error
	if err != nil {
		return PurgeExpiredResult{}, err
	}

	result := PurgeExpiredResult{}
	for _, ref := range refs {
		if err := b.files.DeleteBody(ref.Namespace, ref.Key); err != nil {
			return result, err
		}
		if err := b.writeDB.WithContext(ctx).
			Where("namespace = ? AND key = ?", ref.Namespace, ref.Key).
			Delete(&Ref{}).Error; err != nil {
			return result, err
		}
		result.FilesDeleted++
		result.RowsDeleted++
	}
	return result, nil
}

// ClearNamespace deletes all file-backed values (bodies and ref rows) for one
// namespace. Used when clearing an extension's storage.
func (b *Backend) ClearNamespace(ctx context.Context, namespace string) error {
	ns, err := sanitizeNamespace(namespace)
	if err != nil {
		return err
	}
	var refs []Ref
	if err := b.readDB.WithContext(ctx).
		Where("namespace = ?", ns).
		Find(&refs).Error; err != nil {
		return err
	}
	for _, ref := range refs {
		_ = b.files.DeleteBody(ref.Namespace, ref.Key)
	}
	q := b.writeDB.WithContext(ctx).Where("namespace = ?", ns).Delete(&Ref{})
	if q.Error != nil {
		return q.Error
	}
	// Remove the (now empty) namespace folder, ignoring errors.
	_ = b.files.RemoveNamespaceDir(namespace)
	return nil
}

// DefaultPurgeBatchSize bounds how many expired file-backed values a single
// purge pass removes. The cleanup cron runs in batches so it never deletes an
// unbounded set of files under one lock.
const DefaultPurgeBatchSize = 100

// PurgeExpiredResult reports how much a purge pass cleaned up.
type PurgeExpiredResult struct {
	FilesDeleted int64
	RowsDeleted  int64
}

// SplitStore is the routing NamespaceStore that satisfies
// store.NamespaceStore together with the optional StreamGetter and
// StreamPutter capabilities. Small values are delegated to the in-database
// small store; large values (above threshold) are promoted to file bodies with
// a pointer row; the two are kept mutually exclusive and migrate automatically
// on overwrite.
type SplitStore struct {
	small     store.NamespaceStore
	readDB    *gorm.DB
	writeDB   *gorm.DB
	files     *FileStore
	ns        string
	threshold int64
	now       func() time.Time
}

// Namespace returns the namespace scope of this store.
func (s *SplitStore) NamespaceScope() string { return s.ns }

var _ store.NamespaceStore = (*SplitStore)(nil)
var _ store.StreamGetter = (*SplitStore)(nil)
var _ store.StreamPutter = (*SplitStore)(nil)

func (s *SplitStore) isBig(size int64) bool { return size > s.threshold }

func (s *SplitStore) refExpired(ref Ref) bool {
	return ref.ExpiresAt != nil && !ref.ExpiresAt.After(s.now())
}

func resolveExpiration(now time.Time, options store.PutOptions) *time.Time {
	if options.Expiration != nil {
		t := *options.Expiration
		return &t
	}
	if options.ExpirationTTL != 0 {
		t := now.Add(options.ExpirationTTL)
		return &t
	}
	return nil
}

// Put routes a value by raw size: small values go to the database store,
// large values are promoted to a body file plus a pointer row. Both directions
// of overwrite (small→large, large→small) are migrated automatically.
func (s *SplitStore) Put(ctx context.Context, key string, value []byte, options store.PutOptions) error {
	_, hasRef, err := s.getRef(ctx, key)
	if err != nil {
		return err
	}

	if !s.isBig(int64(len(value))) {
		// Demote: remove any existing body + ref, then store small.
		if hasRef {
			_ = s.files.DeleteBody(s.ns, key)
			if err = s.deleteRef(ctx, key); err != nil {
				return err
			}
		}
		_ = s.small.Delete(ctx, key)
		return s.small.Put(ctx, key, value, options)
	}

	// Promote: write the file body, drop any existing small value, then publish
	// the pointer row. Failures roll the body back so no half-state survives.
	size := int64(len(value))
	if _, err = s.files.PutBody(ctx, s.ns, key, bytes.NewReader(value)); err != nil {
		return err
	}
	_ = s.small.Delete(ctx, key)
	expiresAt := resolveExpiration(s.now(), options)
	metadata, err := json.Marshal(options.Metadata)
	if err != nil {
		_ = s.files.DeleteBody(s.ns, key)
		return fmt.Errorf("kvfile: marshal metadata: %w", err)
	}
	if options.Metadata == nil {
		metadata = nil
	}
	now := s.now()
	record := Ref{
		Namespace:    s.ns,
		Key:          key,
		Size:         size,
		FileName:     hashKey(key) + ".bin",
		MetadataJSON: metadata,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.upsertRef(ctx, record); err != nil {
		_ = s.files.DeleteBody(s.ns, key)
		return err
	}
	return nil
}

// Get returns a value by reading the ref row (large) or the small store. Large
// value reads are loaded fully into memory here; callers that want bounded
// memory should use GetStream instead. This path is permitted but discouraged
// for large values.
func (s *SplitStore) Get(ctx context.Context, key string) (store.Record, bool, error) {
	ref, hasRef, err := s.getRef(ctx, key)
	if err != nil {
		return store.Record{}, false, err
	}
	if hasRef {
		if s.refExpired(ref) {
			return store.Record{}, false, nil
		}
		file, err := s.files.OpenBody(s.ns, key)
		if err != nil {
			if errors.Is(err, ErrBodyMissing) {
				return store.Record{}, false, nil
			}
			return store.Record{}, false, err
		}
		defer file.Close()
		value, err := io.ReadAll(file)
		if err != nil {
			return store.Record{}, false, err
		}
		record := store.Record{
			Key:        key,
			Value:      value,
			Metadata:   append(json.RawMessage(nil), ref.MetadataJSON...),
			Expiration: cloneTimeOpt(ref.ExpiresAt),
		}
		if int64(len(record.Value)) != ref.Size {
			// Body and ref disagree: treat as missing rather than serving a
			// corrupted value.
			return store.Record{}, false, nil
		}
		return record, true, nil
	}
	return s.small.Get(ctx, key)
}

// Delete removes a value from whichever side currently holds it.
func (s *SplitStore) Delete(ctx context.Context, key string) error {
	_, hasRef, err := s.getRef(ctx, key)
	if err != nil {
		return err
	}
	if hasRef {
		_ = s.files.DeleteBody(s.ns, key)
		if err := s.deleteRef(ctx, key); err != nil {
			return err
		}
	}
	return s.small.Delete(ctx, key)
}

// GetStream returns a stream for large values (backed by the committed body
// file) and for small values (backed by an in-memory bytes reader). Expired
// values report not-found.
func (s *SplitStore) GetStream(ctx context.Context, key string) (store.StreamRecord, bool, error) {
	ref, hasRef, err := s.getRef(ctx, key)
	if err != nil {
		return store.StreamRecord{}, false, err
	}
	if hasRef {
		if s.refExpired(ref) {
			return store.StreamRecord{}, false, nil
		}
		file, openErr := s.files.OpenBody(s.ns, key)
		if openErr != nil {
			if errors.Is(openErr, ErrBodyMissing) {
				return store.StreamRecord{}, false, nil
			}
			return store.StreamRecord{}, false, openErr
		}
		return store.StreamRecord{
			Key:        key,
			Body:       file,
			Size:       ref.Size,
			Metadata:   append(json.RawMessage(nil), ref.MetadataJSON...),
			Expiration: cloneTimeOpt(ref.ExpiresAt),
		}, true, nil
	}
	record, found, err := s.small.Get(ctx, key)
	if err != nil || !found {
		return store.StreamRecord{}, found, err
	}
	return store.StreamRecord{
		Key:        key,
		Body:       io.NopCloser(bytes.NewReader(record.Value)),
		Size:       int64(len(record.Value)),
		Metadata:   append(json.RawMessage(nil), record.Metadata...),
		Expiration: cloneTimeOpt(record.Expiration),
	}, true, nil
}

// PutStream streams a body to a unique temporary file incrementally, then
// commits it as a large value if it exceeds the threshold, or folds it into the
// small store if it turns out small. The previous value stays intact until the
// commit succeeds. The body never blocks the database with a long write.
func (s *SplitStore) PutStream(ctx context.Context, key string, body io.Reader, options store.PutOptions) error {
	// Stream to a temp file so the final size is known before choosing sides.
	temp, size, err := s.files.PutBodyTemp(s.ns, body)
	if err != nil {
		return err
	}

	if !s.isBig(size) {
		// Demote: fold the temp body into the small store, then discard it.
		smallBytes, readErr := s.files.ReadTempBytes(temp)
		_ = s.files.DiscardTemp(temp)
		if readErr != nil {
			return readErr
		}
		_ = s.small.Delete(ctx, key)
		return s.small.Put(ctx, key, smallBytes, options)
	}

	// Large: publish the temp body, drop any existing small value, then write
	// the pointer row. Failures roll the body back so no half-state survives.
	if err = s.files.CommitBody(s.ns, key, temp); err != nil {
		return err
	}
	_ = s.small.Delete(ctx, key)
	expiresAt := resolveExpiration(s.now(), options)
	metadata, err := json.Marshal(options.Metadata)
	if err != nil {
		_ = s.files.DeleteBody(s.ns, key)
		return fmt.Errorf("kvfile: marshal metadata: %w", err)
	}
	if options.Metadata == nil {
		metadata = nil
	}
	now := s.now()
	record := Ref{
		Namespace:    s.ns,
		Key:          key,
		Size:         size,
		FileName:     hashKey(key) + ".bin",
		MetadataJSON: metadata,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.upsertRef(ctx, record); err != nil {
		_ = s.files.DeleteBody(s.ns, key)
		return err
	}
	return nil
}

// List merges small values and file refs into one ascending, cursor-paginated
// result. Both sources are queried for limit+1 entries after the cursor and
// merged by key. A key exists on exactly one side, so there is no duplication.
func (s *SplitStore) List(ctx context.Context, options store.ListOptions) (store.ListResult, error) {
	limit := options.Limit
	if limit <= 0 {
		limit = 1000
	}
	after := ""
	if options.Cursor != "" {
		decoded, err := decodeCursor(options.Cursor)
		if err != nil {
			return store.ListResult{}, err
		}
		after = decoded
	}

	smallPage := store.ListOptions{
		Prefix: options.Prefix,
		Limit:  limit + 1,
		Cursor: options.Cursor,
	}
	smallResult, err := s.small.List(ctx, smallPage)
	if err != nil {
		return store.ListResult{}, err
	}

	var refs []Ref
	q := s.readDB.WithContext(ctx).
		Where("namespace = ?", s.ns).
		Order("key ASC").
		Limit(limit + 1)
	if options.Prefix != "" {
		q = q.Where("key LIKE ? ESCAPE '\\'", escapeLike(options.Prefix)+"%")
	}
	if after != "" {
		q = q.Where("key > ?", after)
	}
	if err := q.Find(&refs).Error; err != nil {
		return store.ListResult{}, err
	}

	merged := make([]store.ListKey, 0, limit+1)
	i, j := 0, 0
	for len(merged) <= limit && (i < len(smallResult.Keys) || j < len(refs)) {
		var pick store.ListKey
		var fromRef bool
		switch {
		case i >= len(smallResult.Keys):
			pick, fromRef = refKey(refs[j]), true
			j++
		case j >= len(refs):
			pick = smallResult.Keys[i]
			i++
		case smallResult.Keys[i].Name <= refs[j].Key:
			pick = smallResult.Keys[i]
			i++
		default:
			pick, fromRef = refKey(refs[j]), true
			j++
		}
		if fromRef && pick.Name == "" {
			continue
		}
		merged = append(merged, pick)
	}

	// There is more only if a source still has a next page after merged.
	hasMore := len(merged) > limit ||
		(len(smallResult.Keys) > i && i < len(smallResult.Keys)) ||
		(len(refs) > j && j < len(refs))
	if len(merged) > limit {
		merged = merged[:limit]
	}
	result := store.ListResult{
		Keys:         merged,
		ListComplete: !hasMore,
	}
	if !result.ListComplete && len(merged) > 0 {
		result.Cursor = encodeCursor(merged[len(merged)-1].Name)
	}
	return result, nil
}

// Clear removes every value (small + file) for this namespace.
func (s *SplitStore) Clear(ctx context.Context) error {
	// Clear file side.
	var refs []Ref
	if err := s.readDB.WithContext(ctx).
		Where("namespace = ?", s.ns).
		Find(&refs).Error; err != nil {
		return err
	}
	for _, ref := range refs {
		_ = s.files.DeleteBody(s.ns, ref.Key)
	}
	if err := s.writeDB.WithContext(ctx).
		Where("namespace = ?", s.ns).
		Delete(&Ref{}).Error; err != nil {
		return err
	}
	// Clear small side, if the store supports it.
	if c, ok := s.small.(interface {
		Clear(context.Context) error
	}); ok {
		if err := c.Clear(ctx); err != nil {
			return err
		}
	}
	return nil
}

func refKey(ref Ref) store.ListKey {
	key := store.ListKey{Name: ref.Key}
	if ref.ExpiresAt != nil {
		t := *ref.ExpiresAt
		key.Expiration = &t
	}
	if len(ref.MetadataJSON) > 0 {
		key.Metadata = append(json.RawMessage(nil), ref.MetadataJSON...)
	}
	return key
}

func (s *SplitStore) getRef(ctx context.Context, key string) (Ref, bool, error) {
	var ref Ref
	err := s.readDB.WithContext(ctx).
		Where("namespace = ? AND key = ?", s.ns, key).
		First(&ref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Ref{}, false, nil
	}
	if err != nil {
		return Ref{}, false, err
	}
	return ref, true, nil
}

func (s *SplitStore) upsertRef(ctx context.Context, record Ref) error {
	return s.writeDB.WithContext(ctx).Save(&record).Error
}

func (s *SplitStore) deleteRef(ctx context.Context, key string) error {
	return s.writeDB.WithContext(ctx).
		Where("namespace = ? AND key = ?", s.ns, key).
		Delete(&Ref{}).Error
}

func encodeCursor(key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key))
}

func decodeCursor(cursor string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func cloneTimeOpt(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cloned := *t
	return &cloned
}
