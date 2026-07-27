//nolint:testpackage // This test verifies the unexported cacher implementation directly.
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

	if storeErr := cacher.Store(disabledCtx, "disabled-store", query); storeErr != nil {
		t.Fatal(storeErr)
	}
	got, err := cacher.Get(normalCtx, "disabled-store", &caches.Query[any]{})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("cache-disabled query was stored")
	}

	if storeErr := cacher.Store(normalCtx, "normal-store", query); storeErr != nil {
		t.Fatal(storeErr)
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
