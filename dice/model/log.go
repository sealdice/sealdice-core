package model

import (
	"encoding/json"
	"time"
	"zombiezen.com/go/sqlite/sqlitex"
)

type LogOneItem struct {
	Id          uint64      `json:"id"`
	Nickname    string      `json:"nickname"`
	IMUserId    string      `json:"IMUserId"`
	Time        int64       `json:"time"`
	Message     string      `json:"message"`
	IsDice      bool        `json:"isDice"`
	CommandId   int64       `json:"commandId"`
	CommandInfo interface{} `json:"commandInfo"`
	RawMsgId    interface{} `json:"rawMsgId"`

	UniformId string `json:"uniformId"`
	Channel   string `json:"channel"` // 用于秘密团
}

// LogGetList 获取列表
func LogGetList(db *sqlitex.Pool, groupId string) ([]string, error) {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	lst := []string{}
	stmt := conn.Prep(`select name from logs where group_id=$group_id order by updated_at desc`)
	stmt.SetText("$group_id", groupId)
	defer stmt.Finalize()

	var err error
	var hasRow bool

	for {
		if hasRow, err = stmt.Step(); err != nil {
			break // error
		} else if !hasRow {
			break
		}
		lst = append(lst, stmt.ColumnText(0))
	}

	return lst, err
}

// LogGetIdByGroupIdAndName 获取ID
func LogGetIdByGroupIdAndName(db *sqlitex.Pool, groupId string, logName string) (logId int64, err error) {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	stmt := conn.Prep(`select id from logs where group_id=$group_id and name=$name`)
	stmt.SetText("$group_id", groupId)
	stmt.SetText("$name", logName)
	defer stmt.Finalize()

	var hasRow bool

	for {
		// 加for的原因是 panic: connection returned to pool has active statement
		if hasRow, err = stmt.Step(); err != nil {
			break
		} else if !hasRow {
			break
		}

		logId = stmt.ColumnInt64(0)
	}

	return logId, nil
}

// LogGetAllLines 获取log内容
func LogGetAllLines(db *sqlitex.Pool, groupId string, logName string) ([]*LogOneItem, error) {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	ret := []*LogOneItem{}
	if err != nil {
		return ret, err
	}

	// 获取文本
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	// 取得了id，获取列表
	stmt := conn.Prep(`
		select id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id, removed, parent_id
		from log_items where log_id=$log_id order by time asc`)
	stmt.SetInt64("$log_id", logId)
	defer stmt.Finalize()

	var hasRow bool
	for {
		if hasRow, err = stmt.Step(); err != nil {
			// ... handle error
			break
		} else if !hasRow {
			break
		}

		// 10,11 removed, parent_id
		if stmt.ColumnInt64(10) == 1 {
			continue
		}

		// 反序列化 commandInfo
		buf := []byte(stmt.ColumnText(7))
		commandInfo := map[string]interface{}{}
		_ = json.Unmarshal(buf, &commandInfo)

		item := LogOneItem{
			Id:          uint64(stmt.ColumnInt64(0)),
			Nickname:    stmt.ColumnText(1),
			IMUserId:    stmt.ColumnText(2),
			Time:        stmt.ColumnInt64(3),
			Message:     stmt.ColumnText(4),
			IsDice:      stmt.ColumnInt64(5) == 1,
			CommandId:   stmt.ColumnInt64(6),
			CommandInfo: commandInfo,
			RawMsgId:    stmt.ColumnText(8),

			UniformId: stmt.ColumnText(9),
		}

		ret = append(ret, &item)
	}

	return ret, err
}

// LogLinesCountGet 获取行数
func LogLinesCountGet(db *sqlitex.Pool, groupId string, logName string) (int64, bool) {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil || logId == 0 {
		return 0, false
	}

	// 获取文本
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	// 取得了id，获取列表
	// 注: 这样查询和二重查询花的时间是一样的，removed没有index
	stmt := conn.Prep(`select count(id) from log_items where log_id=$log_id and removed is null`)
	stmt.SetInt64("$log_id", logId)
	defer stmt.Finalize()

	var hasRow bool
	var count int64

	for {
		// 加for的原因是 panic: connection returned to pool has active statement
		if hasRow, err = stmt.Step(); err != nil {
			break
		} else if !hasRow {
			break
		}
		count = stmt.ColumnInt64(0)
	}

	return count, true
}

// LogDelete 删除log
func LogDelete(db *sqlitex.Pool, groupId string, logName string) bool {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil || logId == 0 {
		return false
	}

	// 获取文本
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	// 取得了id，获取列表
	stmt := conn.Prep(`delete from log_items where log_id=$log_id`)
	defer stmt.Finalize()
	stmt.SetInt64("$log_id", logId)
	for {
		if hasRow, err := stmt.Step(); err != nil {
			break // error
		} else if !hasRow {
			break
		}
	}

	stmt2 := conn.Prep(`delete from logs where id=$log_id`)
	defer stmt2.Finalize()
	stmt2.SetInt64("$log_id", logId)
	for {
		if hasRow, err := stmt2.Step(); err != nil {
			break // error
		} else if !hasRow {
			break
		}
	}
	return true
}

// LogAppend 添加消息
func LogAppend(db *sqlitex.Pool, groupId string, logName string, logItem *LogOneItem) bool {
	// 获取 log id
	logId, err := LogGetIdByGroupIdAndName(db, groupId, logName)
	if err != nil {
		return false
	}

	// 如果不存在，创建
	now := time.Now()
	nowTimestamp := now.Unix()

	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	if logId == 0 {
		// 创建
		stmt := conn.Prep(`insert into logs (name, group_id, created_at, updated_at) VALUES ($name, $group_id, $created_at, $updated_at)`)
		defer stmt.Finalize()
		stmt.SetText("$name", logName)
		stmt.SetText("$group_id", groupId)
		stmt.SetInt64("$created_at", nowTimestamp)
		stmt.SetInt64("$updated_at", nowTimestamp)
		for {
			if hasRow, err := stmt.Step(); err != nil {
				break // error
			} else if !hasRow {
				break
			}
		}
		logId = conn.LastInsertRowID()
	}

	// 添加一条信息
	stmt2 := conn.Prep(`insert into log_items (log_id, group_id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id) VALUES ($log_id, $group_id, $nickname, $im_userid, $time, $message, $is_dice, $command_id, $command_info, $raw_msg_id, $user_uniform_id)`)
	defer stmt2.Finalize()
	stmt2.SetInt64("$log_id", logId)
	stmt2.SetText("$group_id", groupId)
	stmt2.SetText("$nickname", logItem.Nickname)
	stmt2.SetText("$im_userid", logItem.IMUserId)
	stmt2.SetInt64("$time", nowTimestamp)
	stmt2.SetText("$message", logItem.Message)
	stmt2.SetBool("$is_dice", logItem.IsDice)
	stmt2.SetInt64("$command_id", int64(logItem.CommandId))
	d, _ := json.Marshal(logItem.CommandInfo)
	stmt2.SetBytes("$command_info", d)
	d2, _ := json.Marshal(logItem.RawMsgId)
	stmt2.SetBytes("$raw_msg_id", d2)
	stmt2.SetText("$user_uniform_id", logItem.UniformId)
	for {
		if hasRow, err := stmt2.Step(); err != nil {
			break // error
		} else if !hasRow {
			break
		}
	}
	return true
}

// LogMarkDeleteByMsgId 撤回删除
func LogMarkDeleteByMsgId(db *sqlitex.Pool, groupId string, logName string, rawId interface{}) error {
	return nil
}
