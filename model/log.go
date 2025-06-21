package model

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type LogOneItemParquet struct {
	ID             uint64 `json:"id" gorm:"column:id" parquet:"id, type=UINT_64"`
	Nickname       string `json:"nickname" gorm:"column:nickname" parquet:"nickname, type=UTF8"`
	IMUserID       string `json:"IMUserId" gorm:"column:im_userid" parquet:"IMUserId, type=UTF8"`
	Time           int64  `json:"time" gorm:"column:time" parquet:"time, type=INT_64"`
	Message        string `json:"message" gorm:"column:message" parquet:"message, type=UTF8"`
	IsDice         bool   `json:"isDice" gorm:"column:is_dice" parquet:"isDice, type=BOOLEAN"`
	CommandID      int64  `json:"commandId" gorm:"column:command_id" parquet:"commandId, type=INT_64"`
	CommandInfoStr string `json:"-" gorm:"column:command_info" parquet:"commandInfo, type=UTF8"`
	UniformID      string `json:"uniformId" gorm:"column:user_uniform_id" parquet:"uniformId, type=UTF8"`
}

// 兼容旧版本的数据库设计
func (*LogOneItemParquet) TableName() string {
	return "log_items"
}

type LogOneItem struct {
	ID             uint64      `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	LogID          uint64      `json:"-" gorm:"column:log_id;index:idx_log_items_log_id"`
	GroupID        string      `gorm:"index:idx_log_items_group_id;column:group_id;index:idx_log_delete_by_id"`
	Nickname       string      `json:"nickname" gorm:"column:nickname"`
	IMUserID       string      `json:"IMUserId" gorm:"column:im_userid"`
	Time           int64       `json:"time" gorm:"column:time"`
	Message        string      `json:"message"  gorm:"column:message"`
	IsDice         bool        `json:"isDice"  gorm:"column:is_dice"`
	CommandID      int64       `json:"commandId"  gorm:"column:command_id"`
	CommandInfo    interface{} `json:"commandInfo" gorm:"-" parquet:"-"`
	CommandInfoStr string      `json:"-" gorm:"column:command_info"`
	// 这里的RawMsgID 真的什么都有可能
	RawMsgID    interface{} `json:"rawMsgId" gorm:"-" parquet:"-"`
	RawMsgIDStr string      `json:"-" gorm:"column:raw_msg_id;index:idx_raw_msg_id;index:idx_log_delete_by_id"`
	UniformID   string      `json:"uniformId" gorm:"column:user_uniform_id"`
	// 数据库里没有的
	Channel string `json:"channel" gorm:"-"`
	// 数据库里有，JSON里没有的
	// 允许default=NULL
	Removed  *int `gorm:"column:removed" json:"-"`
	ParentID *int `gorm:"column:parent_id" json:"-"`
}

// 兼容旧版本的数据库设计
func (*LogOneItem) TableName() string {
	return "log_items"
}

// BeforeSave 钩子函数: 查询前,interface{}转换为json
func (item *LogOneItem) BeforeSave(_ *gorm.DB) (err error) {
	// 将 CommandInfo 转换为 JSON 字符串保存到 CommandInfoStr
	if item.CommandInfo != nil {
		if data, err := json.Marshal(item.CommandInfo); err == nil {
			item.CommandInfoStr = string(data)
		} else {
			return err
		}
	}

	// 将 RawMsgID 转换为 string 字符串，保存到 RawMsgIDStr
	if item.RawMsgID != nil {
		item.RawMsgIDStr = fmt.Sprintf("%v", item.RawMsgID)
	}

	return nil
}

// AfterFind 钩子函数: 查询后,interface{}转换为json
func (item *LogOneItem) AfterFind(_ *gorm.DB) (err error) {
	// 将 CommandInfoStr 从 JSON 字符串反序列化为 CommandInfo
	if item.CommandInfoStr != "" {
		if err := json.Unmarshal([]byte(item.CommandInfoStr), &item.CommandInfo); err != nil {
			return err
		}
	}

	// 将 RawMsgIDStr string 直接赋值给 RawMsgID
	if item.RawMsgIDStr != "" {
		item.RawMsgID = item.RawMsgIDStr
	}

	return nil
}

type LogInfo struct {
	ID        uint64 `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Name      string `json:"name" gorm:"index:idx_log_group_id_name,unique"`
	GroupID   string `json:"groupId" gorm:"index:idx_logs_group;index:idx_log_group_id_name,unique"`
	CreatedAt int64  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt int64  `json:"updatedAt" gorm:"column:updated_at;index:idx_logs_update_at"`
	// 允许数据库NULL值
	// 原版代码中，此处标记了db:size，但实际上，该列并不存在。
	// 考虑到该处数据将会为未来log查询提供优化手段，保留该结构体定义，但不使用。
	// 使用GORM:<-:false 无写入权限，这样它就不会建库，但请注意，下面LogGetLogPage处，如果你查询出的名称不是size
	// 不能在这里绑定column，因为column会给你建立那一列。
	// TODO: 将这个字段使用上会不会比后台查询就JOIN更合适？
	Size *int `json:"size" gorm:"column:size"`
	// 数据库里有，json不展示的
	// 允许数据库NULL值（该字段当前不使用）
	Extra *string `json:"-" gorm:"column:extra"`
	// 原本标记为：测试版特供，由于原代码每次都会执行，故直接启用此处column记录。
	UploadURL  string `json:"-" gorm:"column:upload_url"`  // 测试版特供
	UploadTime int    `json:"-" gorm:"column:upload_time"` // 测试版特供
}

func (*LogInfo) TableName() string {
	return "logs"
}

// ADD FROM MYSQL

type LogInfoHookMySQL struct {
	ID         uint64  `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Name       string  `json:"name" gorm:"column:name"`
	GroupID    string  `json:"groupId" gorm:"column:group_id"`
	CreatedAt  int64   `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt  int64   `json:"updatedAt" gorm:"column:updated_at"`
	Size       *int    `json:"size" gorm:"<-:false"`
	Extra      *string `json:"-" gorm:"column:extra"`
	UploadURL  string  `json:"-" gorm:"column:upload_url"`
	UploadTime int     `json:"-" gorm:"column:upload_time"`
}

func (*LogInfoHookMySQL) TableName() string {
	return "logs"
}

type LogOneItemHookMySQL struct {
	ID             uint64      `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	LogID          uint64      `json:"-" gorm:"column:log_id"`
	GroupID        string      `gorm:"column:group_id"`
	Nickname       string      `json:"nickname" gorm:"column:nickname"`
	IMUserID       string      `json:"IMUserId" gorm:"column:im_userid"`
	Time           int64       `json:"time" gorm:"column:time"`
	Message        string      `json:"message"  gorm:"column:message"`
	IsDice         bool        `json:"isDice"  gorm:"column:is_dice"`
	CommandID      int64       `json:"commandId"  gorm:"column:command_id"`
	CommandInfo    interface{} `json:"commandInfo" gorm:"-"`
	CommandInfoStr string      `json:"-" gorm:"column:command_info"`
	RawMsgID       interface{} `json:"rawMsgId" gorm:"-"`
	RawMsgIDStr    string      `json:"-" gorm:"column:raw_msg_id"`
	UniformID      string      `json:"uniformId" gorm:"column:user_uniform_id"`
	Channel        string      `json:"channel" gorm:"-"`
	Removed        *int        `gorm:"column:removed" json:"-"`
	ParentID       *int        `gorm:"column:parent_id" json:"-"`
}

func (*LogOneItemHookMySQL) TableName() string {
	return "log_items"
}
