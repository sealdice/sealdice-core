package dbutil

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

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
