package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type LogOne struct {
	// Version int           `json:"version,omitempty"`
	Items []LogOneItem `json:"items"`
	Info  LogInfo      `json:"info"`
}

type LogOneItem struct {
	ID          uint64      `json:"id" db:"id"`
	Nickname    string      `json:"nickname" db:"nickname"`
	IMUserID    string      `json:"IMUserId" db:"im_userid"`
	Time        int64       `json:"time" db:"time"`
	Message     string      `json:"message" db:"message"`
	IsDice      bool        `json:"isDice" db:"is_dice"`
	CommandID   int64       `json:"commandId" db:"command_id"`
	CommandInfo interface{} `json:"commandInfo" db:"command_info"`
	RawMsgID    interface{} `json:"rawMsgId" db:"raw_msg_id"`

	UniformID string `json:"uniformId" db:"user_uniform_id"`
	Channel   string `json:"channel"`
}

type LogInfo struct {
	ID        uint64 `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	GroupID   string `json:"groupId" db:"groupId"`
	CreatedAt int64  `json:"createdAt" db:"created_at"`
	UpdatedAt int64  `json:"updatedAt" db:"updated_at"`
	Size      int    `json:"size" db:"size"`
}

func LogGetInfo(db *sqlx.DB) ([]int, error) {
	lst := []int{0, 0, 0, 0}
	err := db.Get(&lst[0], "SELECT seq FROM sqlite_sequence WHERE name == 'logs'")
	if err != nil {
		return nil, err
	}
	err = db.Get(&lst[1], "SELECT seq FROM sqlite_sequence WHERE name == 'log_items'")
	if err != nil {
		return nil, err
	}
	err = db.Get(&lst[2], "SELECT COUNT(*) FROM logs")
	if err != nil {
		return nil, err
	}
	err = db.Get(&lst[3], "SELECT COUNT(*) FROM log_items")
	if err != nil {
		return nil, err
	}
	return lst, nil
}

// Deprecated: replaced by page
func LogGetLogs(db *sqlx.DB) ([]*LogInfo, error) {
	var lst []*LogInfo
	rows, err := db.Queryx("SELECT id,name,group_id,created_at, updated_at FROM logs")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		log := &LogInfo{}
		if err := rows.Scan(
			&log.ID,
			&log.Name,
			&log.GroupID,
			&log.CreatedAt,
			&log.UpdatedAt,
		); err != nil {
			return nil, err
		}
		lst = append(lst, log)
	}
	return lst, nil
}

type QueryLogPage struct {
	PageNum          int    `db:"page_num" query:"pageNum"`
	PageSize         int    `db:"page_siz" query:"pageSize"`
	Name             string `db:"name" query:"name"`
	GroupID          string `db:"group_id" query:"groupId"`
	CreatedTimeBegin string `db:"created_time_begin" query:"createdTimeBegin"`
	CreatedTimeEnd   string `db:"created_time_end" query:"createdTimeEnd"`
}

// LogGetLogPage 获取分页
func LogGetLogPage(db *sqlx.DB, param *QueryLogPage) ([]*LogInfo, error) {
	query := `
SELECT logs.id         as id,
       logs.name       as name,
       logs.group_id   as group_id,
       logs.created_at as created_at,
       logs.updated_at as updated_at,
       count(logs.id)  as size
FROM logs
         LEFT JOIN log_items items ON logs.id = items.log_id
`
	var conditions []string
	if param.Name != "" {
		conditions = append(conditions, "logs.name like '%' || :name || '%'")
	}
	if param.GroupID != "" {
		conditions = append(conditions, "logs.group_id like '%' || :group_id || '%'")
	}
	if param.CreatedTimeBegin != "" {
		conditions = append(conditions, "logs.created_at >= :created_time_begin")
	}
	if param.CreatedTimeEnd != "" {
		conditions = append(conditions, "logs.created_at <= :created_time_end")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" GROUP BY logs.id LIMIT %d, %d", (param.PageNum-1)*param.PageSize, param.PageSize)

	var lst []*LogInfo
	rows, err := db.NamedQuery(query, param)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		log := &LogInfo{}
		if err := rows.Scan(
			&log.ID,
			&log.Name,
			&log.GroupID,
			&log.CreatedAt,
			&log.UpdatedAt,
			&log.Size,
		); err != nil {
			return nil, err
		}
		lst = append(lst, log)
	}
	return lst, nil
}

// LogGetList 获取列表
func LogGetList(db *sqlx.DB, groupID string) ([]string, error) {
	var lst []string
	err := db.Select(&lst, "SELECT name FROM logs WHERE group_id = $1 ORDER BY updated_at DESC", groupID)
	if err != nil {
		return nil, err
	}
	return lst, nil
}

// LogGetIDByGroupIDAndName 获取ID
func LogGetIDByGroupIDAndName(db *sqlx.DB, groupID string, logName string) (logID int64, err error) {
	err = db.Get(&logID, "SELECT id FROM logs WHERE group_id = $1 AND name = $2", groupID, logName)
	if err != nil {
		// 如果出现错误，判断是否没有找到对应的记录
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return logID, nil
}

func LogGetUploadInfo(db *sqlx.DB, groupID string, logName string) (url string, uploadTime, updateTime int64, err error) {
	res, err := db.Queryx(
		`SELECT updated_at, upload_url, upload_time FROM logs WHERE group_id = $1 AND name = $2`,
		groupID, logName,
	)
	if err != nil {
		return "", 0, 0, err
	}
	defer func() { _ = res.Close() }()

	for res.Next() {
		err = res.Scan(&updateTime, &url, &uploadTime)
		if err != nil {
			return "", 0, 0, err
		}
	}

	return
}

func LogSetUploadInfo(db *sqlx.DB, groupID string, logName string, url string) error {
	if len(url) == 0 {
		return nil
	}

	now := time.Now().Unix()

	_, err := db.Exec(
		`UPDATE logs SET upload_url = $1, upload_time = $2 WHERE group_id = $3 AND name = $4`,
		url, now, groupID, logName,
	)
	return err
}

// LogGetAllLines 获取log的所有行数据
func LogGetAllLines(db *sqlx.DB, groupID string, logName string) ([]*LogOneItem, error) {
	// 获取log的ID
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return nil, err
	}

	// 查询行数据
	rows, err := db.Queryx(`SELECT id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id
	                        FROM log_items WHERE log_id=$1 ORDER BY time ASC`, logID)
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
			&item.ID,
			&item.Nickname,
			&item.IMUserID,
			&item.Time,
			&item.Message,
			&item.IsDice,
			&item.CommandID,
			&commandInfoStr,
			&item.RawMsgID,
			&item.UniformID,
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

type QueryLogLinePage struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	GroupID  string `query:"groupId"`
	LogName  string `query:"logName"`
}

// LogGetLinePage 获取log的行分页
func LogGetLinePage(db *sqlx.DB, param *QueryLogLinePage) ([]*LogOneItem, error) {
	// 获取log的ID
	logID, err := LogGetIDByGroupIDAndName(db, param.GroupID, param.LogName)
	if err != nil {
		return nil, err
	}

	// 查询行数据
	rows, err := db.Queryx(`
SELECT id,
       nickname,
       im_userid,
       time,
       message,
       is_dice,
       command_id,
       command_info,
       raw_msg_id,
       user_uniform_id
FROM log_items
WHERE log_id =$1
ORDER BY time ASC
LIMIT $2, $3;`, logID, (param.PageNum-1)*param.PageSize, param.PageSize)

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
			&item.ID,
			&item.Nickname,
			&item.IMUserID,
			&item.Time,
			&item.Message,
			&item.IsDice,
			&item.CommandID,
			&commandInfoStr,
			&item.RawMsgID,
			&item.UniformID,
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
func LogLinesCountGet(db *sqlx.DB, groupID string, logName string) (int64, bool) {
	// 获取日志 ID
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil || logID == 0 {
		return 0, false
	}

	// 获取日志行数
	var count int64
	err = db.Get(&count, `
		SELECT COUNT(id) FROM log_items WHERE log_id=$1 AND removed IS NULL
	`, logID)
	if err != nil {
		return 0, false
	}

	return count, true
}

// LogDelete 删除log
func LogDelete(db *sqlx.DB, groupID string, logName string) bool {
	// 获取 log id
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil || logID == 0 {
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
	_, err = tx.Exec("DELETE FROM log_items WHERE log_id = $1", logID)
	if err != nil {
		return false
	}

	// 删除log_id相关的logs记录
	_, err = tx.Exec("DELETE FROM logs WHERE id = $1", logID)
	if err != nil {
		return false
	}

	// 提交事务
	err = tx.Commit()
	return err == nil
}

// LogAppend 向指定的log中添加一条信息
func LogAppend(db *sqlx.DB, groupID string, logName string, logItem *LogOneItem) bool {
	// 获取 log id
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
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

	if logID == 0 {
		// 创建一个新的log
		query := "INSERT INTO logs (name, group_id, created_at, updated_at) VALUES (?, ?, ?, ?)"
		rst, errNew := tx.Exec(query, logName, groupID, nowTimestamp, nowTimestamp)
		if errNew != nil {
			return false
		}
		// 获取新创建log的ID
		logID, errNew = rst.LastInsertId()
		if errNew != nil {
			return false
		}
	}

	// 向log_items表中添加一条信息
	data, err := json.Marshal(logItem.CommandInfo)
	query := "INSERT INTO log_items (log_id, group_id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	rid := ""
	if logItem.RawMsgID != nil {
		rid = fmt.Sprintf("%v", logItem.RawMsgID)
	}

	// fmt.Println("log append", logId, rid, "|", groupId, logName)
	_, err = tx.Exec(query, logID, groupID, logItem.Nickname, logItem.IMUserID, nowTimestamp, logItem.Message, logItem.IsDice, logItem.CommandID, data, rid, logItem.UniformID)
	_, err = tx.Exec("UPDATE logs SET updated_at = ? WHERE id = ?", nowTimestamp, logID)
	if err != nil {
		return false
	}

	// 提交事务
	err = tx.Commit()
	return err == nil
}

// LogMarkDeleteByMsgID 撤回删除
func LogMarkDeleteByMsgID(db *sqlx.DB, groupID string, logName string, rawID interface{}) error {
	// 获取 log id
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return err
	}

	// 删除记录
	rid := ""
	if rawID != nil {
		rid = fmt.Sprintf("%v", rawID)
	}

	// fmt.Printf("log delete %v %d\n", rawId, logId)
	_, err = db.Exec("DELETE FROM log_items WHERE log_id=? AND raw_msg_id=?", logID, rid)
	if err != nil {
		fmt.Println("log delete error", err.Error())
		return err
	}

	return nil
}

func LogEditByMsgID(db *sqlx.DB, groupID, logName, newContent string, rawID interface{}) error {
	logID, err := LogGetIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return err
	}

	rid := ""
	if rawID != nil {
		rid = fmt.Sprintf("%v", rawID)
	}

	_, err = db.Exec("SELECT * FROM log_items WHERE log_id = ? AND raw_msg_id = ?", logID, rid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 开启 log 后修改开启前的消息可能很常见，这边太吵了
			// fmt.Printf("\n")
			return nil
		}
		return fmt.Errorf("query log: %w", err)
	}

	_, err = db.Exec(`UPDATE log_items
SET message = ?
WHERE log_id = ? AND raw_msg_id = ?`, newContent, logID, rid)
	if err != nil {
		return fmt.Errorf("log edit: %w", err)
	}

	return nil
}
