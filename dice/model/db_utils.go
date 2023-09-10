package model

import (
	"os"
	"path/filepath"
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
	return ok
}
