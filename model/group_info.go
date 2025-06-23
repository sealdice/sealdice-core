package model

import (
	"github.com/sealdice/dicescript"
	"golang.org/x/time/rate"
)

// GroupInfo 模型
type GroupInfo struct {
	ID        string `gorm:"column:id;primaryKey"` // 主键，字符串类型
	CreatedAt int64  `gorm:"column:created_at"`    // 创建时间
	UpdatedAt *int64 `gorm:"column:updated_at"`    // 更新时间，int64类型
	Data      []byte `gorm:"column:data"`          // BLOB 类型字段，使用 []byte 表示
}

func (*GroupInfo) TableName() string {
	return "group_info"
}

// GroupPlayerInfoBase 群内玩家信息 迁移自 im_session.go
type GroupPlayerInfoBase struct {
	// 补充这个字段，从而保证包含主键ID
	ID     uint   `yaml:"-" jsbind:"-" gorm:"column:id;primaryKey;autoIncrement"` // 主键ID字段，自增
	Name   string `yaml:"name" jsbind:"name" gorm:"column:name"`                  // 玩家昵称
	UserID string `yaml:"userId" jsbind:"userId" gorm:"column:user_id;index:idx_group_player_info_user_id; uniqueIndex:idx_group_player_info_group_user"`
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
	ValueMapTemp *dicescript.ValueMap `yaml:"-"  gorm:"-"` // 玩家的群内临时变量
	// ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	// 非数据库信息
	TempValueAlias *map[string][]string `yaml:"-"  gorm:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	// 非数据库信息
	UpdatedAtTime int64 `yaml:"-" json:"-"  gorm:"-"`
	// 非数据库信息
	RecentUsedTime int64 `yaml:"-" json:"-"  gorm:"-"`
	// 缺少信息 -> 这边原来就是int吗？
	CreatedAt int    `yaml:"-" json:"-" gorm:"column:created_at"` // 创建时间
	UpdatedAt int    `yaml:"-" json:"-" gorm:"column:updated_at"` // 更新时间
	GroupID   string `yaml:"-" json:"-" gorm:"column:group_id;index:idx_group_player_info_group_id; uniqueIndex:idx_group_player_info_group_user"`
}

// 兼容设置
func (GroupPlayerInfoBase) TableName() string {
	return "group_player_info"
}
