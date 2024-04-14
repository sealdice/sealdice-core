package model

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type EndpointInfo struct {
	UserID      string `db:"user_id"`
	CmdNum      int64  `db:"cmd_num"`
	CmdLastTime int64  `db:"cmd_last_time"`
	OnlineTime  int64  `db:"online_time"`
	UpdatedAt   int64  `db:"updated_at"`
}

var ErrEndpointInfoUIDEmpty = errors.New("user id is empty")

func (e *EndpointInfo) Query(db *sqlx.DB) error {
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}
	if db == nil {
		return errors.New("db is nil")
	}
	row := db.QueryRowx(
		`SELECT cmd_num, cmd_last_time, online_time, updated_at FROM endpoint_info WHERE user_id = $1`,
		e.UserID,
	)
	err := row.Scan(&e.CmdNum, &e.CmdLastTime, &e.OnlineTime, &e.UpdatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

func (e *EndpointInfo) Save(db *sqlx.DB) error {
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}
	if db == nil {
		return errors.New("db is nil")
	}
	now := time.Now().Unix()
	e.UpdatedAt = now

	_, err := db.Exec(
		`REPLACE INTO endpoint_info (user_id, cmd_num, cmd_last_time, online_time, updated_at) VALUES (?, ?, ?, ?, ?)`,
		e.UserID, e.CmdNum, e.CmdLastTime, e.OnlineTime, e.UpdatedAt,
	)
	return err
}
