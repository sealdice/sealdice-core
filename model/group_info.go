package model

import (
	"github.com/sealdice/dicescript"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// GroupInfo 模型 已被派恩废弃
type GroupInfo struct {
	ID        string `gorm:"column:id;primaryKey"` // 主键，字符串类型
	CreatedAt int64  `gorm:"column:created_at"`    // 创建时间
	UpdatedAt *int64 `gorm:"column:updated_at"`    // 更新时间，int64类型
	Data      []byte `gorm:"column:data"`          // BLOB 类型字段，使用 []byte 表示
}

func (*GroupInfo) TableName() string {
	return "group_info"
}

// GroupInfoDB
type GroupInfoDB struct {
	ID                  string              `gorm:"primarykey"`
	CreatedAt           int64               `json:"created_at"`
	UpdatedAt           int64               `json:"updated_at"`
	DeletedAt           gorm.DeletedAt      `gorm:"index"`
	Active              bool                `gorm:"column:active"                 json:"active"`
	ActivatedExtList    []string            `gorm:"column:activated_ext_list;type:TEXT;serializer:json"   json:"activatedExtList"`
	InactivatedExtSet   []string            `gorm:"column:inactivated_ext_set;type:TEXT;serializer:json"  json:"inactivatedExtSet"`
	GroupId             string              `gorm:"column:group_id"               json:"groupId"`
	GuildId             string              `gorm:"column:guild_id"               json:"guildId"`
	ChannelId           string              `gorm:"column:channel_id"             json:"channelId"`
	GroupName           string              `gorm:"column:group_name"             json:"groupName"`
	DiceIdActiveMap     map[string]bool     `gorm:"column:dice_id_active_map;type:TEXT;serializer:json"  json:"diceIdActiveMap"`
	DiceIdExistsMap     map[string]bool     `gorm:"column:dice_id_exists_map;type:TEXT;serializer:json"  json:"diceIdExistsMap"`
	BotList             map[string]bool     `gorm:"column:bot_list;type:TEXT;serializer:json"                json:"botList"`
	DiceSideNum         int                 `gorm:"column:dice_side_num"          json:"diceSideNum"`
	DiceSideExpr        string              `gorm:"column:dice_side_expr"         json:"diceSideExpr"`
	System              string              `gorm:"column:system"                  json:"system"`
	HelpPackages        []string            `gorm:"column:help_packages;type:TEXT;serializer:json"           json:"helpPackages"`
	CocRuleIndex        int                 `gorm:"column:coc_rule_index"          json:"cocRuleIndex"`
	LogCurName          string              `gorm:"column:log_cur_name"            json:"logCurName"`
	LogOn               bool                `gorm:"column:log_on"                  json:"logOn"`
	RecentDiceSendTime  int64               `gorm:"column:recent_dice_send_time"   json:"recentDiceSendTime"`
	ShowGroupWelcome    bool                `gorm:"column:show_group_welcome"      json:"showGroupWelcome"`
	GroupWelcomeMessage string              `gorm:"column:group_welcome_message"   json:"groupWelcomeMessage"`
	EnteredTime         int64               `gorm:"column:entered_time"            json:"enteredTime"`
	InviteUserId        string              `gorm:"column:invite_user_id"          json:"inviteUserId"`
	DefaultHelpGroup    string              `gorm:"column:default_help_group"      json:"defaultHelpGroup"`
	PlayerGroups        map[string][]string `gorm:"column:player_groups;type:TEXT;serializer:json"      json:"playerGroups"`
	ExtAppliedVersion   int                 `gorm:"column:ext_applied_version"     json:"extAppliedVersion"`
}

func (*GroupInfoDB) TableName() string {
	// 使用V2版本和V1区分开，不破坏V1版本的数据结构
	return "group_info_v2"
}

// GroupPlayerInfoBase 群内玩家信息 迁移自 im_session.go
type GroupPlayerInfoBase struct {
	// 补充这个字段，从而保证包含主键ID
	ID     uint   `gorm:"column:id;primaryKey;autoIncrement"                                                               jsbind:"-"      yaml:"-"`    // 主键ID字段，自增
	Name   string `gorm:"column:name"                                                                                      jsbind:"name"   yaml:"name"` // 玩家昵称
	UserID string `gorm:"column:user_id;index:idx_group_player_info_user_id; uniqueIndex:idx_group_player_info_group_user" jsbind:"userId" yaml:"userId"`
	// 非数据库信息：是否在群内
	InGroup         bool  `gorm:"-"                        yaml:"inGroup"`                                  // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime int64 `gorm:"column:last_command_time" jsbind:"lastCommandTime" yaml:"lastCommandTime"` // 上次发送指令时间
	// 非数据库信息
	RateLimiter *rate.Limiter `gorm:"-" yaml:"-"` // 限速器
	// 非数据库信息
	RateLimitWarned     bool   `gorm:"-"                             yaml:"-"`                                                // 是否已经警告过限速
	AutoSetNameTemplate string `gorm:"column:auto_set_name_template" jsbind:"autoSetNameTemplate" yaml:"autoSetNameTemplate"` // 名片模板

	// level int 权限
	DiceSideNum int `gorm:"column:dice_side_num" yaml:"diceSideNum"` // 面数，为0时等同于d100
	// 非数据库信息
	ValueMapTemp *dicescript.ValueMap `gorm:"-" yaml:"-"` // 玩家的群内临时变量
	// ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	// 非数据库信息
	UpdatedAtTime int64 `gorm:"-" json:"-" yaml:"-"`
	// 非数据库信息
	RecentUsedTime int64 `gorm:"-" json:"-" yaml:"-"`
	// 缺少信息 -> 这边原来就是int吗？
	CreatedAt int    `gorm:"column:created_at"                                                                                  json:"-" yaml:"-"` // 创建时间
	UpdatedAt int    `gorm:"column:updated_at"                                                                                  json:"-" yaml:"-"` // 更新时间
	GroupID   string `gorm:"column:group_id;index:idx_group_player_info_group_id; uniqueIndex:idx_group_player_info_group_user" json:"-" yaml:"-"`
}

// 兼容设置
func (GroupPlayerInfoBase) TableName() string {
	return "group_player_info"
}
