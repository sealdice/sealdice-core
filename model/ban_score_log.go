package model

// BanScoreLog 怒气值变更日志模型
// GORM STRUCT
type BanScoreLog struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement;column:id"                  json:"id"`
	UserID        string `gorm:"index:idx_ban_score_log_user_id;column:user_id"      json:"userId"`        // 用户/群组ID
	UserName      string `gorm:"column:user_name"                                    json:"userName"`      // 用户/群组名称
	GroupID       string `gorm:"column:group_id"                                     json:"groupId"`       // 事发群组ID
	Score         int64  `gorm:"column:score"                                        json:"score"`         // 变更的分数(正为增加，负为减少)
	ScoreAfter    int64  `gorm:"column:score_after"                                  json:"scoreAfter"`    // 变更后的总分
	Reason        string `gorm:"column:reason"                                       json:"reason"`        // 变更原因
	RankBefore    int    `gorm:"column:rank_before"                                  json:"rankBefore"`    // 变更前等级
	RankAfter     int    `gorm:"column:rank_after"                                   json:"rankAfter"`     // 变更后等级
	ChangeType    string `gorm:"index:idx_ban_score_log_change_type;column:change_type" json:"changeType"` // 变更类型: censor/muted/kicked/spam/manual/joint
	CensorWords   string `gorm:"column:censor_words"                                 json:"censorWords"`   // 触发的违禁词(JSON数组)
	CensorLevel   string `gorm:"column:censor_level"                                 json:"censorLevel"`   // 违禁词等级
	IsWarning     bool   `gorm:"column:is_warning"                                   json:"isWarning"`     // 是否触发警告
	IsBanned      bool   `gorm:"column:is_banned"                                    json:"isBanned"`      // 是否触发黑名单
	CreatedAt     int64  `gorm:"index:idx_ban_score_log_created_at;column:created_at" json:"createdAt"`    // 创建时间
}

func (BanScoreLog) TableName() string {
	return "ban_score_log"
}
