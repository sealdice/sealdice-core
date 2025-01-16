//go:build cgo
// +build cgo

package database

import (
	"database/sql"
	"fmt"
	"runtime"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"

	"sealdice-core/dice/model/database/cache"
)

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	// 使用即时事务
	path = fmt.Sprintf("%v?_txlock=immediate&_busy_timeout=99999", path)
	// ReadDB调整为最大打开 = runtime.NumCPU/8个
	readDB, err := sql.Open(sqlite.DriverName, path)
	if err != nil {
		return nil, err
	}
	readDB.SetMaxOpenConns(max(8, runtime.NumCPU()))
	// writeDB仅允许打开一个链接
	writeDB, err := sql.Open(sqlite.DriverName, path)
	if err != nil {
		return nil, err
	}
	writeDB.SetMaxOpenConns(1)
	// 启动主数据库为writeDB
	open, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: writeDB,
	}), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	err = open.Use(dbresolver.Register(dbresolver.Config{
		// 创建读取DB
		Replicas: []gorm.Dialector{
			sqlite.New(sqlite.Config{
				Conn: readDB,
			}),
		},
		TraceResolverMode: true,
	}))
	if err != nil {
		return nil, err
	}
	// 启用缓存
	open, err = cache.GetOtterCacheDB(open)
	// 所有优化增加
	if err != nil {
		return nil, err
	}

	if useWAL {
		err = open.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			panic(err)
		}
	}
	// PRAGMA synchronous = NORMAL;
	// PRAGMA cache_size = 1000000000;

	return open, err
}
