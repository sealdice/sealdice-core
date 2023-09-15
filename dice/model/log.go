package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"time"
)

type LogOne struct {
	// Version int           `json:"version,omitempty"`
	Items []LogOneItem `json:"items"`
	Info  LogInfo      `json:"info"`
}

type LogOneItem struct {
	Id          uint64      `json:"id" db:"id"`
	Nickname    string      `json:"nickname" db:"nickname"`
	IMUserId    string      `json:"IMUserId" db:"im_userid"`
	Time        int64       `json:"time" db:"time"`
	Message     string      `json:"message" db:"message"`
	IsDice      bool        `json:"isDice" db:"is_dice"`
	CommandId   int64       `json:"commandId" db:"command_id"`
	CommandInfo interface{} `json:"commandInfo" db:"command_info"`
	RawMsgId    interface{} `json:"rawMsgId" db:"raw_msg_id"`

	UniformId string `json:"uniformId" db:"user_uniform_id"`
	Channel   string `json:"channel"`
}

type LogInfo struct {
	Id        uint64 `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	GroupId   string `json:"groupId" db:"groupId"`
	CreatedAt int64  `json:"createdAt" db:"created_at"`
	UpdatedAt int64  `json:"updatedAt" db:"updated_at"`
}

func LogGetInfo(db *sqlx.DB) ([]int, error) {
	lst := []int{0, 0}
	err := db.Get(&lst[0], "SELECT seq FROM sqlite_sequence WHERE name == 'logs'")
	if err != nil {
		return nil, err
	}
	err = db.Get(&lst[1], "SELECT seq FROM sqlite_sequence WHERE name == 'log_items'")
	if err != nil {
		return nil, err
	}
	return lst, nil
}

func LogGetLogs(db *sqlx.DB) ([]*LogInfo, error) {
	var lst []*LogInfo
	rows, err := db.Queryx("SELECT id,name,group_id,created_at, updated_at FROM logs")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		log := &LogInfo{}
		if err := rows.Scan(
			&log.Id,
			&log.Name,
			&log.GroupId,
			&log.CreatedAt,
			&log.UpdatedAt,
		); err != nil {
			return nil, err
		}
		lst = append(lst, log)
	}
	return lst, nil
}

// LogGetList 获取列表
func LogGetList(db *sqlx.DB, groupId string) ([]string, error) {
	var lst []string
	err := db.Select(&lst, "SELECT name FROM logs WHERE group_id = $1 ORDER BY updated_at DESC", groupId)
	if err != nil {
		return nil, err
	}
	return lst, nil
}

// LogGetIdByGroupIdAndName 获取ID
func LogGetIdByGroupIdAndName(db *sqlx.DB, groupId string, logName string) (logId int64, err error) {
	err = db.Get(&logId, "SELECT id FROM logs WHERE group_id = $1 AND name = $2", groupId, logName)
	if err != nil {
		// 如果出现错误，判断是否没有找到对应的记录
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return logId, nil
}

// LogGetAllLines 获取log的所有行数据
func LogGetAllLines(db *sqlx.DB, groupId string, logName string) ([]*LogOneItem, error) {
	// 获取log的ID
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil {
		return nil, err
	}

	// 查询行数据
	rows, err := db.Queryx(`SELECT id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id 
	                        FROM log_items WHERE log_id=$1 ORDER BY time ASC`, logId)
	if err != nil {
		return nil, err
	}
	defer func(rows *sqlx.Rows) {
		_ = rows.Close()
	}(rows)

	var ret []*LogOneItem
	for rows.Next() {
		item := &LogOneItem{}
		var commandInfoStr []byte

		// 使用Scan方法将查询结果映射到结构体中
		if err := rows.Scan(
			&item.Id,
			&item.Nickname,
			&item.IMUserId,
			&item.Time,
			&item.Message,
			&item.IsDice,
			&item.CommandId,
			&commandInfoStr,
			&item.RawMsgId,
			&item.UniformId,
		); err != nil {
			return nil, err
		}

		// 反序列化commandInfo
		if commandInfoStr != nil {
			_ = json.Unmarshal(commandInfoStr, &item.CommandInfo)
		}

		ret = append(ret, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

// LogLinesCountGet 获取日志行数
func LogLinesCountGet(db *sqlx.DB, groupId string, logName string) (int64, bool) {
	// 获取日志 ID
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil || logId == 0 {
		return 0, false
	}

	// 获取日志行数
	var count int64
	err = db.Get(&count, `
		SELECT COUNT(id) FROM log_items WHERE log_id=$1 AND removed IS NULL
	`, logId)
	if err != nil {
		return 0, false
	}

	return count, true
}

// LogDelete 删除log
func LogDelete(db *sqlx.DB, groupId string, logName string) bool {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil || logId == 0 {
		return false
	}

	// 获取文本
	// 通过BeginTxx方法开启事务
	tx, err := db.Beginx()
	if err != nil {
		return false
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 删除log_id相关的log_items记录
	_, err = tx.Exec("DELETE FROM log_items WHERE log_id = $1", logId)
	if err != nil {
		return false
	}

	// 删除log_id相关的logs记录
	_, err = tx.Exec("DELETE FROM logs WHERE id = $1", logId)
	if err != nil {
		return false
	}

	// 提交事务
	err = tx.Commit()
	return err == nil
}

// LogAppend 向指定的log中添加一条信息
func LogAppend(db *sqlx.DB, groupId string, logName string, logItem *LogOneItem) bool {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil {
		return false
	}

	// 如果不存在，创建
	now := time.Now()
	nowTimestamp := now.Unix()

	// 开始事务
	tx, err := db.Beginx()
	if err != nil {
		return false
	}
	// 执行事务时发生错误时回滚
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if logId == 0 {
		// 创建一个新的log
		query := "INSERT INTO logs (name, group_id, created_at, updated_at) VALUES (?, ?, ?, ?)"
		rst, err := tx.Exec(query, logName, groupId, nowTimestamp, nowTimestamp)
		if err != nil {
			return false
		}
		// 获取新创建log的ID
		logId, err = rst.LastInsertId()
		if err != nil {
			return false
		}
	}

	// 向log_items表中添加一条信息
	data, err := json.Marshal(logItem.CommandInfo)
	query := "INSERT INTO log_items (log_id, group_id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	rid := ""
	if logItem.RawMsgId != nil {
		rid = fmt.Sprintf("%v", logItem.RawMsgId)
	}

	//fmt.Println("log append", logId, rid, "|", groupId, logName)
	_, err = tx.Exec(query, logId, groupId, logItem.Nickname, logItem.IMUserId, nowTimestamp, logItem.Message, logItem.IsDice, logItem.CommandId, data, rid, logItem.UniformId)
	_, err = tx.Exec("UPDATE logs SET updated_at = ? WHERE id = ?", nowTimestamp, logId)
	if err != nil {
		return false
	}

	// 提交事务
	err = tx.Commit()
	return err == nil
}

// LogMarkDeleteByMsgId 撤回删除
func LogMarkDeleteByMsgId(db *sqlx.DB, groupId string, logName string, rawId interface{}) error {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil {
		return err
	}

	// 删除记录
	rid := ""
	if rawId != nil {
		rid = fmt.Sprintf("%v", rawId)
	}

	//fmt.Printf("log delete %v %d\n", rawId, logId)
	_, err = db.Exec("DELETE FROM log_items WHERE log_id=? AND raw_msg_id=?", logId, rid)
	if err != nil {
		fmt.Println("log delete error", err.Error())
		return err
	}

	return nil
}
