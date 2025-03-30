package model

type CensorLog struct {
	ID           uint64 `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	MsgType      string `json:"msgType" gorm:"column:msg_type"`
	UserID       string `json:"userId" gorm:"index:idx_censor_log_user_id;column:user_id"`
	GroupID      string `json:"groupId" gorm:"column:group_id"`
	Content      string `json:"content" gorm:"column:content"`
	HighestLevel int    `json:"highestLevel" gorm:"index:idx_censor_log_level;column:highest_level"`
	CreatedAt    int    `json:"createdAt" gorm:"column:created_at"`
	// 补充gorm有的部分：
	SensitiveWords string `json:"-" gorm:"column:sensitive_words"`
	ClearMark      bool   `json:"-" gorm:"column:clear_mark;type:bool"`
}

func (CensorLog) TableName() string {
	return "censor_log"
}
