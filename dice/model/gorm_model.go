package model

// LOG
// Logs 模型

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
