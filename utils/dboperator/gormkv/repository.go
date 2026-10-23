package gormkv

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sealdice-core/utils/dboperator/gormkv/codec"
)

const (
	DefaultTableName = "kv_records"
	MaxListLimit     = 1000
	MaxMetadataBytes = 1024
	defaultListLimit = MaxListLimit

	DefaultPurgeBatchSize = 1000
	MaxPurgeBatchSize     = 10_000
)

var (
	ErrDBRequired        = errors.New("db is required")
	ErrNamespaceRequired = errors.New("namespace is required")
	ErrEmptyKey          = errors.New("key is required")
	ErrMetadataTooLarge  = errors.New("metadata exceeds 1024 bytes")
	ErrInvalidCursor     = errors.New("cursor is invalid")
	ErrHashCollision     = errors.New("hash collision detected")
)

// BackendOptions configures the shared database-backed KV backend. A Backend
// owns database handles and codec configuration; individual plugins use
// namespace-scoped Repository values created by Backend.Namespace.
type BackendOptions struct {
	TableName string
	// DB is used for both reads and writes when ReadDB / WriteDB are not set.
	DB *gorm.DB
	// ReadDB handles read-only queries (Get / List). Falls back to DB when nil.
	ReadDB *gorm.DB
	// WriteDB handles mutations and migrations (Put / Delete / PurgeExpired).
	// Falls back to DB when nil.
	WriteDB     *gorm.DB
	Now         func() time.Time
	Compression codec.Options
}

// Backend is the shared gormkv storage for one Dice instance. It does not own
// the underlying database lifecycle; DatabaseOperator closes those connections.
type Backend struct {
	readDB     *gorm.DB
	writeDB    *gorm.DB
	tableName  string
	now        func() time.Time
	compressor *codec.Compressor
}

type PurgeExpiredResult struct {
	Deleted int64
	HasMore bool
}

type Repository struct {
	readDB     *gorm.DB
	writeDB    *gorm.DB
	namespace  string
	tableName  string
	now        func() time.Time
	compressor *codec.Compressor
}

type kvRecord struct {
	ID           uint       `gorm:"column:id;primaryKey"`
	Namespace    string     `gorm:"column:namespace;size:255;not null;uniqueIndex:idx_namespace_hash,priority:1;index:idx_namespace_expires_at,priority:1"`
	KeyHash      string     `gorm:"column:key_hash;size:16;not null;uniqueIndex:idx_namespace_hash,priority:2"`
	RawKey       string     `gorm:"column:raw_key;type:text;not null"`
	ValueBlob    []byte     `gorm:"column:value_blob;not null"`
	ValueCodec   uint8      `gorm:"column:value_codec;not null;default:0"`
	MetadataJSON []byte     `gorm:"column:metadata_json"`
	ExpiresAt    *time.Time `gorm:"column:expires_at;index:idx_namespace_expires_at,priority:2"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;not null"`
}

func NewBackend(options BackendOptions) (*Backend, error) {
	readDB := options.ReadDB
	if readDB == nil {
		readDB = options.DB
	}
	writeDB := options.WriteDB
	if writeDB == nil {
		writeDB = options.DB
	}
	if readDB == nil || writeDB == nil {
		return nil, ErrDBRequired
	}

	tableName := strings.TrimSpace(options.TableName)
	if tableName == "" {
		tableName = DefaultTableName
	}

	nowFn := options.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	compressor, err := codec.New(options.Compression)
	if err != nil {
		return nil, err
	}

	return &Backend{
		readDB:     readDB,
		writeDB:    writeDB,
		tableName:  tableName,
		now:        nowFn,
		compressor: compressor,
	}, nil
}

// Namespace returns an immutable view of the backend for one plugin.
func (b *Backend) Namespace(namespace string) (*Repository, error) {
	if b == nil || b.readDB == nil || b.writeDB == nil {
		return nil, ErrDBRequired
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, ErrNamespaceRequired
	}
	return &Repository{
		readDB:     b.readDB,
		writeDB:    b.writeDB,
		namespace:  namespace,
		tableName:  b.tableName,
		now:        b.now,
		compressor: b.compressor,
	}, nil
}

// AutoMigrate creates the shared KV table. It is intentionally separate from
// NewBackend so the startup migration owns schema initialization explicitly.
func (b *Backend) AutoMigrate() error {
	if b == nil {
		return ErrDBRequired
	}
	return autoMigrate(b.writeDB, b.tableName)
}

// WriteDB exposes the write handle for migration metadata owned by callers.
// DatabaseOperator remains responsible for closing the underlying connection.
func (b *Backend) WriteDB() *gorm.DB {
	if b == nil {
		return nil
	}
	return b.writeDB
}

// PurgeExpired removes a bounded batch across all plugin namespaces without
// loading value blobs. Callers can schedule repeated batches when HasMore is true.
func (b *Backend) PurgeExpired(ctx context.Context, limit int) (PurgeExpiredResult, error) {
	if b == nil || b.writeDB == nil {
		return PurgeExpiredResult{}, ErrDBRequired
	}
	limit = normalizePurgeLimit(limit)
	cutoff := b.now()
	ids := make([]uint, 0, limit+1)
	result := b.writeDB.Table(b.tableName).WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at <= ?", cutoff).
		Order("id ASC").
		Limit(limit+1).
		Pluck("id", &ids)
	if result.Error != nil {
		return PurgeExpiredResult{}, result.Error
	}
	if len(ids) == 0 {
		return PurgeExpiredResult{}, nil
	}

	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}
	deleted := b.writeDB.Table(b.tableName).WithContext(ctx).
		Where("id IN ? AND expires_at IS NOT NULL AND expires_at <= ?", ids, cutoff).
		Delete(&kvRecord{})
	return PurgeExpiredResult{Deleted: deleted.RowsAffected, HasMore: hasMore}, deleted.Error
}

// AutoMigrate creates the shared KV table. It is intentionally separate from
// NewRepository so database startup and data migration have one explicit
// schema owner.
func AutoMigrate(db *gorm.DB) error {
	return autoMigrate(db, DefaultTableName)
}

func autoMigrate(db *gorm.DB, tableName string) error {
	if db == nil {
		return ErrDBRequired
	}
	if strings.TrimSpace(tableName) == "" {
		tableName = DefaultTableName
	}
	if err := db.Table(tableName).AutoMigrate(&kvRecord{}); err != nil {
		return err
	}
	return ensureRawKeyIndex(db, tableName)
}

func ensureRawKeyIndex(db *gorm.DB, tableName string) error {
	const indexName = "idx_namespace_raw_key"
	if db.Migrator().HasIndex(tableName, indexName) {
		return nil
	}
	dialector := db.Dialector
	quote := func(identifier string) string {
		var builder strings.Builder
		dialector.QuoteTo(&builder, identifier)
		return builder.String()
	}
	rawKeyColumn := quote("raw_key")
	if dialector.Name() == "mysql" {
		rawKeyColumn += "(255)"
	}
	statement := fmt.Sprintf(
		"CREATE INDEX %s ON %s (%s, %s)",
		quote(indexName),
		quote(tableName),
		quote("namespace"),
		rawKeyColumn,
	)
	return db.Exec(statement).Error
}

func (r *Repository) Put(ctx context.Context, key string, value []byte, options store.PutOptions) error {
	if strings.TrimSpace(key) == "" {
		return ErrEmptyKey
	}

	metadata, err := marshalMetadata(options.Metadata)
	if err != nil {
		return err
	}

	keyHash := hashKey(key)
	existing, found, err := r.findByHash(ctx, keyHash)
	if err != nil {
		return err
	}
	if found && existing.RawKey != key {
		return fmt.Errorf("%w for namespace %q", ErrHashCollision, r.namespace)
	}

	expiration := cloneTime(options.Expiration)
	if expiration == nil && options.ExpirationTTL != 0 {
		computed := r.now().Add(options.ExpirationTTL)
		expiration = &computed
	}

	blob, valueCodec := r.compressor.CompressWithKind(value, options.ValueKind)

	record := kvRecord{
		Namespace:    r.namespace,
		KeyHash:      keyHash,
		RawKey:       key,
		ValueBlob:    blob,
		ValueCodec:   valueCodec,
		MetadataJSON: metadata,
		ExpiresAt:    expiration,
	}

	return r.writeTable().WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "namespace"},
			{Name: "key_hash"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"raw_key":       record.RawKey,
			"value_blob":    record.ValueBlob,
			"value_codec":   record.ValueCodec,
			"metadata_json": record.MetadataJSON,
			"expires_at":    record.ExpiresAt,
			"updated_at":    r.now(),
		}),
	}).Create(&record).Error
}

func (r *Repository) Get(ctx context.Context, key string) (store.Record, bool, error) {
	if strings.TrimSpace(key) == "" {
		return store.Record{}, false, ErrEmptyKey
	}

	var row kvRecord
	result := r.activeRows(ctx).
		Select("value_blob", "value_codec", "metadata_json", "expires_at").
		Where("key_hash = ? AND raw_key = ?", hashKey(key), key).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return store.Record{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return store.Record{}, false, nil
	}

	record, err := r.toRecord(key, row)
	if err != nil {
		return store.Record{}, false, err
	}
	return record, true, nil
}

func (r *Repository) Delete(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return ErrEmptyKey
	}

	return r.writeTable().WithContext(ctx).
		Where("namespace = ? AND key_hash = ? AND raw_key = ?", r.namespace, hashKey(key), key).
		Delete(&kvRecord{}).Error
}

// Clear removes all records belonging to this namespace.
func (r *Repository) Clear(ctx context.Context) error {
	return r.writeTable().WithContext(ctx).
		Where("namespace = ?", r.namespace).
		Delete(&kvRecord{}).Error
}

// Count returns the number of non-expired records in this namespace.
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.activeRows(ctx).Model(&kvRecord{}).Count(&count)
	return count, result.Error
}

func (r *Repository) List(ctx context.Context, options store.ListOptions) (store.ListResult, error) {
	limit := normalizeLimit(options.Limit)

	query := r.activeRows(ctx).
		Select("raw_key", "metadata_json", "expires_at").
		Order("raw_key ASC").
		Limit(limit + 1)
	if options.Prefix != "" {
		query = query.Where("raw_key LIKE ? ESCAPE '\\'", escapeLike(options.Prefix)+"%")
	}
	if options.Cursor != "" {
		afterKey, err := decodeCursor(options.Cursor)
		if err != nil {
			return store.ListResult{}, err
		}
		query = query.Where("raw_key > ?", afterKey)
	}

	var rows []kvRecord
	if err := query.Find(&rows).Error; err != nil {
		return store.ListResult{}, err
	}

	result := store.ListResult{
		Keys:         make([]store.ListKey, 0, min(limit, len(rows))),
		ListComplete: true,
	}

	if len(rows) > limit {
		result.ListComplete = false
		rows = rows[:limit]
		result.Cursor = encodeCursor(rows[len(rows)-1].RawKey)
	}

	for _, row := range rows {
		result.Keys = append(result.Keys, store.ListKey{
			Name:       row.RawKey,
			Expiration: cloneTime(row.ExpiresAt),
			Metadata:   append(json.RawMessage(nil), row.MetadataJSON...),
		})
	}

	return result, nil
}

func (r *Repository) Namespace() string {
	return r.namespace
}

func (r *Repository) readTable() *gorm.DB {
	return r.readDB.Table(r.tableName)
}

func (r *Repository) writeTable() *gorm.DB {
	return r.writeDB.Table(r.tableName)
}

func (r *Repository) activeRows(ctx context.Context) *gorm.DB {
	return r.readTable().WithContext(ctx).
		Where("namespace = ?", r.namespace).
		Where("expires_at IS NULL OR expires_at > ?", r.now())
}

func (r *Repository) findByHash(ctx context.Context, keyHash string) (kvRecord, bool, error) {
	var row kvRecord
	result := r.writeTable().WithContext(ctx).
		Select("raw_key").
		Where("namespace = ? AND key_hash = ?", r.namespace, keyHash).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return kvRecord{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return kvRecord{}, false, nil
	}
	return row, true, nil
}

func hashKey(key string) string {
	return strconv.FormatUint(xxhash.Sum64String(key), 16)
}

func marshalMetadata(metadata any) ([]byte, error) {
	if metadata == nil {
		return nil, nil
	}

	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	if len(raw) > MaxMetadataBytes {
		return nil, fmt.Errorf("%w: %d bytes", ErrMetadataTooLarge, len(raw))
	}
	return raw, nil
}

func (r *Repository) toRecord(key string, row kvRecord) (store.Record, error) {
	value, err := r.compressor.Decompress(row.ValueBlob, row.ValueCodec)
	if err != nil {
		return store.Record{}, err
	}
	return store.Record{
		Key:        key,
		Value:      value,
		Metadata:   append(json.RawMessage(nil), row.MetadataJSON...),
		Expiration: cloneTime(row.ExpiresAt),
	}, nil
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultListLimit
	case limit > MaxListLimit:
		return MaxListLimit
	default:
		return limit
	}
}

func normalizePurgeLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultPurgeBatchSize
	case limit > MaxPurgeBatchSize:
		return MaxPurgeBatchSize
	default:
		return limit
	}
}

func encodeCursor(key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key))
}

func decodeCursor(cursor string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidCursor, cursor)
	}
	return string(decoded), nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(value)
}
