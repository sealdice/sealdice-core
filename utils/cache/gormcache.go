package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorm/caches/v4"
	"github.com/maypok86/otter"
	"gorm.io/gorm"
)

type OtterDBCacher struct {
	otter *otter.Cache[string, []byte]
}

// 定义一个基于 string 的新类型 cacheKey
type cacheKey string

const (
	CacheKey          cacheKey = "gorm_cache"
	LogsDBCacheKey    cacheKey = "logs-db::"
	DataDBCacheKey    cacheKey = "data-db::"
	CensorsDBCacheKey cacheKey = "censor-db::"
)

func (c *OtterDBCacher) getKeyWithCtx(ctx context.Context, key string) string {
	// 获取ctx中的key字段
	ctxCacheKey := fmt.Sprintf("%v", ctx.Value(CacheKey))
	if ctxCacheKey == "" {
		ctxCacheKey = caches.IdentifierPrefix // gorm-caches::
	}
	storeCacheKey := fmt.Sprintf("%s%s", ctxCacheKey, key)
	return storeCacheKey
}

// Get 从缓存中获取与给定键关联的数据。
// 该方法接受一个上下文、一个键和一个查询对象作为参数。
// 它首先将键转换为哈希键，然后从数据库中获取相应的值。
// 如果键不存在于数据库中，则返回nil, nil。
// 如果存在错误，将返回错误信息。
// 如果成功获取数据，将返回填充了数据的查询对象。
// TODO: 有点奇怪的逻辑，或许我应该直接存储Byte？
func (c *OtterDBCacher) Get(ctx context.Context, key string, q *caches.Query[any]) (*caches.Query[any], error) {
	result, ok := c.otter.Get(c.getKeyWithCtx(ctx, key))
	if !ok {
		// 设计如此
		return nil, nil //nolint:nilnil
	}
	if err := q.Unmarshal(result); err != nil {
		return nil, err
	}
	return q, nil
}

// Store 方法用于将查询结果存储到缓存中。
// 该方法接收一个上下文、一个键和一个查询对象作为参数。
// 它首先对键进行哈希处理，然后将查询对象序列化为字节切片。
// 序列化成功后，它将数据存储到缓存数据库中，并设置数据过期时间为5秒。
// 参数:
//
//	_ context.Context: 上下文，本例中未使用。
//	key string: 需要存储的数据的键。
//	val *caches.Query[any]: 需要存储的查询对象。
//
// 返回值:
//
//	error: 在序列化或存储过程中遇到的错误，如果没有错误则返回nil。
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

// Invalidate 使缓存器中的所有缓存项失效。
// 该方法通过删除数据库中所有以 caches.IdentifierPrefix 开头的键来实现。
// 参数:
//
//	_context.Context: 未使用。
//
// 返回值:
//
//	error: 如果在使缓存项失效的过程中发生错误，则返回该错误。
func (c *OtterDBCacher) Invalidate(ctx context.Context) error {
	// 查看插入的是哪个链接的，删除对应的数据项
	prefix := c.getKeyWithCtx(ctx, "")
	// 删除所有以prefix开头的键
	c.otter.DeleteByFunc(func(key string, _ []byte) bool {
		return strings.HasPrefix(key, prefix)
	})
	return nil
}

// GetOtterCacheDB 思路：查询使用的DB全部使用WithContext处理 保底一个，如果取不到数据就放在默认内，以避免报错等。
// 提供一个查询函数打印日志，该函数可以查询缓存的状态（缓存库提供了）
// 这样的话就需要稍微改动一下
func GetOtterCacheDB(db *gorm.DB) (*gorm.DB, error) {
	plugin, err := GetOtterCacheDBPluginInstance()
	if err != nil {
		return nil, err
	}
	err = db.Use(plugin)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetOtterCacheDBPluginInstance() (*caches.Caches, error) {
	cacheInstance, err := otter.MustBuilder[string, []byte](10_000).
		CollectStats().
		WithTTL(time.Hour).
		Build()
	if err != nil {
		return nil, err
	}
	return &caches.Caches{Conf: &caches.Config{
		Easer: true,
		Cacher: &OtterDBCacher{
			otter: &cacheInstance,
		},
	}}, nil
}
