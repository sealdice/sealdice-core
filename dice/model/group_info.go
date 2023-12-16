package model

import (
	"fmt"

	"github.com/fy0/lockfree"
	"github.com/jmoiron/sqlx"
	"golang.org/x/time/rate"
)

func GroupInfoListGet(db *sqlx.DB, callback func(id string, updatedAt int64, data []byte)) error {
	rows, err := db.Queryx("SELECT id, updated_at, data FROM group_info")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var updatedAt int64
		var data []byte

		var pUpdatedAt *int64

		err = rows.Scan(&id, &pUpdatedAt, &data)
		if err != nil {
			fmt.Println("!!!", err.Error())
			return err
		}

		if pUpdatedAt != nil {
			updatedAt = *pUpdatedAt
		}
		callback(id, updatedAt, data)
	}

	return rows.Err()
}

// GroupInfoSave 保存群组信息
func GroupInfoSave(db *sqlx.DB, groupID string, updatedAt int64, data []byte) error {
	// INSERT OR REPLACE 语句可以根据是否已存在对应记录自动插入或更新记录
	_, err := db.Exec("INSERT OR REPLACE INTO group_info (id, updated_at, data) VALUES (?, ?, ?)", groupID, updatedAt, data)
	return err
}

// GroupPlayerNumGet 查询指定群组中玩家数量
func GroupPlayerNumGet(db *sqlx.DB, groupID string) (int64, error) {
	var count int64

	// 使用Named方法绑定命名参数
	// 	sqlitex.ExecuteTransient(conn, `select count(id) from group_player_info where group_id=$group_id`, &sqlitex.ExecOptions{
	query, args, err := sqlx.Named("SELECT COUNT(id) FROM group_player_info WHERE group_id = :group_id", map[string]interface{}{"group_id": groupID})
	if err != nil {
		return 0, err
	}

	// 执行查询并将结果存储到 count 变量中
	if err := db.QueryRowx(query, args...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

type PlayerVariablesItem struct {
	Loaded        bool             `yaml:"-"`
	ValueMap      lockfree.HashMap `yaml:"-"`
	LastWriteTime int64            `yaml:"lastUsedTime"`
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name                string        `yaml:"name" jsbind:"name"` // 玩家昵称
	UserID              string        `yaml:"userId" jsbind:"userId"`
	InGroup             bool          `yaml:"inGroup"`                                          // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime     int64         `yaml:"lastCommandTime" jsbind:"lastCommandTime"`         // 上次发送指令时间
	RateLimiter         *rate.Limiter `yaml:"-"`                                                // 限速器
	RateLimitWarned     bool          `yaml:"-"`                                                // 是否已经警告过限速
	AutoSetNameTemplate string        `yaml:"autoSetNameTemplate" jsbind:"autoSetNameTemplate"` // 名片模板

	// level int 权限
	DiceSideNum  int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	Vars         *PlayerVariablesItem `yaml:"-"`           // 玩家的群内变量
	ValueMapTemp lockfree.HashMap     `yaml:"-"`           // 玩家的群内临时变量
	// ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	TempValueAlias *map[string][]string `yaml:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	UpdatedAtTime  int64 `yaml:"-" json:"-"`
	RecentUsedTime int64 `yaml:"-" json:"-"`
}

func GroupPlayerInfoGet(db *sqlx.DB, groupID string, playerID string) *GroupPlayerInfoBase {
	var ret GroupPlayerInfoBase

	rows, err := db.NamedQuery("SELECT name, last_command_time, auto_set_name_template, dice_side_num FROM group_player_info WHERE group_id=:group_id AND user_id=:user_id", map[string]interface{}{
		"group_id": groupID,
		"user_id":  playerID,
	})

	if err != nil {
		fmt.Printf("error getting group player info: %s", err.Error())
		return nil
	}

	defer rows.Close()

	//Name:                stmt.ColumnText(0),
	//UserId:              playerId,
	//LastCommandTime:     stmt.ColumnInt64(2),
	//AutoSetNameTemplate: stmt.ColumnText(3),
	//DiceSideNum:         int(stmt.ColumnInt64(4)),

	exists := false
	for rows.Next() {
		exists = true
		// 使用Scan方法将查询结果映射到结构体中
		if err := rows.Scan(
			&ret.Name,
			&ret.LastCommandTime,
			&ret.AutoSetNameTemplate,
			&ret.DiceSideNum,
		); err != nil {
			fmt.Printf("error getting group player info: %s", err.Error())
			return nil
		}
	}

	if !exists {
		return nil
	}
	ret.UserID = playerID
	return &ret
}

func GroupPlayerInfoSave(db *sqlx.DB, groupID string, playerID string, info *GroupPlayerInfoBase) error {
	_, err := db.NamedExec("REPLACE INTO group_player_info (name, updated_at, last_command_time, auto_set_name_template, dice_side_num, group_id, user_id) VALUES (:name, :updated_at, :last_command_time, :auto_set_name_template, :dice_side_num, :group_id, :user_id)", map[string]interface{}{
		"name":                   info.Name,
		"updated_at":             info.UpdatedAtTime,
		"last_command_time":      info.LastCommandTime,
		"auto_set_name_template": info.AutoSetNameTemplate,
		"dice_side_num":          info.DiceSideNum,
		"group_id":               groupID,
		"user_id":                playerID,
	})
	return err
}
