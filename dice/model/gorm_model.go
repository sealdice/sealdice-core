package model

import (
	"gorm.io/gorm"
)

// GroupPlayerInfo 模型
type GroupPlayerInfo struct {
	ID                  int    `gorm:"primarykey;autoIncrement"`
	GroupID             string `gorm:"index:idx_group_player_info_group_id"`
	UserID              string `gorm:"index:idx_group_player_info_user_id"`
	Name                string
	CreatedAt           int
	UpdatedAt           int
	LastCommandTime     int
	AutoSetNameTemplate string
	DiceSideNum         string
	// 定义联合唯一索引
	gorm.Model
}

// GroupInfo 模型
type GroupInfo struct {
	ID        string `gorm:"primarykey"`
	CreatedAt int
	UpdatedAt int
	Data      []byte // 使用[]byte表示BLOB类型
}

// BanInfo 模型
type BanInfo struct {
	ID           string `gorm:"primarykey"`
	BanUpdatedAt int    `gorm:"index:idx_ban_info_ban_updated_at"`
	UpdatedAt    int    `gorm:"index:idx_ban_info_updated_at"`
	Data         []byte // 使用[]byte表示BLOB类型
}

type EndpointInfo struct {
	UserID      string `gorm:"column:user_id;"`
	CmdNum      int64  `gorm:"column:cmd_num;"`
	CmdLastTime int64  `gorm:"column:cmd_last_time;"`
	OnlineTime  int64  `gorm:"column:online_time;"`
	UpdatedAt   int64  `gorm:"column:updated_at;"`
}

// Attrs 模型
type Attrs struct {
	ID             string `gorm:"primarykey"`
	Data           []byte // 使用[]byte表示BYTEA类型
	AttrsType      string `gorm:"index:idx_attrs_attrs_type_id"`
	BindingSheetID string `gorm:"default:'';index:idx_attrs_binding_sheet_id"`
	Name           string `gorm:"default:''"`
	OwnerID        string `gorm:"default:'';index:idx_attrs_owner_id_id"`
	SheetType      string `gorm:"default:''"`
	IsHidden       bool   `gorm:"default:false"`
	CreatedAt      int    `gorm:"default:0"`
	UpdatedAt      int    `gorm:"default:0"`
}

// LOG
// Logs 模型
type Logs struct {
	ID         int `gorm:"primarykey;autoIncrement"`
	Name       string
	GroupID    string `gorm:"index:idx_logs_group"`
	Extra      string
	CreatedAt  int
	UpdatedAt  int
	UploadURL  string `gorm:"-"` // 测试版特供
	UploadTime int    `gorm:"-"` // 测试版特供
	gorm.Model
}

// LogItems 模型
type LogItems struct {
	ID            int `gorm:"primarykey;autoIncrement"`
	LogID         int
	GroupID       string `gorm:"index:idx_log_items_group_id"`
	Nickname      string
	IMUserID      string
	Time          int
	Message       string
	IsDice        int
	CommandID     int
	CommandInfo   string
	RawMsgID      string
	UserUniformID string
	Removed       int
	ParentID      int `gorm:"index:idx_log_items_log_id"`
}

// CensorLog 模型
type CensorLogGorm struct {
	ID             int `gorm:"primarykey;autoIncrement"`
	MsgType        string
	UserID         string `gorm:"index:idx_censor_log_user_id"`
	GroupID        string
	Content        string
	SensitiveWords string
	HighestLevel   int `gorm:"index:idx_censor_log_level"`
	CreatedAt      int
	ClearMark      bool
}
