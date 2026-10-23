package kvfile //nolint:testpackage // 需要访问未导出的 getRef/files 内部结构

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// memSmall is an in-memory store.NamespaceStore double standing in for the
// gormkv repository during tests.
type memSmall struct {
	mu  sync.Mutex
	m   map[string]store.Record
	now func() time.Time
}

func newMemSmall() *memSmall {
	return &memSmall{m: map[string]store.Record{}, now: time.Now}
}

func (s *memSmall) Put(ctx context.Context, key string, value []byte, options store.PutOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var metadata json.RawMessage
	if options.Metadata != nil {
		b, err := json.Marshal(options.Metadata)
		if err != nil {
			return err
		}
		metadata = b
	}
	exp := options.Expiration
	if exp == nil && options.ExpirationTTL != 0 {
		t := s.now().Add(options.ExpirationTTL)
		exp = &t
	}
	rec := store.Record{
		Key:        key,
		Value:      append([]byte(nil), value...),
		Metadata:   append(json.RawMessage(nil), metadata...),
		Expiration: cloneTimeOpt(exp),
	}
	s.m[key] = rec
	return nil
}

func (s *memSmall) Get(ctx context.Context, key string) (store.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.m[key]
	if !ok {
		return store.Record{}, false, nil
	}
	if rec.Expiration != nil && !rec.Expiration.After(s.now()) {
		delete(s.m, key)
		return store.Record{}, false, nil
	}
	return rec, true, nil
}

func (s *memSmall) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	return nil
}

func (s *memSmall) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.m)
	return nil
}

func (s *memSmall) List(ctx context.Context, options store.ListOptions) (store.ListResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := options.Limit
	if limit <= 0 {
		limit = 1000
	}
	after := ""
	if options.Cursor != "" {
		b, err := base64.StdEncoding.DecodeString(options.Cursor)
		if err != nil {
			return store.ListResult{}, err
		}
		after = string(b)
	}
	keys := make([]string, 0, len(s.m))
	for key := range s.m {
		if options.Prefix != "" && !strings.HasPrefix(key, options.Prefix) {
			continue
		}
		if after != "" && key <= after {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := store.ListResult{Keys: []store.ListKey{}, ListComplete: true}
	if len(keys) > limit {
		result.ListComplete = false
	}
	keys = keys[:min(len(keys), limit)]
	for _, key := range keys {
		rec := s.m[key]
		item := store.ListKey{Name: key}
		if len(rec.Metadata) > 0 {
			item.Metadata = append(json.RawMessage(nil), rec.Metadata...)
		}
		if rec.Expiration != nil {
			item.Expiration = cloneTimeOpt(rec.Expiration)
		}
		result.Keys = append(result.Keys, item)
	}
	return result, nil
}

var (
	_ store.NamespaceStore = (*SplitStore)(nil)
	_ store.StreamGetter   = (*SplitStore)(nil)
	_ store.StreamPutter   = (*SplitStore)(nil)
)

type testHarness struct {
	small *memSmall
	be    *Backend
	store *SplitStore
	db    *gorm.DB
}

func newTestHarness(t *testing.T, threshold int64) *testHarness {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	be, err := NewBackend(db, db, t.TempDir(), threshold)
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}
	if err = be.AutoMigrate(); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	small := newMemSmall()
	sp, err := be.Namespace(small, "testext")
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	return &testHarness{small: small, be: be, store: sp, db: db}
}

func TestSmallValueStaysInDB(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	if err := h.store.Put(ctx, "cfg:a", []byte("hello"), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "cfg:a"); hasRef {
		t.Fatal("small value must not create a file ref")
	}
	if _, ok, _ := h.small.Get(ctx, "cfg:a"); !ok {
		t.Fatal("small value must live in the small store")
	}
	got, found, err := h.store.Get(ctx, "cfg:a")
	if err != nil || !found || string(got.Value) != "hello" {
		t.Fatalf("get small = %q found=%v err=%v", got.Value, found, err)
	}
}

func TestLargeValuePromotesToFile(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	big := bytes.Repeat([]byte("A"), 1024*1024)
	if err := h.store.Put(ctx, "blob:x", big, store.PutOptions{
		Metadata:      map[string]any{"kind": "avatar"},
		ExpirationTTL: 3600 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	ref, hasRef, err := h.store.getRef(ctx, "blob:x")
	if err != nil || !hasRef {
		t.Fatalf("large value must create a ref: %v", err)
	}
	if ref.Size != int64(len(big)) {
		t.Fatalf("ref size = %d want %d", ref.Size, len(big))
	}
	if ref.ExpiresAt == nil {
		t.Fatal("expiration must be recorded")
	}
	if _, ok, _ := h.small.Get(ctx, "blob:x"); ok {
		t.Fatal("large value must not remain in small store")
	}
	// Body file must exist.
	if _, err = h.be.files.OpenBody("testext", "blob:x"); err != nil {
		t.Fatalf("body missing: %v", err)
	}
	// Get reads whole value back.
	rec, found, err := h.store.Get(ctx, "blob:x")
	if err != nil || !found || !bytes.Equal(rec.Value, big) {
		t.Fatalf("get large value mismatch found=%v err=%v", found, err)
	}
	var meta map[string]any
	if err := json.Unmarshal(rec.Metadata, &meta); err != nil || meta["kind"] != "avatar" {
		t.Fatalf("metadata not preserved: %v %v", meta, err)
	}
}

func TestPromoteDemoteOverwrite(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	big := bytes.Repeat([]byte("B"), 2048)

	// small -> large
	if err := h.store.Put(ctx, "k", []byte("small"), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Put(ctx, "k", big, store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "k"); !hasRef {
		t.Fatal("expected ref after promote")
	}
	if _, ok, _ := h.small.Get(ctx, "k"); ok {
		t.Fatal("small replica must be gone after promote")
	}

	// large -> small
	if err := h.store.Put(ctx, "k", []byte("tiny"), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "k"); hasRef {
		t.Fatal("ref must be removed after demote")
	}
	if _, err := h.be.files.OpenBody("testext", "k"); !errors.Is(err, ErrBodyMissing) {
		t.Fatalf("body must be removed after demote, got %v", err)
	}
	rec, found, _ := h.store.Get(ctx, "k")
	if !found || string(rec.Value) != "tiny" {
		t.Fatalf("get after demote = %q found=%v", rec.Value, found)
	}
}

func TestGetStreamLarge(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	big := bytes.Repeat([]byte("S"), 4096)
	if err := h.store.Put(ctx, "s", big, store.PutOptions{
		Metadata: map[string]any{"n": 1},
	}); err != nil {
		t.Fatal(err)
	}
	stream, found, err := h.store.GetStream(ctx, "s")
	if err != nil || !found {
		t.Fatalf("getstream %v %v", found, err)
	}
	if stream.Size != int64(len(big)) {
		t.Fatalf("stream size = %d", stream.Size)
	}
	got, err := io.ReadAll(stream.Body)
	_ = stream.Body.Close()
	if err != nil || !bytes.Equal(got, big) {
		t.Fatalf("stream content mismatch")
	}
	var meta map[string]any
	if err := json.Unmarshal(stream.Metadata, &meta); err != nil || meta["n"] != float64(1) {
		t.Fatalf("stream metadata: %v %v", meta, err)
	}
}

func TestGetStreamSmallWrapsBytes(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	if err := h.store.Put(ctx, "s", []byte("abc"), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	stream, found, err := h.store.GetStream(ctx, "s")
	if err != nil || !found {
		t.Fatalf("getstream small %v %v", found, err)
	}
	got, _ := io.ReadAll(stream.Body)
	_ = stream.Body.Close()
	if string(got) != "abc" {
		t.Fatalf("small stream = %q", got)
	}
}

func TestPutStreamSmallDemotesToDB(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	// Small streamed payload should end up in the small store, no file.
	if err := h.store.PutStream(ctx, "p", strings.NewReader("tiny"), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "p"); hasRef {
		t.Fatal("small stream must not create a ref")
	}
	rec, found, _ := h.store.Get(ctx, "p")
	if !found || string(rec.Value) != "tiny" {
		t.Fatalf("demoted stream = %q found=%v", rec.Value, found)
	}
}

func TestPutStreamLargePromotes(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	big := bytes.Repeat([]byte("T"), 4096)
	if err := h.store.PutStream(ctx, "p", bytes.NewReader(big), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "p"); !hasRef {
		t.Fatal("big stream must create a ref")
	}
	stream, found, _ := h.store.GetStream(ctx, "p")
	if !found {
		t.Fatal("not found")
	}
	got, _ := io.ReadAll(stream.Body)
	_ = stream.Body.Close()
	if !bytes.Equal(got, big) {
		t.Fatal("stream mismatch")
	}
}

func TestListMergeAndPaginate(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	for i := range 7 {
		k := "k" + string(rune('a'+i))
		if i%2 == 0 { // even -> small
			if err := h.store.Put(ctx, k, []byte("x"), store.PutOptions{}); err != nil {
				t.Fatal(err)
			}
		} else { // odd -> large
			if err := h.store.Put(ctx, k, bytes.Repeat([]byte("x"), 2048), store.PutOptions{}); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Full list, limit 3 pages.
	var names []string
	cursor := ""
	for {
		page, err := h.store.List(ctx, store.ListOptions{Limit: 3, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range page.Keys {
			names = append(names, k.Name)
		}
		if page.ListComplete {
			break
		}
		cursor = page.Cursor
		if cursor == "" {
			t.Fatal("incomplete page without cursor")
		}
	}
	want := []string{"ka", "kb", "kc", "kd", "ke", "kf", "kg"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("merged list = %v want %v", names, want)
	}
}

func TestExpiredLargeReadAsMissingAndPurged(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	now := time.Now().UTC()
	h.store.now = func() time.Time { return now }
	big := bytes.Repeat([]byte("E"), 2048)
	if err := h.store.Put(ctx, "e", big, store.PutOptions{
		ExpirationTTL: 60 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	// Not yet expired.
	if _, found, _ := h.store.Get(ctx, "e"); !found {
		t.Fatal("should be found before expiry")
	}
	// Advance the clock past expiry.
	h.store.now = func() time.Time { return now.Add(120 * time.Second) }
	if _, found, _ := h.store.Get(ctx, "e"); found {
		t.Fatal("expired value must read as missing")
	}
	// Purge removes file + row.
	res, err := h.be.PurgeExpired(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if res.RowsDeleted != 1 || res.FilesDeleted != 1 {
		t.Fatalf("purge = %+v", res)
	}
	if _, err := h.be.files.OpenBody("testext", "e"); !errors.Is(err, ErrBodyMissing) {
		t.Fatalf("body should be gone after purge, got %v", err)
	}
}

func TestDeleteAndClear(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	if err := h.store.Put(ctx, "small", []byte("x"), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Put(ctx, "big", bytes.Repeat([]byte("B"), 2048), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Delete(ctx, "big"); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "big"); hasRef {
		t.Fatal("ref must be gone after delete")
	}
	if _, err := h.be.files.OpenBody("testext", "big"); !errors.Is(err, ErrBodyMissing) {
		t.Fatalf("body must be gone after delete, got %v", err)
	}

	if err := h.store.Put(ctx, "big2", bytes.Repeat([]byte("B"), 2048), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if _, hasRef, _ := h.store.getRef(ctx, "big2"); hasRef {
		t.Fatal("ref must be gone after clear")
	}
	if _, ok, _ := h.small.Get(ctx, "small"); ok {
		t.Fatal("small must be gone after clear")
	}
}

func TestPutStreamErrorLeavesNoState(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	// A reader that fails mid-stream.
	failing := &errReader{data: bytes.Repeat([]byte("X"), 2048), n: 10, err: errors.New("boom")}
	if err := h.store.PutStream(ctx, "bad", failing, store.PutOptions{}); err == nil {
		t.Fatal("expected failure")
	}
	if _, hasRef, _ := h.store.getRef(ctx, "bad"); hasRef {
		t.Fatal("no ref must remain after failed stream")
	}
	if _, err := h.be.files.OpenBody("testext", "bad"); !errors.Is(err, ErrBodyMissing) {
		t.Fatalf("no body must remain after failed stream, got %v", err)
	}
}

// TestPutStreamOverwriteKeepsOldOnFailure: an existing large value must stay
// intact (body + ref) when an overwriting stream aborts mid-write.
func TestPutStreamOverwriteKeepsOldOnFailure(t *testing.T) {
	h := newTestHarness(t, 512)
	ctx := context.Background()
	old := bytes.Repeat([]byte("OLD"), 2048)
	if err := h.store.PutStream(ctx, "k", bytes.NewReader(old), store.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	failing := &errReader{data: bytes.Repeat([]byte("NEW"), 2048), n: 4, err: errors.New("boom")}
	if err := h.store.PutStream(ctx, "k", failing, store.PutOptions{}); err == nil {
		t.Fatal("expected failure")
	}
	got, found, _ := h.store.Get(ctx, "k")
	if !found || !bytes.Equal(got.Value, old) {
		t.Fatalf("old value must survive failed overwrite, got len=%d found=%v", len(got.Value), found)
	}
}

type errReader struct {
	data []byte
	n    int
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	step := min(len(p), 4)
	copy(p, r.data[:step])
	r.data = r.data[step:]
	return step, nil
}
