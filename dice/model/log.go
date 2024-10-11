package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type LogOne struct {
	// Version int           `json:"version,omitempty"`
	Items []LogOneItem `json:"items"`
	Info  LogInfo      `json:"info"`
}

type LogOneItem struct {
	ID             uint64      `json:"id" db:"id" gorm:"primaryKey;autoIncrement;column:id"`
	LogID          uint64      `json:"-" gorm:"column:log_id;index:idx_log_items_log_id"`
	GroupID        string      `gorm:"index:idx_log_items_group_id;column:group_id"`
	Nickname       string      `json:"nickname" db:"nickname" gorm:"column:nickname"`
	IMUserID       string      `json:"IMUserId" db:"im_userid" gorm:"column:im_userid"`
	Time           int64       `json:"time" db:"time" gorm:"column:time"`
	Message        string      `json:"message" db:"message" gorm:"column:message"`
	IsDice         bool        `json:"isDice" db:"is_dice" gorm:"column:is_dice"`
	CommandID      int64       `json:"commandId" db:"command_id" gorm:"column:command_id"`
	CommandInfo    interface{} `json:"commandInfo" db:"-" gorm:"-"`
	CommandInfoStr string      `json:"-" db:"command_info" gorm:"column:command_info"`
	// 这里的RawMsgID 真的什么都有可能
	RawMsgID    interface{} `json:"rawMsgId" db:"-" gorm:"-"`
	RawMsgIDStr string      `json:"-" db:"raw_msg_id" gorm:"column:raw_msg_id"`

	UniformID string `json:"uniformId" db:"user_uniform_id" gorm:"column:user_uniform_id"`
	// 数据库里没有的
	Channel string `json:"channel" gorm:"-"`
	// 数据库里有，JSON里没有的
	// 允许default=NULL
	Removed *int `gorm:"column:removed" json:"-"`
	// 允许default=NULL
	ParentID *int `gorm:"column:parent_id" json:"-"`
}

// 钩子函数: 保存前
func (item *LogOneItem) BeforeSave(tx *gorm.DB) (err error) {
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

// 钩子函数: 查询后
func (item *LogOneItem) AfterFind(tx *gorm.DB) (err error) {
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
	ID        uint64 `json:"id" db:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Name      string `json:"name" db:"name" gorm:"column:name;index:idx_log_group_id_name"`
	GroupID   string `json:"groupId" db:"groupId" gorm:"index:idx_logs_group;index:idx_log_group_id_name;column:group_id"`
	CreatedAt int64  `json:"createdAt" db:"created_at" gorm:"column:created_at"`
	UpdatedAt int64  `json:"updatedAt" db:"updated_at" gorm:"column:updated_at;index:idx_logs_update_at"`
	// 允许数据库NULL值
	Size *int `json:"size" db:"size" gorm:"column:size"`
	// 数据库里有，json不展示的
	// 允许数据库NULL值
	Extra *string `json:"-" gorm:"column:extra"`
	// 原本标记为：测试版特供，由于原代码每次都会执行，故直接启用此处column记录。
	UploadURL  string `json:"-" gorm:"column:upload_url"`  // 测试版特供
	UploadTime int    `json:"-" gorm:"column:upload_time"` // 测试版特供
}

// 兼容旧版本的数据库设计
func (*LogOneItem) TableName() string {
	return "log_items"
}

func (*LogInfo) TableName() string {
	return "logs"
}

// LogGetInfo 查询日志简略信息，使用通用函数替代SQLITE专属函数
func LogGetInfo(db *gorm.DB) ([]int, error) {
	lst := []int{0, 0, 0, 0}

	var maxID sql.NullInt64      // 使用sql.NullInt64来处理NULL值
	var itemsMaxID sql.NullInt64 // 使用sql.NullInt64来处理NULL值

	// 获取 logs 表的记录数和最大 ID
	err := db.Table("logs").Select("COUNT(*)").Scan(&lst[2]).Error
	if err != nil {
		return nil, err
	}

	err = db.Table("logs").Select("MAX(id)").Scan(&maxID).Error
	if err != nil {
		return nil, err
	}
	lst[0] = int(maxID.Int64)

	// 获取 log_items 表的记录数和最大 ID
	err = db.Table("log_items").Select("COUNT(*)").Scan(&lst[3]).Error
	if err != nil {
		return nil, err
	}

	err = db.Table("log_items").Select("MAX(id)").Scan(&itemsMaxID).Error
	if err != nil {
		return nil, err
	}
	lst[1] = int(itemsMaxID.Int64)

	return lst, nil
}

// Deprecated: replaced by page
func LogGetLogs(db *gorm.DB) ([]*LogInfo, error) {
	var lst []*LogInfo

	// 使用 GORM 查询 logs 表
	if err := db.Table("logs").
		Select("id, name, group_id, created_at, updated_at").
		Find(&lst).Error; err != nil {
		return nil, err
	}

	return lst, nil
}

type QueryLogPage struct {
	PageNum          int    `db:"page_num" query:"pageNum"`
	PageSize         int    `db:"page_siz" query:"pageSize"`
	Name             string `db:"name" query:"name"`
	GroupID          string `db:"group_id" query:"groupId"`
	CreatedTimeBegin string `db:"created_time_begin" query:"createdTimeBegin"`
	CreatedTimeEnd   string `db:"created_time_end" query:"createdTimeEnd"`
}

// LogGetLogPage 获取分页
func LogGetLogPage(db *gorm.DB, param *QueryLogPage) (int, []*LogInfo, error) {
	var lst []*LogInfo

	// 构建查询
	query := db.Table("logs").Select("logs.id, logs.name, logs.group_id, logs.created_at, logs.updated_at, COUNT(log_items.id) as size").
		Joins("LEFT JOIN log_items ON logs.id = log_items.log_id")

	// 添加条件
	if param.Name != "" {
		query = query.Where("logs.name LIKE ?", "%"+param.Name+"%")
	}
	if param.GroupID != "" {
		query = query.Where("logs.group_id LIKE ?", "%"+param.GroupID+"%")
	}
	if param.CreatedTimeBegin != "" {
		query = query.Where("logs.created_at >= ?", param.CreatedTimeBegin)
	}
	if param.CreatedTimeEnd != "" {
		query = query.Where("logs.created_at <= ?", param.CreatedTimeEnd)
	}

	// 获取总数
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, nil, err
	}

	// 分页查询
	query = query.Group("logs.id").Limit(param.PageSize).Offset((param.PageNum - 1) * param.PageSize)

	// 执行查询
	if err := query.Scan(&lst).Error; err != nil {
		return 0, nil, err
	}

	return int(count), lst, nil
}

// LogGetList 获取列表
func LogGetList(db *gorm.DB, groupID string) ([]string, error) {
	var lst []string

	// 执行查询
	if err := db.Table("logs").
		Select("name").
		Where("group_id = ?", groupID).
		Order("updated_at DESC").
		Pluck("name", &lst).Error; err != nil {
		return nil, err
	}

	return lst, nil
}

// LogGetIDByGroupIDAndName 获取ID
func LogGetIDByGroupIDAndName(db *gorm.DB, groupID string, logName string) (logID uint64, err error) {
	err = db.Table("logs").
		Select("id").
		Where("group_id = ? AND name = ?", groupID, logName).
		Scan(&logID).Error

	if err != nil {
		// 如果出现错误，判断是否没有找到对应的记录
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return logID, nil
}

// LogGetUploadInfo 获取上传信息
func LogGetUploadInfo(db *gorm.DB, groupID string, logName string) (url string, uploadTime, updateTime int64, err error) {
	var logInfo struct {
		UpdatedAt  int64  `gorm:"column:updated_at"`
		UploadURL  string `gorm:"column:upload_url"`
		UploadTime int64  `gorm:"column:upload_time"`
	}

	err = db.Table("logs").
		Select("updated_at, upload_url, upload_time").
		Where("group_id = ? AND name = ?", groupID, logName).
		Scan(&logInfo).Error

	if err != nil {
		return "", 0, 0, err
	}

	// 提取结果
	updateTime = logInfo.UpdatedAt
	url = logInfo.UploadURL
	uploadTime = logInfo.UploadTime

	return
}

// LogSetUploadInfo 设置上传信息
func LogSetUploadInfo(db *gorm.DB, groupID string, logName string, url string) error {
	if len(url) == 0 {
		return nil
	}

	now := time.Now().Unix()

	// 使用 GORM 更新上传信息
	err := db.Table("logs").Where("group_id = ? AND name = ?", groupID, logName).
		Update("upload_url", url).
		Update("upload_time", now).
		Error

	return err
}

// LogGetAllLines 获取log的所有行数据
func LogGetAllLines(db *gorm.DB, groupID string, logName string) ([]*LogOneItem, error) {
	// 获取log的ID
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return nil, err
	}

	var items []*LogOneItem

	// 查询行数据
	err = db.Table("log_items").
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID).
		Order("time ASC").
		Scan(&items).Error

	if err != nil {
		return nil, err
	}

	// 好像是不需要再反序列化成那个奇怪的interface{}了？

	return items, nil
}

type QueryLogLinePage struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	GroupID  string `query:"groupId"`
	LogName  string `query:"logName"`
}

// LogGetLinePage 获取log的行分页
func LogGetLinePage(db *gorm.DB, param *QueryLogLinePage) ([]*LogOneItem, error) {
	// 获取log的ID
	logID, err := LogGetIDByGroupIDAndName(db, param.GroupID, param.LogName)
	if err != nil {
		return nil, err
	}

	var items []*LogOneItem

	// 查询行数据
	err = db.Table("log_items").
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID).
		Order("time ASC").
		Limit(param.PageSize).
		Offset((param.PageNum - 1) * param.PageSize).
		Scan(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

// LogLinesCountGet 获取日志行数
func LogLinesCountGet(db *gorm.DB, groupID string, logName string) (int64, bool) {
	// 获取日志 ID
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil || logID == 0 {
		return 0, false
	}

	// 获取日志行数
	var count int64
	err = db.Table("log_items").
		//  AND removed IS NULL
		Where("log_id = ? and removed = 0", logID).
		Count(&count).Error

	if err != nil {
		return 0, false
	}

	return count, true
}

// LogDelete 删除log
func LogDelete(db *gorm.DB, groupID string, logName string) bool {
	// 获取 log ID
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil || logID == 0 {
		return false
	}

	// 开启事务
	tx := db.Begin()
	if err := tx.Error; err != nil {
		return false
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 删除 log_id 相关的 log_items 记录
	if err := tx.Where("log_id = ?", logID).Delete(&LogOneItem{}).Error; err != nil {
		return false
	}

	// 删除 log_id 相关的 logs 记录
	if err := tx.Where("id = ?", logID).Delete(&LogInfo{}).Error; err != nil {
		return false
	}

	// 提交事务
	err = tx.Commit().Error
	return err == nil
}

// LogAppend 向指定的log中添加一条信息
func LogAppend(db *gorm.DB, groupID string, logName string, logItem *LogOneItem) bool {
	// 获取 log ID
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return false
	}

	// 获取当前时间戳
	now := time.Now()
	nowTimestamp := now.Unix()

	// 开始事务
	tx := db.Begin()
	if err := tx.Error; err != nil {
		return false
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if logID == 0 {
		// 创建一个新的 log
		newLog := LogInfo{Name: logName, GroupID: groupID, CreatedAt: nowTimestamp, UpdatedAt: nowTimestamp}
		if err := tx.Create(&newLog).Error; err != nil {
			return false
		}
		logID = newLog.ID // 假设 LogInfo 结构体有 ID 字段
	}

	// 向 log_items 表中添加一条信息
	// Pinenutn: 由此可以推知，CommandInfo必然是一个 map[string]interface{}
	data, err := json.Marshal(logItem.CommandInfo)

	if err != nil {
		return false
	}

	newLogItem := LogOneItem{
		LogID:       logID,
		GroupID:     groupID,
		Nickname:    logItem.Nickname,
		IMUserID:    logItem.IMUserID,
		Time:        nowTimestamp,
		Message:     logItem.Message,
		IsDice:      logItem.IsDice,
		CommandID:   logItem.CommandID,
		CommandInfo: data,
		RawMsgID:    logItem.RawMsgID,
		UniformID:   logItem.UniformID,
	}

	if err := tx.Create(&newLogItem).Error; err != nil {
		return false
	}

	// 更新 logs 表中的 updated_at 字段
	if err := tx.Model(&LogInfo{}).Where("id = ?", logID).Update("updated_at", nowTimestamp).Error; err != nil {
		return false
	}

	// 提交事务
	err = tx.Commit().Error
	return err == nil
}

// LogMarkDeleteByMsgID 撤回删除
func LogMarkDeleteByMsgID(db *gorm.DB, groupID string, logName string, rawID interface{}) error {
	// 获取 log id
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return err
	}

	// 删除记录
	rid := ""
	if rawID != nil {
		rid = fmt.Sprintf("%v", rawID)
	}

	// fmt.Printf("log delete %v %d\n", rawId, logId)
	// TODO: 此处的代码是否有点问题？
	if err := db.Where("log_id = ? AND raw_msg_id = ?", logID, rid).Delete(&LogOneItem{}).Error; err != nil {
		fmt.Println("log delete error", err.Error())
		return err
	}

	return nil
}

// LogEditByMsgID 编辑日志
func LogEditByMsgID(db *gorm.DB, groupID, logName, newContent string, rawID interface{}) error {
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return err
	}

	rid := ""
	if rawID != nil {
		rid = fmt.Sprintf("%v", rawID)
	}

	// 更新 log_items 表中的内容
	if err := db.Model(&LogOneItem{}).
		Where("log_id = ? AND raw_msg_id = ?", logID, rid).
		Update("message", newContent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("log edit: %w", err)
	}

	return nil
}
