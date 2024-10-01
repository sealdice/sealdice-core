package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"sealdice-core/dice/censor"
)

type CensorLog struct {
	ID           uint64 `json:"id" gorm:"primarykey;autoIncrement;column:id"`
	MsgType      string `json:"msgType" gorm:"column:msg_type"`
	UserID       string `json:"userId" gorm:"index:idx_censor_log_user_id;column:user_id"`
	GroupID      string `json:"groupId" gorm:"column:group_id"`
	Content      string `json:"content" gorm:"column:content"`
	HighestLevel int    `json:"highestLevel" gorm:"index:idx_censor_log_level;column:highest_level"`
	CreatedAt    int    `json:"createdAt" gorm:"column:created_at"`
	// 补充gorm有的部分：
	SensitiveWords string `json:"-" gorm:"column:sensitive_words"`
	ClearMark      bool   `json:"-" gorm:"column:clear_mark"`
}

func CensorAppend(db *gorm.DB, msgType string, userID string, groupID string, content string, sensitiveWords interface{}, highestLevel int) bool {
	// 获取当前时间的 Unix 时间戳
	nowTimestamp := time.Now().Unix()

	// 将敏感词转换为 JSON 字符串
	words, err := json.Marshal(sensitiveWords)
	if err != nil {
		return false
	}

	// 创建 CensorLog 实例，手动设置 CreatedAt
	censorLog := CensorLog{
		MsgType:        msgType,
		UserID:         userID,
		GroupID:        groupID,
		Content:        content,
		SensitiveWords: string(words),
		HighestLevel:   highestLevel,
		CreatedAt:      int(nowTimestamp), // Unix 时间戳
		ClearMark:      false,
	}
	// 使用 GORM 的 Create 方法插入记录
	if err := db.Create(&censorLog).Error; err != nil {
		return false
	}
	return true
}

func CensorCount(db *gorm.DB, userID string) map[censor.Level]int {
	// 定义要查询的不同敏感级别
	levels := [5]censor.Level{censor.Ignore, censor.Notice, censor.Caution, censor.Warning, censor.Danger}
	var temp int64
	res := make(map[censor.Level]int)

	// 遍历每个敏感级别并执行查询
	for _, level := range levels {
		// 使用 GORM 的链式查询
		err := db.Model(&CensorLog{}).Where("user_id = ? AND highest_level = ? AND clear_mark = ?", userID, level, false).
			Count(&temp).Error

		// 如果查询出现错误，忽略并赋值为 0
		if err != nil {
			res[level] = 0
		} else {
			res[level] = int(temp)
		}
	}

	return res
}

func CensorClearLevelCount(db *gorm.DB, userID string, level censor.Level) {
	// 使用 GORM 的链式查询执行批量更新
	err := db.Model(&CensorLog{}).
		Where("user_id = ? AND highest_level = ?", userID, level).
		Update("clear_mark", true).Error
	if err != nil {
		fmt.Println(err)
	}
}

// QueryCensorLog 是分页查询的参数
type QueryCensorLog struct {
	PageNum  int    `query:"pageNum"`  // 当前页码
	PageSize int    `query:"pageSize"` // 每页条数
	UserID   string `query:"userId"`   // 用户ID
	Level    int    `query:"level"`    // 敏感级别
}

// CensorGetLogPage 使用 GORM 进行分页查询
func CensorGetLogPage(db *gorm.DB, params QueryCensorLog) (int64, []CensorLog, error) {
	var total int64
	var logs []CensorLog

	// 首先统计总记录数
	query := db.Model(&CensorLog{})

	// 如果传入了 UserID 和 Level，则添加查询条件
	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Level != 0 {
		query = query.Where("highest_level = ?", params.Level)
	}

	// 统计符合条件的总记录数
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 查询分页数据
	if err := query.
		Order("created_at DESC").                       // 按照创建时间倒序排列
		Limit(params.PageSize).                         // 限制返回条数
		Offset((params.PageNum - 1) * params.PageSize). // 偏移
		Find(&logs).                                    // 查询数据
		Error; err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}
