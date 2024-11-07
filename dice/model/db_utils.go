package model

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	log "sealdice-core/utils/kratos"
	"sealdice-core/utils/spinner"
)

// DBCacheDelete 删除SQLite数据库缓存文件
// TODO: 判断缓存是否应该被删除
func DBCacheDelete() bool {
	dataDir := "./data/default"

	tryDelete := func(fn string) bool {
		fnPath, _ := filepath.Abs(filepath.Join(dataDir, fn))
		if _, err := os.Stat(fnPath); err != nil {
			// 文件不在了，就当作删除成功
			return true
		}
		return os.Remove(fnPath) == nil
	}

	// 非 Windows 系统不删除缓存
	if runtime.GOOS != "windows" {
		return true
	}
	ok := true
	if ok {
		ok = tryDelete("data.db-shm")
	}
	if ok {
		ok = tryDelete("data.db-wal")
	}
	if ok {
		ok = tryDelete("data-logs.db-shm")
	}
	if ok {
		ok = tryDelete("data-logs.db-wal")
	}
	if ok {
		ok = tryDelete("data-censor.db-shm")
	}
	if ok {
		ok = tryDelete("data-censor.db-wal")
	}
	return ok
}

// DBVacuum 整理数据库
func DBVacuum() {
	done := make(chan interface{}, 1)
	log.Info("开始进行数据库整理")

	go spinner.WithLines(done, 3, 10)
	defer func() {
		done <- struct{}{}
	}()

	wg := sync.WaitGroup{}
	wg.Add(3)

	vacuum := func(path string, wg *sync.WaitGroup) {
		defer wg.Done()
		// 使用 GORM 初始化数据库
		vacuumDB, err := _SQLiteDBInit(path, true)
		// 数据库类型不是 SQLite 直接返回
		if !strings.Contains(vacuumDB.Dialector.Name(), "sqlite") {
			return
		}
		defer func() {
			rawdb, err2 := vacuumDB.DB()
			if err2 != nil {
				return
			}
			err = rawdb.Close()
			if err != nil {
				return
			}
		}()
		if err != nil {
			log.Errorf("清理 %q 时出现错误：%v", path, err)
			return
		}
		err = vacuumDB.Exec("VACUUM;").Error
		if err != nil {
			log.Errorf("清理 %q 时出现错误：%v", path, err)
		}
	}

	go vacuum("./data/default/data.db", &wg)
	go vacuum("./data/default/data-logs.db", &wg)
	go vacuum("./data/default/data-censor.db", &wg)

	wg.Wait()

	log.Info("数据库整理完成")
}
