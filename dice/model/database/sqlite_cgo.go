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

	"sealdice-core/dice/model/database/cache"
)

// 警告：不要在一个事务（写事务）里使用读的DB！否则读的DB会发现有人在写而锁住，从而死锁。

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	// 使用即时事务
	path = fmt.Sprintf("%v?_txlock=immediate&_busy_timeout=99999", path)
	open, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	// Enable Cache Mode
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
	return open, err
}

func createReadDB(path string, gormConf *gorm.Config) (*gorm.DB, error) {
	// ---- 创建读连接 -----
	readDB, err := gorm.Open(sqlite.Open(path), gormConf)
	if err != nil {
		return nil, err
	}
	readPool, err := readDB.DB()
	if err != nil {
		return nil, err
	}
	err = SetDefaultPragmas(readPool)
	if err != nil {
		return nil, err
	}
	readPool.SetMaxOpenConns(max(4, runtime.NumCPU()))
	return readDB, nil
}

func createWriteDB(path string, gormConf *gorm.Config) (*gorm.DB, error) {
	// ---- 创建写连接 -----
	writeDB, err := gorm.Open(sqlite.Open(path), gormConf)
	if err != nil {
		return nil, err
	}
	writePool, err := writeDB.DB()
	if err != nil {
		return nil, err
	}
	err = SetDefaultPragmas(writePool)
	if err != nil {
		return nil, err
	}
	writePool.SetMaxOpenConns(1) // only use one active connection for writing
	return writeDB, nil
}

func SQLiteDBRWInit(path string) (*gorm.DB, *gorm.DB, error) {
	// 由于现在我们只有一个写入连接，所以不需要使用事务
	gormConf := &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	}
	readDB, err := createReadDB(path, gormConf)
	if err != nil {
		return nil, nil, err
	}
	writeDB, err := createWriteDB(path, gormConf)
	if err != nil {
		return nil, nil, err
	}
	// ----- 启用共享缓存插件 -----
	plugin, err := cache.GetOtterCacheDBPluginInstance()
	if err != nil {
		return nil, nil, err
	}
	err = readDB.Use(plugin)
	if err != nil {
		return nil, nil, err
	}
	err = writeDB.Use(plugin)
	if err != nil {
		return nil, nil, err
	}
	return readDB, writeDB, nil
}

// SetDefaultPragmas defines some sqlite pragmas for good performance and litestream compatibility
// https://highperformancesqlite.com/articles/sqlite-recommended-pragmas
// https://litestream.io/tips/
// copied from https://github.com/bihe/monorepo
func SetDefaultPragmas(db *sql.DB) error {
	var (
		stmt string
		val  string
	)
	defaultPragmas := map[string]string{
		"journal_mode": "wal",   // https://www.sqlite.org/pragma.html#pragma_journal_mode
		"busy_timeout": "5000",  // https://www.sqlite.org/pragma.html#pragma_busy_timeout
		"synchronous":  "1",     // NORMAL --> https://www.sqlite.org/pragma.html#pragma_synchronous
		"cache_size":   "10000", // 10000 pages = 40MB --> https://www.sqlite.org/pragma.html#pragma_cache_size
		// 外键的暂时弃用，反正咱也不用外键（乐）
		// "foreign_keys": "1",     // 1(bool) --> https://www.sqlite.org/pragma.html#pragma_foreign_keys
	}

	// set the pragmas
	for k := range defaultPragmas {
		stmt = fmt.Sprintf("pragma %s = %s", k, defaultPragmas[k])
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// validate the pragmas
	for k := range defaultPragmas {
		row := db.QueryRow(fmt.Sprintf("pragma %s", k))
		err := row.Scan(&val)
		if err != nil {
			return err
		}
		if val != defaultPragmas[k] {
			return fmt.Errorf("could not set pragma %s to %s", k, defaultPragmas[k])
		}
	}

	return nil
}
