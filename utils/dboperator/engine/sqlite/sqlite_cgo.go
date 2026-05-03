//go:build cgo

package sqlite

import (
	"database/sql"
	"fmt"
	"runtime"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/utils/cache"
)

// Do not use the read DB handle inside a write transaction. SQLite will block
// on the writer and can deadlock the process.

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	path = fmt.Sprintf("file:%v?_txlock=immediate&_busy_timeout=15000", path)
	open, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
	path = fmt.Sprintf("file:%v?_txlock=immediate", path)
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
	path = fmt.Sprintf("file:%v?_txlock=immediate", path)
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
	writePool.SetMaxOpenConns(1)
	return writeDB, nil
}

func SQLiteDBRWInit(path string) (*gorm.DB, *gorm.DB, *cache.Plugin, error) {
	gormConf := gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Info),
		SkipDefaultTransaction: true,
	}
	readDB, err := createReadDB(path, gormConf)
	if err != nil {
		return nil, nil, nil, err
	}
	writeDB, err := createWriteDB(path, gormConf)
	if err != nil {
		return nil, nil, nil, err
	}

	plugin, err := cache.GetOtterCacheDBPluginInstance()
	if err != nil {
		return nil, nil, nil, err
	}
	err = readDB.Use(plugin)
	if err != nil {
		plugin.Close()
		return nil, nil, nil, err
	}
	err = writeDB.Use(plugin)
	if err != nil {
		plugin.Close()
		return nil, nil, nil, err
	}
	return readDB, writeDB, plugin, nil
}

// SetDefaultPragmas configures SQLite defaults that work well for the current
// deployment model and Litestream-compatible setups.
func SetDefaultPragmas(db *sql.DB) error {
	var (
		stmt string
		val  string
	)
	defaultPragmas := map[string]string{
		"journal_mode": "wal",
		"busy_timeout": "15000",
		"synchronous":  "1",
		"cache_size":   "536870912",
	}

	for k := range defaultPragmas {
		stmt = fmt.Sprintf("pragma %s = %s", k, defaultPragmas[k])
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

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
	if _, err := db.Exec("PRAGMA optimize=0x10002;"); err != nil {
		return fmt.Errorf("unable set optimize pragma: %w", err)
	}

	return nil
}
