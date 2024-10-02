package model

import (
	"fmt"
	"time"

	"golang.org/x/time/rate"
	"gorm.io/gorm"

	ds "github.com/sealdice/dicescript"
)

// GroupInfo 模型
type GroupInfo struct {
	ID        string `gorm:"column:id;primaryKey"` // 主键，字符串类型
	CreatedAt int    `gorm:"column:created_at"`    // 创建时间
	UpdatedAt int64  `gorm:"column:updated_at"`    // 更新时间，int64类型
	Data      []byte `gorm:"column:data"`          // BLOB 类型字段，使用 []byte 表示
}

func (GroupInfo) TableName() string {
	return "group_info"
}

// GroupInfoListGet 使用 GORM 实现，遍历 group_info 表中的数据并调用回调函数
func GroupInfoListGet(db *gorm.DB, callback func(id string, updatedAt int64, data []byte)) error {
	// 创建一个保存查询结果的结构体
	var results []struct {
		ID        string `gorm:"column:id"`         // 字段 id
		UpdatedAt *int64 `gorm:"column:updated_at"` // 由于可能存在 NULL，定义为指针类型
		Data      []byte `gorm:"column:data"`       // 字段 data
	}

	// 使用 GORM 查询 group_info 表中的 id, updated_at, data 列
	err := db.Table("group_info").Select("id, updated_at, data").Find(&results).Error
	if err != nil {
		// 如果查询发生错误，返回错误信息
		return err
	}

	// 遍历查询结果
	for _, result := range results {
		var updatedAt int64

		// 如果 updatedAt 是 NULL，需要跳过该字段
		if result.UpdatedAt != nil {
			updatedAt = *result.UpdatedAt
		}

		// 调用回调函数，传递 id, updatedAt, data
		callback(result.ID, updatedAt, result.Data)
	}

	// 返回 nil 表示操作成功
	return nil
}

// GroupInfoSave 保存群组信息，兼容多种数据库
func GroupInfoSave(db *gorm.DB, groupID string, updatedAt int64, data []byte) error {
	// 创建一个 GroupInfo 实例
	groupInfo := GroupInfo{
		ID:        groupID,
		UpdatedAt: updatedAt,
		Data:      data,
	}

	// 使用 GORM 的 Save 方法：Save 会根据主键决定是插入还是更新记录
	// 如果记录存在则更新，不存在则插入。
	err := db.Save(&groupInfo).Error
	return err
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name   string `yaml:"name" jsbind:"name" gorm:"column:name"` // 玩家昵称
	UserID string `yaml:"userId" jsbind:"userId" gorm:"column:user_id;index:idx_group_player_info_user_id"`
	// 非数据库信息：是否在群内
	InGroup         bool  `yaml:"inGroup" gorm:"-"`                                                         // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime int64 `yaml:"lastCommandTime" jsbind:"lastCommandTime" gorm:"column:last_command_time"` // 上次发送指令时间
	// 非数据库信息
	RateLimiter *rate.Limiter `yaml:"-" gorm:"-"` // 限速器
	// 非数据库信息
	RateLimitWarned     bool   `yaml:"-" gorm:"-"`                                                                            // 是否已经警告过限速
	AutoSetNameTemplate string `yaml:"autoSetNameTemplate" jsbind:"autoSetNameTemplate" gorm:"column:auto_set_name_template"` // 名片模板

	// level int 权限
	DiceSideNum int `yaml:"diceSideNum" gorm:"column:dice_side_num"` // 面数，为0时等同于d100
	// 非数据库信息
	ValueMapTemp *ds.ValueMap `yaml:"-"  gorm:"-"` // 玩家的群内临时变量
	// ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	// 非数据库信息
	TempValueAlias *map[string][]string `yaml:"-"  gorm:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	// 非数据库信息
	UpdatedAtTime int64 `yaml:"-" json:"-"  gorm:"-"`
	// 非数据库信息
	RecentUsedTime int64 `yaml:"-" json:"-"  gorm:"-"`
	// 缺少信息
	CreatedAt int    `yaml:"-" json:"-" gorm:"column:created_at"` // 创建时间
	UpdatedAt int    `yaml:"-" json:"-" gorm:"column:updated_at"` // 更新时间
	GroupID   string `yaml:"-" json:"-" gorm:"index:idx_group_player_info_group_id"`
}

// 兼容设置
func (GroupPlayerInfoBase) TableName() string {
	return "group_player_info"
}

// GroupPlayerNumGet 获取指定群组的玩家数量
func GroupPlayerNumGet(db *gorm.DB, groupID string) (int64, error) {
	var count int64

	// 使用 GORM 的 Table 方法指定表名进行查询
	// db.Table("表名").Where("条件").Count(&count) 是通用的 GORM 用法
	// 将 group_id 作为查询条件
	err := db.Table("group_player_info").Where("group_id = ?", groupID).Count(&count).Error
	if err != nil {
		// 如果查询出现错误，返回错误信息
		return 0, err
	}

	// 返回统计的数量
	return count, nil
}

// GroupPlayerInfoGet 获取指定群组中的玩家信息
func GroupPlayerInfoGet(db *gorm.DB, groupID string, playerID string) *GroupPlayerInfoBase {
	var ret GroupPlayerInfoBase

	// 使用 GORM 查询数据并绑定到结构体中
	// db.Table("表名").Where("条件").First(&ret) 查询一条数据并映射到结构体
	err := db.Table("group_player_info").
		Where("group_id = ? AND user_id = ?", groupID, playerID).
		Select("name, last_command_time, auto_set_name_template, dice_side_num").
		Scan(&ret).Error

	// 如果查询发生错误，打印错误并返回 nil
	if err != nil {
		fmt.Printf("error getting group player info: %s", err.Error())
		return nil
	}

	// 如果查询到的数据为空，返回 nil
	if db.RowsAffected == 0 {
		return nil
	}

	// 将 playerID 赋值给结构体中的 UserID 字段
	ret.UserID = playerID

	// 返回查询结果
	return &ret
}

// GroupPlayerInfoSave 保存玩家信息，使用 REPLACE INTO 语句
func GroupPlayerInfoSave(db *gorm.DB, groupID string, playerID string, info *GroupPlayerInfoBase) error {
	// 使用 GORM 的 Create 方法插入新的记录
	// 设置 groupID 和 playerID 到 info 中
	info.UserID = playerID
	info.GroupID = groupID
	info.UpdatedAt = int(time.Now().Unix()) // 更新当前时间为 UpdatedAt

	// 直接使用 Create 方法插入记录，ID 字段由数据库自增生成
	err := db.Table("group_player_info").Create(info).Error

	// 如果保存操作出现错误，返回错误信息
	if err != nil {
		return err
	}

	// 返回 nil 表示操作成功
	return nil
}
