package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// timeFormat 定义了时间格式，用于时间的字符串表示和解析
const timeFormat = "2006-01-02 15:04:05"

// timezone 定义了时间的时区，用于将时间转换到特定的时区
const timeZone = "Asia/Shanghai"

// Time 是 time.Time 的别名，用于添加自定义的序列化和反序列化行为
type Time time.Time

// MarshalJSON 实现了 Time 类型的自定义 JSON 序列化
// 返回值:
// - []byte: 序列化后的 JSON 字符串
// - error: 错误信息
func (t Time) MarshalJSON() ([]byte, error) {
	// 创建一个足够大的字节切片来存储时间格式
	b := make([]byte, 0, len(timeFormat)+2)
	// 添加双引号开始
	b = append(b, '"')
	// 将时间格式化并追加到字节切片中
	b = time.Time(t).AppendFormat(b, timeFormat)
	// 添加双引号结束
	b = append(b, '"')
	// 返回序列化后的 JSON 字符串和错误信息
	return b, nil
}

// UnmarshalJSON 实现了 Time 类型的自定义 JSON 反序列化
// 参数:
// - data: 需要反序列化的 JSON 数据
// 返回值:
// - error: 错误信息
func (t *Time) UnmarshalJSON(data []byte) (err error) {
	// 检查数据是否为空或空字符串
	if len(data) == 0 || string(data) == `""` {
		// 如果数据为空或空字符串，设置时间为默认初始值 9999-12-31
		*t = Time(time.Date(9999, 12, 31, 0, 0, 0, 0, time.Local))
		// 返回 nil 表示成功
		return nil
	}
	// 解析 JSON 数据中的时间字符串
	now, err := time.ParseInLocation(`"`+timeFormat+`"`, string(data), time.Local)
	// 检查解析是否出错
	if err != nil {
		// 返回解析错误
		return err
	}
	// 设置解析后的时间
	*t = Time(now)
	// 返回 nil 表示成功
	return
}

// String 实现了 Time 类型的自定义字符串表示
// 返回值:
// - string: 时间的字符串表示
func (t Time) String() string {
	// 格式化时间并返回字符串表示
	return time.Time(t).Format(timeFormat)
}

// Local 返回 Time 类型的时间在指定时区的表示
// 返回值:
// - time.Time: 转换后的本地时间
func (t Time) Local() time.Time {
	// 加载指定的时区
	loc, _ := time.LoadLocation(timeZone)
	// 将时间转换到指定的时区并返回
	return time.Time(t).In(loc)
}

// Value 实现了 Time 类型的数据库驱动值表示
// 返回值:
// - driver.Value: 数据库驱动值
// - error: 错误信息
func (t Time) Value() (driver.Value, error) {
	// 定义零时间
	var zeroTime time.Time
	// 将 Time 类型转换为 time.Time 类型
	var ti = time.Time(t)
	// 检查时间是否为零时间
	if ti.UnixNano() == zeroTime.UnixNano() {
		// 如果是零时间，返回 nil 表示未初始化的数据
		return nil, nil //nolint:nilnil
	}
	// 返回时间的数据库驱动值
	return ti, nil
}

// Scan 实现了 Time 类型的数据库驱动值反序列化
// 参数:
// - v: 数据库驱动值
// 返回值:
// - error: 错误信息
func (t *Time) Scan(v interface{}) error {
	// 尝试将数据库驱动值转换为 time.Time 类型
	value, ok := v.(time.Time)
	// 检查转换是否成功
	if ok {
		// 如果转换成功，设置时间
		*t = Time(value)
		// 返回 nil 表示成功
		return nil
	}
	// 如果转换失败，返回错误信息
	return fmt.Errorf("can not convert %v to timestamp", v)
}
