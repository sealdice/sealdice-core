//go:build !cgo

package sqlite

import (
	"database/sql"
	"fmt"
	"runtime"

	_ "github.com/ncruces/go-sqlite3/embed"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/utils/cache"
)

// 警告：不要在一个事务（写事务）里使用读的DB！否则读的DB会发现有人在写而锁住，从而死锁。

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	// 使用即时事务
	path = fmt.Sprintf("file:%v?_txlock=immediate&_busy_timeout=15000", path)
	open, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	// Enable Cache Mode
	open, err = cache.GetOtterCacheDB(open)
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

func createReadDB(path string, gormConf gorm.Config) (*gorm.DB, error) {
	// _txlock=immediate 解决BEGIN IMMEDIATELY
	path = fmt.Sprintf("file:%v?_txlock=immediate", path)
	// ---- 创建读连接 -----
	readDB, err := gorm.Open(sqlite.Open(path), &gormConf)
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

func createWriteDB(path string, gormConf gorm.Config) (*gorm.DB, error) {
	// 注意基于wasm的版本必须添加file:
	path = fmt.Sprintf("file:%v?_txlock=immediate", path)
	// ---- 创建写连接 -----
	writeDB, err := gorm.Open(sqlite.Open(path), &gormConf)
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
	gormConf := gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Info),
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
// add PRAGMA optimize=0x10002; from https://github.com/Palats/mastopoof
func SetDefaultPragmas(db *sql.DB) error {
	var (
		stmt string
		val  string
	)
	// 外键的暂时弃用，反正咱也不用外键536870912
	// "foreign_keys": "1",     // 1(bool) --> https://www.sqlite.org/pragma.html#pragma_foreign_keys
	defaultPragmas := map[string]string{
		"journal_mode": "wal",   // https://www.sqlite.org/pragma.html#pragma_journal_mode
		"busy_timeout": "15000", // https://www.sqlite.org/pragma.html#pragma_busy_timeout
		// 在 WAL 模式下使用 synchronous=NORMAL 提交的事务可能会在断电或系统崩溃后回滚。
		// 无论同步设置或日志模式如何，事务在应用程序崩溃时都是持久的。
		// 对于在 WAL 模式下运行的大多数应用程序来说，synchronous=NORMAL 设置是一个不错的选择。
		"synchronous": "1",         // NORMAL --> https://www.sqlite.org/pragma.html#pragma_synchronous
		"cache_size":  "536870912", // 536870912 = 512MB --> https://www.sqlite.org/pragma.html#pragma_cache_size
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
	// 这个不能在上面，因为他没有任何返回值
	// Setup some regular optimization according to sqlite doc:
	//  https://www.sqlite.org/lang_analyze.html
	if _, err := db.Exec("PRAGMA optimize=0x10002;"); err != nil {
		return fmt.Errorf("unable set optimize pragma: %w", err)
	}

	return nil
}
