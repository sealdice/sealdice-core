package model

// 仅修改为gorm格式
type EndpointInfo struct {
	UserID      string `gorm:"column:user_id;primaryKey"`
	CmdNum      int64  `gorm:"column:cmd_num;"`
	CmdLastTime int64  `gorm:"column:cmd_last_time;"`
	OnlineTime  int64  `gorm:"column:online_time;"`
	UpdatedAt   int64  `gorm:"column:updated_at;"`
}

func (EndpointInfo) TableName() string {
	return "endpoint_info"
}
