package model

import (
	"encoding/json"
	"sealdice-core/dice/censor"
	"time"

	"github.com/jmoiron/sqlx"
)

type CensorLog struct {
	ID           uint64 `json:"id"`
	MsgType      string `json:"msgType"`
	UserID       string `json:"userId"`
	GroupID      string `json:"groupId"`
	Content      string `json:"content"`
	HighestLevel int    `json:"highestLevel"`
	CreatedAt    int    `json:"createdAt"`
}

func CensorAppend(db *sqlx.DB, msgType string, userID string, groupID string, content string, sensitiveWords interface{}, highestLevel int) bool {
	now := time.Now()
	nowTimestamp := now.Unix()

	words, err := json.Marshal(sensitiveWords)
	if err != nil {
		return false
	}

	_, err = db.Exec(`
INSERT INTO censor_log(
    msg_type,
    user_id,
    group_id,
    content,
    sensitive_words,
    highest_level,
    created_at,
    clear_mark
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msgType, userID, groupID, content, words, highestLevel, nowTimestamp, false)

	if err != nil {
		return false
	}
	return err == nil
}

func CensorCount(db *sqlx.DB, userID string) map[censor.Level]int {
	levels := [5]censor.Level{censor.Ignore, censor.Notice, censor.Caution, censor.Warning, censor.Danger}
	var temp int
	res := make(map[censor.Level]int)
	for _, level := range levels {
		_ = db.Get(&temp, `SELECT COUNT(*) FROM censor_log WHERE user_id = ? AND highest_level = ? AND clear_mark = ?`, userID, level, false)
		res[level] = temp
	}
	return res
}

func CensorClearLevelCount(db *sqlx.DB, userID string, level censor.Level) {
	_, _ = db.Exec(`UPDATE censor_log SET clear_mark = ? WHERE user_id = ? AND highest_level = ?`, true, userID, level)
}

type QueryCensorLog struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	UserID   string `query:"userId"`
	Level    int    `query:"level"`
}

func CensorGetLogPage(db *sqlx.DB, _ *QueryCensorLog) ([]*CensorLog, error) {
	var res []*CensorLog
	rows, err := db.Queryx(`
SELECT id,
       msg_type,
       user_id,
       group_id,
       content,
       highest_level,
       created_at
FROM censor_log`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sqlx.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		log := &CensorLog{}
		err := rows.Scan(
			&log.ID,
			&log.MsgType,
			&log.UserID,
			&log.GroupID,
			&log.Content,
			&log.HighestLevel,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, log)
	}

	return res, nil
}
