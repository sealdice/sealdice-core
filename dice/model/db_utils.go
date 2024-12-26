package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"

	"sealdice-core/dice/model/database"
	log "sealdice-core/utils/kratos"
	"sealdice-core/utils/spinner"
)

// BYTES类
// 如果我们使用FirstOrCreate,不可避免的会遇到这样的问题：
// 传入的是BYTE数组，由于使用了any会被转换为[]int8,而gorm又不会处理这种数据，进而导致转换失败
// 通过强制设置一个封装，可以确认any的类型，进而避免转换失败

// 定义一个新的类型 JSON，封装 []byte
type BYTE []byte

// Scan 实现 sql.Scanner 接口，用于扫描数据库中的 JSON 数据
func (j *BYTE) Scan(value interface{}) error {
	// 将数据库中的值转换为 []byte
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	// 将 []byte 赋值给 JSON 类型的指针
	*j = bytes
	return nil
}

// Value 实现 driver.Valuer 接口，用于将 JSON 类型存储到数据库中
func (j BYTE) Value() (driver.Value, error) {
	// 如果 BYTE 数据为空，则返回 nil
	if len(j) == 0 {
		return nil, nil //nolint:nilnil
	}
	// 返回原始的 []byte
	return []byte(j), nil
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
		vacuumDB, err := database.SQLiteDBInit(path, true)
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
