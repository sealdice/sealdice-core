package model

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-gorm/caches/v4"
	"github.com/spaolacci/murmur3"
	"github.com/tidwall/buntdb"
	"gorm.io/gorm"
)

type buntDBCacher struct {
	db *buntdb.DB
}

func generateHashKey(key string) string {
	hash := murmur3.Sum64([]byte(key))
	return strconv.FormatUint(hash, 16) // 返回十六进制字符串
}

// Get 从缓存中获取与给定键关联的数据。
// 该方法接受一个上下文、一个键和一个查询对象作为参数。
// 它首先将键转换为哈希键，然后从数据库中获取相应的值。
// 如果键不存在于数据库中，则返回nil, nil。
// 如果存在错误，将返回错误信息。
// 如果成功获取数据，将返回填充了数据的查询对象。
func (c *buntDBCacher) Get(_ context.Context, key string, q *caches.Query[any]) (*caches.Query[any], error) {
	// 生成哈希键以确定缓存的位置。
	hashedKey := generateHashKey(key)

	// 尝试查找对应的关联值
	var res string
	err := c.db.View(func(tx *buntdb.Tx) error {
		var err error
		// 从事务中获取与哈希键关联的值。
		res, err = tx.Get(hashedKey)
		return err
	})

	// 如果键在数据库中不存在，记录信息并返回nil, nil。
	if errors.Is(err, buntdb.ErrNotFound) {
		// 此处不得不忽略，因为这个cache的实现机理就是如此，除非修改gorm cache的源码。
		return nil, nil //nolint:nilnil
	}

	// 如果发生其他错误，返回错误信息。
	if err != nil {
		return nil, err
	}
	// 将获取到的值解码为查询对象。
	if err = q.Unmarshal([]byte(res)); err != nil {
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
func (c *buntDBCacher) Store(_ context.Context, key string, val *caches.Query[any]) error {
	// 生成哈希键以确保键的均匀分布和避免潜在的键冲突。
	hashedKey := generateHashKey(key)
	// 将查询对象序列化为字节切片，以便存储到缓存中。
	res, err := val.Marshal()
	if err != nil {
		return err
	}
	// 使用数据库的Update方法来原子地设置数据。
	err = c.db.Update(func(tx *buntdb.Tx) error {
		// 设置键值对，并指定数据过期时间为5秒。
		_, _, err = tx.Set(hashedKey, string(res), &buntdb.SetOptions{Expires: true, TTL: time.Second * 5})
		return err
	})

	return err
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
func (c *buntDBCacher) Invalidate(_ context.Context) error {
	// 清理所有缓存
	err := c.db.Update(func(tx *buntdb.Tx) error {
		err := tx.DeleteAll()
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GetBuntCacheDB(db *gorm.DB) (*gorm.DB, error) {
	open, err := buntdb.Open(":memory:")
	if err != nil {
		return nil, err
	}
	// Easer参数：使用ServantGo任务执行与合并库
	// ServantGo提供了一种简单且惯用的方法来合并同时运行的相同类型的任务。
	// 可以先尝试一下easer=true是否可以加速
	cachesPlugin := &caches.Caches{Conf: &caches.Config{
		Easer: true,
		Cacher: &buntDBCacher{
			db: open,
		},
	}}
	err = db.Use(cachesPlugin)
	if err != nil {
		return nil, err
	}
	return db, nil
}
