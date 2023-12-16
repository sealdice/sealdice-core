package model

import (
	"fmt"
	"os"
	"path/filepath"
	"sealdice-core/utils/spinner"
	"sync"
)

func DBCacheDelete() bool {
	// d.BaseConfig.DataDir
	dataDir := "./data/default"

	tryDelete := func(fn string) bool {
		fnPath, _ := filepath.Abs(filepath.Join(dataDir, fn))
		if _, err := os.Stat(fnPath); err != nil {
			// 文件不在了，就当作删除成功
			return true
		}
		fmt.Printf("删除文件 %s", fnPath)
		return os.Remove(fnPath) == nil
	}

	ok := true
	if ok {
		ok = tryDelete("data.db-shm")
	}
	if ok {
		tryDelete("data.db-wal")
	}
	if ok {
		tryDelete("data-logs.db-shm")
	}
	if ok {
		tryDelete("data-logs.db-wal")
	}
	if ok {
		tryDelete("data-censor.db-shm")
	}
	if ok {
		tryDelete("data-censor.db-wal")
	}
	return ok
}

func DBVacuum() {
	done := make(chan interface{}, 1)
	fmt.Println("开始进行数据库整理")

	go spinner.WithLines(done, 3, 10)
	defer func() {
		done <- struct{}{}
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)

	vacuum := func(path string, wg *sync.WaitGroup) {
		defer wg.Done()
		db, err := _SQLiteDBInit(path, true)
		defer func() { _ = db.Close() }()
		if err != nil {
			fmt.Printf("清理 %q 时出现错误：%v", path, err)
			return
		}
		_, err = db.Exec("VACUUM;")
		if err != nil {
			fmt.Printf("清理 %q 时出现错误：%v", path, err)
		}
	}

	go vacuum("./data/default/data.db", &wg)
	go vacuum("./data/default/data-logs.db", &wg)
	go vacuum("./data/default/data-censor.db", &wg)

	wg.Wait()

	fmt.Println("\n数据库整理完成")
}
