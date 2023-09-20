package model

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

type CensorLog struct {
	Id           uint64 `json:"id"`
	MsgType      string `json:"msgType"`
	UserId       string `json:"userId"`
	GroupId      string `json:"groupId"`
	Content      string `json:"content"`
	HighestLevel int    `json:"highestLevel"`
	CreatedAt    int    `json:"createdAt"`
}

func CensorAppend(db *sqlx.DB, msgType string, userId string, groupId string, content string, sensitiveWords interface{}, highestLevel int) bool {
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
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msgType, userId, groupId, content, words, highestLevel, nowTimestamp)

	if err != nil {
		return false
	}
	return err == nil
}

type QueryCensorLog struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	UserId   string `query:"userId"`
	Level    int    `query:"level"`
}

func CensorGetLogPage(db *sqlx.DB, query *QueryCensorLog) ([]*CensorLog, error) {
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
			&log.Id,
			&log.MsgType,
			&log.UserId,
			&log.GroupId,
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
