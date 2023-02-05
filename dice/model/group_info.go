package model

import (
	"github.com/fy0/lockfree"
	"strconv"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func GroupInfoListGet(db *sqlitex.Pool, callback func(id string, updatedAt int64, data []byte)) error {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	err := sqlitex.ExecuteTransient(conn, `select id, updated_at, data from group_info`, &sqlitex.ExecOptions{
		//Named: map[string]interface{}{},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			callback(stmt.ColumnText(0), stmt.ColumnInt64(1), []byte(stmt.ColumnText(2)))
			return nil
		},
	})

	return err
}

func GroupInfoSave(db *sqlitex.Pool, groupId string, updatedAt int64, data []byte) {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	stmt := conn.Prep(`
		replace into group_info (id, updated_at, data)
		VALUES ($id, $updated_at, $data)`)
	defer stmt.Finalize()

	stmt.SetInt64("$updated_at", updatedAt)
	stmt.SetBytes("$data", data)
	stmt.SetText("$id", groupId)

	for {
		if hasRow, err := stmt.Step(); err != nil {
			break
		} else if !hasRow {
			break
		}
	}
}

func GroupPlayerNumGet(db *sqlitex.Pool, groupId string) int64 {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	var ret int64
	sqlitex.ExecuteTransient(conn, `select count(id) from group_player_info where group_id=$group_id`, &sqlitex.ExecOptions{
		Named: map[string]interface{}{
			"$group_id": groupId,
		},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			// 这玩意内部封装了for的过程
			ret = stmt.ColumnInt64(0)
			return nil
		},
	})

	return ret
}

type PlayerVariablesItem struct {
	Loaded        bool             `yaml:"-"`
	ValueMap      lockfree.HashMap `yaml:"-"`
	LastWriteTime int64            `yaml:"lastUsedTime"`
	//ValueMap            map[string]*VMValue `yaml:"-"`
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name                string `yaml:"name" jsbind:"name"` // 玩家昵称
	UserId              string `yaml:"userId" jsbind:"userId"`
	InGroup             bool   `yaml:"inGroup"`                                          // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime     int64  `yaml:"lastCommandTime" jsbind:"lastCommandTime"`         // 上次发送指令时间
	AutoSetNameTemplate string `yaml:"autoSetNameTemplate" jsbind:"autoSetNameTemplate"` // 名片模板

	// level int 权限
	DiceSideNum  int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	Vars         *PlayerVariablesItem `yaml:"-"`           // 玩家的群内变量
	ValueMapTemp lockfree.HashMap     `yaml:"-"`           // 玩家的群内临时变量
	//ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	TempValueAlias *map[string][]string `yaml:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	UpdatedAtTime  int64 `yaml:"-" json:"-"`
	RecentUsedTime int64 `yaml:"-" json:"-"`
}

func GroupPlayerInfoGet(db *sqlitex.Pool, groupId string, playerId string) *GroupPlayerInfoBase {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	var ret *GroupPlayerInfoBase
	sqlitex.ExecuteTransient(conn, `select name, updated_at, last_command_time, auto_set_name_template, dice_side_num from group_info where group_id=$group_id and user_id=$user_id`, &sqlitex.ExecOptions{
		Named: map[string]interface{}{
			"$group_id": groupId,
			"$user_id":  playerId,
		},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			// 这玩意内部封装了for的过程
			// name, updated_at, last_command_time, auto_set_name_template, dice_side_num
			ret = &GroupPlayerInfoBase{
				Name:                stmt.ColumnText(0),
				UserId:              playerId,
				LastCommandTime:     stmt.ColumnInt64(2),
				AutoSetNameTemplate: stmt.ColumnText(3),
				DiceSideNum:         int(stmt.ColumnInt64(4)),
				//ValueMapTemp:        lockfree.NewHashMap(),
			}
			return nil
		},
	})

	return ret
}

func GroupPlayerInfoSave(db *sqlitex.Pool, groupId string, playerId string, info *GroupPlayerInfoBase) {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	stmt := conn.Prep(`
		replace into group_player_info (name, updated_at, last_command_time, auto_set_name_template, dice_side_num, group_id, user_id)
		VALUES ($name, $updated_at, $last_command_time, $auto_set_name_template, $dice_side_num, $group_id, $user_id)`)
	defer stmt.Finalize()

	// $name, $updated_at, $last_command_time, $auto_set_name_template, $dice_side_num
	stmt.SetText("$name", info.Name)
	stmt.SetInt64("$updated_at", info.UpdatedAtTime)
	stmt.SetInt64("$last_command_time", info.LastCommandTime)
	stmt.SetText("$auto_set_name_template", info.AutoSetNameTemplate)
	stmt.SetText("$dice_side_num", strconv.Itoa(info.DiceSideNum))

	stmt.SetText("$group_id", groupId)
	stmt.SetText("$user_id", playerId)

	for {
		if hasRow, err := stmt.Step(); err != nil {
			break
		} else if !hasRow {
			break
		}
	}
}
