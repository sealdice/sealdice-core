//go:build cgo

package sqlite

import (
	"database/sql"
	"fmt"
	"sync"

	sqlite3 "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"sealdice-core/logger"
)

var registerHookedDriverOnce sync.Once

// hookedDialector 返回一个使用带连接钩子驱动的 dialector。
// 连接钩子会在每个新建连接上执行连接级 pragma，避免连接池后续新建的连接退回默认值。
func hookedDialector(dsn string) gorm.Dialector {
	registerHookedDriverOnce.Do(func() {
		sql.Register(hookedDriverName, &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				for _, stmt := range connPragmaStatements() {
					if _, err := conn.Exec(stmt, nil); err != nil {
						return err
					}
				}
				return nil
			},
		})
	})
	return sqlite.New(sqlite.Config{DriverName: hookedDriverName, DSN: dsn})
}

// 警告：不要在一个事务（写事务）里使用读的DB！否则读的DB会发现有人在写而锁住，从而死锁。

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	// 使用即时事务
	path = fmt.Sprintf("file:%v?_txlock=immediate&_busy_timeout=15000", path)
	open, err := gorm.Open(hookedDialector(path), &gorm.Config{
		Logger: logger.DefaultSealLogger,
	})
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
	// _txlock=deferred 读连接不应获取写锁
	path = readDBDSN(path)
	// ---- 创建读连接 -----
	readDB, err := gorm.Open(hookedDialector(path), &gormConf)
	if err != nil {
		return nil, err
	}
	readPool, err := readDB.DB()
	if err != nil {
		return nil, err
	}
	if err = ensureIncrementalAutoVacuum(readPool); err != nil {
		return nil, err
	}
	err = SetDefaultPragmas(readPool)
	if err != nil {
		return nil, err
	}
	configureReadPool(readPool)
	return readDB, nil
}

func createWriteDB(path string, gormConf gorm.Config) (*gorm.DB, error) {
	// ---- 创建写连接 -----
	path = writeDBDSN(path)
	writeDB, err := gorm.Open(hookedDialector(path), &gormConf)
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
	configureWritePool(writePool) // only use one active connection for writing
	return writeDB, nil
}

func SQLiteDBRWInit(path string) (*gorm.DB, *gorm.DB, error) {
	// 由于现在我们只有一个写入连接，所以不需要使用事务
	gormConf := gorm.Config{
		Logger:                 logger.DefaultSealLogger,
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
	return readDB, writeDB, nil
}
