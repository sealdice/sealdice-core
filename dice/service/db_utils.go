package service

import (
	"strings"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/logger"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/engine/sqlite"
	"sealdice-core/utils/spinner"
)

var log = zap.S().Named(logger.LogKeyDatabase)

// BYTES类
// 如果我们使用FirstOrCreate,不可避免的会遇到这样的问题：
// 传入的是BYTE数组，由于使用了any会被转换为[]int8,而gorm又不会处理这种数据，进而导致转换失败
// 通过强制设置一个封装，可以确认any的类型，进而避免转换失败

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
		vacuumDB, err := sqlite.SQLiteDBInit(path, true)
		// 数据库类型不是 SQLite 直接返回
		if !strings.Contains(vacuumDB.Name(), "sqlite") {
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
		err = sqlite.ConvertToIncrementalAutoVacuum(vacuumDB)
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

// DBIncrementalVacuum 对 SQLite 数据库执行增量页回收。
// 只有数据库处于 auto_vacuum=INCREMENTAL 时才会真正回收，否则跳过。
func DBIncrementalVacuum(operator engine.DatabaseOperator) {
	if operator == nil || operator.Type() != "sqlite" {
		return
	}
	dbs := map[string]*gorm.DB{
		"data":   operator.GetDataDB(constant.WRITE),
		"logs":   operator.GetLogDB(constant.WRITE),
		"censor": operator.GetCensorDB(constant.WRITE),
	}
	for name, db := range dbs {
		if db == nil {
			continue
		}
		reclaimed, err := sqlite.ReclaimIncrementalVacuum(db)
		if err != nil {
			log.Errorf("数据库 %s 增量回收失败：%v", name, err)
			continue
		}
		if reclaimed > 0 {
			log.Debugf("数据库 %s 增量回收完成，回收 %d 页", name, reclaimed)
		}
	}
}
