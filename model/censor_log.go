package model

type CensorLog struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement;column:id"              json:"id"`
	MsgType      string `gorm:"column:msg_type"                                 json:"msgType"`
	UserID       string `gorm:"index:idx_censor_log_user_id;column:user_id"     json:"userId"`
	GroupID      string `gorm:"column:group_id"                                 json:"groupId"`
	Content      string `gorm:"column:content"                                  json:"content"`
	HighestLevel int    `gorm:"index:idx_censor_log_level;column:highest_level" json:"highestLevel"`
	CreatedAt    int    `gorm:"column:created_at"                               json:"createdAt"`
	// 补充gorm有的部分：
	SensitiveWords string `gorm:"column:sensitive_words"      json:"-"`
	ClearMark      bool   `gorm:"column:clear_mark;type:bool" json:"-"`
}

func (CensorLog) TableName() string {
	return "censor_log"
}
