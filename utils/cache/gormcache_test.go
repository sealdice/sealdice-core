package cache

import (
	"context"
	"testing"

	"github.com/go-gorm/caches/v4"
	"github.com/maypok86/otter"
)

func TestDatabaseCacheDisabledContext(t *testing.T) {
	cacheInstance, err := otter.MustBuilder[string, []byte](10).Build()
	if err != nil {
		t.Fatal(err)
	}
	cacher := &OtterDBCacher{otter: &cacheInstance}
	normalCtx := context.WithValue(context.Background(), CacheKey, DataDBCacheKey)
	disabledCtx := WithDatabaseCacheDisabled(normalCtx)
	query := &caches.Query[any]{Dest: map[string]int{"value": 1}, RowsAffected: 1}

	if err := cacher.Store(disabledCtx, "disabled-store", query); err != nil {
		t.Fatal(err)
	}
	got, err := cacher.Get(normalCtx, "disabled-store", &caches.Query[any]{})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("cache-disabled query was stored")
	}

	if err := cacher.Store(normalCtx, "normal-store", query); err != nil {
		t.Fatal(err)
	}
	got, err = cacher.Get(disabledCtx, "normal-store", &caches.Query[any]{})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("cache-disabled query read a cached result")
	}
	got, err = cacher.Get(normalCtx, "normal-store", &caches.Query[any]{})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.RowsAffected != 1 {
		t.Fatalf("normal cached query = %#v", got)
	}
}
