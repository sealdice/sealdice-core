package model

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

type CensorLog struct {
	Id             uint64
	UserId         string
	Content        string
	SensitiveWords interface{}
	HighestLevel   int
}

func CensorAppend(db *sqlx.DB, msgType string, userId string, groupId string, content string, sensitiveWords interface{}, highestLevel int) bool {
	now := time.Now()
	nowTimestamp := now.Unix()

	words, err := json.Marshal(sensitiveWords)
	_, err = db.Exec(`
INSERT INTO censor_log(
    msg_type,
    user_id,
    group_id, 
    content,
    sensitive_words,
    highest_level,
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msgType, userId, groupId, content, words, highestLevel, nowTimestamp)

	if err != nil {
		return false
	}
	return err == nil
}
