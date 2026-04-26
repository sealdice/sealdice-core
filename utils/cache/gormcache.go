package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-gorm/caches/v4"
	"github.com/maypok86/otter"
	"gorm.io/gorm"
)

var (
	pluginRegistryMu sync.Mutex
	pluginRegistry   = map[*Plugin]struct{}{}
)

type Plugin struct {
	*caches.Caches
	otter     *otter.Cache[string, []byte]
	closeOnce sync.Once
}

func (p *Plugin) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		if p.otter != nil {
			p.otter.Close()
		}
		pluginRegistryMu.Lock()
		delete(pluginRegistry, p)
		pluginRegistryMu.Unlock()
	})
}

func CloseAllOtterCachePlugins() {
	pluginRegistryMu.Lock()
	plugins := make([]*Plugin, 0, len(pluginRegistry))
	for plugin := range pluginRegistry {
		plugins = append(plugins, plugin)
	}
	pluginRegistryMu.Unlock()

	for _, plugin := range plugins {
		plugin.Close()
	}
}

type OtterDBCacher struct {
	otter *otter.Cache[string, []byte]
}

type cacheKey string

const (
	CacheKey          cacheKey = "gorm_cache"
	LogsDBCacheKey    cacheKey = "logs-db::"
	DataDBCacheKey    cacheKey = "data-db::"
	CensorsDBCacheKey cacheKey = "censor-db::"
)

func (c *OtterDBCacher) getKeyWithCtx(ctx context.Context, key string) string {
	ctxCacheKey := fmt.Sprintf("%v", ctx.Value(CacheKey))
	if ctxCacheKey == "" {
		ctxCacheKey = caches.IdentifierPrefix
	}
	return fmt.Sprintf("%s%s", ctxCacheKey, key)
}

func (c *OtterDBCacher) Get(ctx context.Context, key string, q *caches.Query[any]) (*caches.Query[any], error) {
	result, ok := c.otter.Get(c.getKeyWithCtx(ctx, key))
	if !ok {
		return nil, nil //nolint:nilnil
	}
	if err := q.Unmarshal(result); err != nil {
		return nil, err
	}
	return q, nil
}

func (c *OtterDBCacher) Store(ctx context.Context, key string, val *caches.Query[any]) error {
	storeBytes, err := val.Marshal()
	if err != nil {
		return err
	}
	ok := c.otter.Set(c.getKeyWithCtx(ctx, key), storeBytes)
	if !ok {
		return errors.New("cache store in otter failed")
	}
	return nil
}

func (c *OtterDBCacher) Invalidate(ctx context.Context) error {
	prefix := c.getKeyWithCtx(ctx, "")
	c.otter.DeleteByFunc(func(key string, _ []byte) bool {
		return strings.HasPrefix(key, prefix)
	})
	return nil
}

func GetOtterCacheDB(db *gorm.DB) (*gorm.DB, error) {
	cacheDB, _, err := AttachOtterCache(db)
	if err != nil {
		return nil, err
	}
	return cacheDB, nil
}

func AttachOtterCache(db *gorm.DB) (*gorm.DB, *Plugin, error) {
	plugin, err := GetOtterCacheDBPluginInstance()
	if err != nil {
		return nil, nil, err
	}
	err = db.Use(plugin)
	if err != nil {
		plugin.Close()
		return nil, nil, err
	}
	return db, plugin, nil
}

func GetOtterCacheDBPluginInstance() (*Plugin, error) {
	cacheInstance, err := otter.MustBuilder[string, []byte](10_000).
		CollectStats().
		WithTTL(time.Hour).
		Build()
	if err != nil {
		return nil, err
	}

	plugin := &Plugin{
		Caches: &caches.Caches{Conf: &caches.Config{
			Easer: true,
			Cacher: &OtterDBCacher{
				otter: &cacheInstance,
			},
		}},
		otter: &cacheInstance,
	}

	pluginRegistryMu.Lock()
	pluginRegistry[plugin] = struct{}{}
	pluginRegistryMu.Unlock()

	return plugin, nil
}
