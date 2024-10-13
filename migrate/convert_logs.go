package migrate

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.etcd.io/bbolt"

	log "sealdice-core/utils/kratos"
)

type LogOneItem struct {
	ID          uint64      `json:"id"`
	Nickname    string      `json:"nickname"`
	IMUserID    string      `json:"IMUserId"`
	Time        int64       `json:"time"`
	Message     string      `json:"message"`
	IsDice      bool        `json:"isDice"`
	CommandID   uint64      `json:"commandId"`
	CommandInfo interface{} `json:"commandInfo"`
	RawMsgID    interface{} `json:"rawMsgId"`

	UniformID string `json:"uniformId"`
	Channel   string `json:"channel"` // 用于秘密团
}

type IMSession struct {
	ServiceAtNew map[string]*GroupInfo `json:"servicesAt" yaml:"servicesAt"`
}

type Dice struct {
	DB        *bbolt.DB  `yaml:"-"` // 数据库对象
	ImSession *IMSession `yaml:"imSession" jsbind:"imSession"`
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name                string `yaml:"name" jsbind:"name"` // 玩家昵称
	UserID              string `yaml:"userId" jsbind:"userId"`
	InGroup             bool   `yaml:"inGroup"`                                          // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime     int64  `yaml:"lastCommandTime" jsbind:"lastCommandTime"`         // 上次发送指令时间
	AutoSetNameTemplate string `yaml:"autoSetNameTemplate" jsbind:"autoSetNameTemplate"` // 名片模板

	// level int 权限
	DiceSideNum int `yaml:"diceSideNum"` // 面数，为0时等同于d100
}

type GroupPlayerInfo struct {
	GroupPlayerInfoBase `yaml:",inline,flow"`
}

type ExtInfo struct {
	Name string `yaml:"name" json:"name" jsbind:"name"` // 名字
}

type GroupInfo struct {
	GroupID   string `yaml:"groupId" json:"groupId" jsbind:"groupId"`
	GroupName string `yaml:"groupName" json:"groupName" jsbind:"groupName"`

	LogCurName string `yaml:"logCurFile" json:"logCurName" jsbind:"logCurName"`
	LogOn      bool   `yaml:"logOn" json:"logOn" jsbind:"logOn"`

	// ============================
	Active           bool                        `json:"active" yaml:"active" jsbind:"active"`          // 是否在群内开启 - 过渡为象征意义
	ActivatedExtList []*ExtInfo                  `yaml:"activatedExtList,flow" json:"activatedExtList"` // 当前群开启的扩展列表
	Players          map[string]*GroupPlayerInfo `yaml:"players" json:"-"`                              // 群员角色数据
	NotInGroup       bool                        `yaml:"notInGroup" json:"notInGroup"`                  // 是否已经离开群 - 准备处理单骰多号情况

	ActiveDiceIds   map[string]bool `yaml:"diceIds,flow" json:"diceIdActiveMap"` // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	DiceIDExistsMap map[string]bool `yaml:"-" json:"diceIdExistsMap"`            // 对应的骰子ID(格式 平台:ID)是否存在于群内
	BotList         map[string]bool `yaml:"botList,flow" json:"botList"`         // 其他骰子列表
	DiceSideNum     int64           `yaml:"diceSideNum" json:"diceSideNum"`      // 以后可能会支持 1d4 这种默认面数，暂不开放给js
	System          string          `yaml:"system" json:"system"`                // 规则系统，概念同bcdice的gamesystem，距离如dnd5e coc7

	CocRuleIndex int `yaml:"cocRuleIndex" json:"cocRuleIndex" jsbind:"cocRuleIndex"`

	RecentCommandTime   int64  `yaml:"recentCommandTime" json:"recentCommandTime" jsbind:"recentCommandTime"` // 最近一次发送有效指令的时间
	ShowGroupWelcome    bool   `yaml:"showGroupWelcome" json:"showGroupWelcome" jsbind:"showGroupWelcome"`    // 是否迎新
	GroupWelcomeMessage string `yaml:"groupWelcomeMessage" json:"groupWelcomeMessage" jsbind:"groupWelcomeMessage"`

	EnteredTime  int64  `yaml:"enteredTime" json:"enteredTime" jsbind:"enteredTime"`    // 入群时间
	InviteUserID string `yaml:"inviteUserId" json:"inviteUserId" jsbind:"inviteUserId"` // 邀请人
}

type MsgContext struct {
	Dice *Dice // 对应的 Dice
}

func BoltDBInit(path string) *bbolt.DB {
	db, err := bbolt.Open(path, 0644, nil)
	if err != nil {
		panic(err)
	}

	_ = db.Update(func(tx *bbolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("attrs_group"))      // 组属性
		_, _ = tx.CreateBucketIfNotExists([]byte("attrs_user"))       // 用户属性
		_, _ = tx.CreateBucketIfNotExists([]byte("attrs_group_user")) // 组_用户_属性
		return nil
	})

	return db
}

func CreateFakeCtx() *MsgContext {
	return &MsgContext{
		Dice: &Dice{
			DB: BoltDBInit("./data/default/data.bdb"),
		},
	}
}

// LogGetList 获取列表
func LogGetList(ctx *MsgContext, groupID string) ([]string, error) {
	var ret []string
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(groupID))
		if b1 == nil {
			// return errors.New("群组记录不存在，群号是否正确？例QQ-Group:12345")
			// 空列表
			return nil
		}

		return b1.ForEach(func(k, v []byte) error {
			if strings.HasSuffix(string(k), "-delMark") {
				// 跳过撤回记录
				return nil
			}
			ret = append(ret, string(k))
			return nil
		})
	})
}

func LogGetAllLines(ctx *MsgContext, groupID string, logName string) ([]*LogOneItem, error) {
	var ret []*LogOneItem
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(groupID))
		if b1 == nil {
			return errors.New("群组记录不存在，群号是否正确？例QQ-Group:12345")
		}

		b := b1.Bucket([]byte(logName))
		if b == nil {
			return errors.New("日志名不存在。请确认给定的日志名正确。")
		}

		return b.ForEach(func(k, v []byte) error {
			logItem := LogOneItem{}
			err := json.Unmarshal(v, &logItem)
			if err == nil {
				ret = append(ret, &logItem)
			}

			return nil
		})
	})
}

func LogGetAllLinesWithoutDeleted(ctx *MsgContext, groupID string, logName string) ([]*LogOneItem, error) {
	badRawIds, err2 := LogGetAllDeleted(ctx, groupID, logName)

	var ret []*LogOneItem
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(groupID))
		if b1 == nil {
			return nil
		}

		b := b1.Bucket([]byte(logName))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			logItem := LogOneItem{}
			err := json.Unmarshal(v, &logItem)
			if err == nil {
				// 跳过撤回
				if err2 == nil {
					if badRawIds[logItem.RawMsgID] {
						return nil
					}
				}
				// 正常添加
				ret = append(ret, &logItem)
			}

			return nil
		})
	})
}

func LogAppend(ctx *MsgContext, group *GroupInfo, l *LogOneItem) error {
	return ctx.Dice.DB.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("logs"))
		if err != nil {
			// ctx.Dice.Zlogger.Error("日志写入问题", err.Error())
			return err
		}

		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("logs"))
		b1, err := b0.CreateBucketIfNotExists([]byte(group.GroupID))
		if err != nil {
			return err
		}

		b, err := b1.CreateBucketIfNotExists([]byte(group.LogCurName))
		_ = b.Put([]byte("modified"), []byte(strconv.FormatInt(time.Now().Unix(), 10)))
		if err == nil {
			l.ID, _ = b.NextSequence()
			buf, errMarshal := json.Marshal(l)
			if errMarshal != nil {
				return errMarshal
			}

			return b.Put(itob(l.ID), buf)
		}
		return err
	})
}

func LogMarkDeleteByMsgID(ctx *MsgContext, group *GroupInfo, rawID interface{}) error {
	if rawID == nil {
		return nil
	}
	return ctx.Dice.DB.Update(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		b1, err := b0.CreateBucketIfNotExists([]byte(group.GroupID))
		if err != nil {
			return err
		}

		b, err := b1.CreateBucketIfNotExists([]byte(group.LogCurName + "-delMark"))
		if err == nil {
			id, _ := b.NextSequence()
			buf, errMarshal := json.Marshal(rawID)
			if errMarshal != nil {
				return errMarshal
			}

			return b.Put(itob(id), buf)
		}
		return err
	})
}

func LogGetAllDeleted(ctx *MsgContext, groupID string, logName string) (map[interface{}]bool, error) {
	ret := map[interface{}]bool{}
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(groupID))
		if b1 == nil {
			return nil
		}

		b := b1.Bucket([]byte(logName + "-delMark"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var val interface{}
			err := json.Unmarshal(v, &val)
			if err == nil {
				ret[val] = true
			}
			return nil
		})
	})
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func ConvertLogs() error {
	texts := []string{
		`
create table if not exists logs
(
    id         INTEGER  primary key autoincrement,
    name       TEXT,
    group_id   TEXT,
    extra      TEXT,
    created_at INTEGER,
    updated_at INTEGER,
    upload_url TEXT,
    upload_time INTEGER
);`,
		`
create index if not exists idx_logs_group
    on logs (group_id);`,
		`
create index if not exists idx_logs_update_at
    on logs (updated_at);`,
		`
create unique index if not exists idx_log_group_id_name
    on logs (group_id, name);`,
		`
create table if not exists log_items
(
    id              INTEGER primary key autoincrement,
    log_id          INTEGER,
    group_id        TEXT,
    nickname        TEXT,
    im_userid       TEXT,
    time            INTEGER,
    message         TEXT,
    is_dice         INTEGER,
    command_id      INTEGER,
    command_info    TEXT,
    raw_msg_id      TEXT,
    user_uniform_id TEXT,
    removed         INTEGER,
    parent_id       INTEGER
);`,
		`
create index if not exists idx_log_items_group_id
    on log_items (log_id);`,
		`
create index if not exists idx_log_items_log_id
    on log_items (log_id);`,
	}

	// flags := sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL
	dbDataLogsPath, _ := filepath.Abs("./data/default/data-logs.db")

	dbSQL, err := openDB(dbDataLogsPath)
	if err != nil {
		log.Error("xxx", err)
		return err
	}
	defer func(dbSql *sqlx.DB) {
		_ = dbSql.Close()
	}(dbSQL)

	for _, i := range texts {
		_, _ = dbSQL.Exec(i)
	}

	// 加载数据
	ctx := CreateFakeCtx()
	db := ctx.Dice.DB
	defer func(db *bbolt.DB) {
		_ = db.Close()
	}(db)

	var groupIds []string
	_ = db.View(func(tx *bbolt.Tx) error {
		logs := tx.Bucket([]byte("logs"))
		return logs.ForEach(func(k, v []byte) error {
			groupIds = append(groupIds, string(k))
			return nil
		})
	})

	log.Info("群组数量", len(groupIds))

	times := 0
	itemNumber := 0

	now := time.Now()
	nowTimestamp := now.Unix()

	logNum := 0

	num := 0
	err = dbSQL.Get(&num, "select count(id) from log_items")
	if err != nil {
		return err
	}

	for _, i := range groupIds {
		lst, _ := LogGetList(ctx, i)
		times += len(lst)

		for _, j := range lst {
			args := map[string]interface{}{
				"name":       j,
				"group_id":   i,
				"created_at": nowTimestamp,
				"updated_at": nowTimestamp,
			}
			exec, errExec := dbSQL.NamedExec(`insert into logs (name, group_id, created_at, updated_at) VALUES (:name, :group_id, :created_at, :updated_at)`, args)
			if errExec != nil {
				log.Error("错误:", errExec, i, j)
				return errExec
			}

			logID, _ := exec.LastInsertId()
			logNum++
			if logNum%10 == 0 {
				log.Infof("进度: %d\n", logNum)
			}

			tx := dbSQL.MustBegin()
			items, _ := LogGetAllLines(ctx, i, j)
			itemNumber += len(items)

			for _, logItem := range items {
				d, _ := json.Marshal(logItem.CommandInfo)
				d2, _ := json.Marshal(logItem.RawMsgID)

				args := map[string]interface{}{
					"log_id":          logID,
					"group_id":        i,
					"nickname":        logItem.Nickname,
					"im_userid":       logItem.IMUserID,
					"time":            logItem.Time,
					"message":         logItem.Message,
					"is_dice":         logItem.IsDice,
					"command_id":      int64(logItem.CommandID),
					"command_info":    d,
					"raw_msg_id":      d2,
					"user_uniform_id": logItem.UniformID,
				}

				_, _ = tx.NamedExec(`insert into log_items (log_id, group_id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id) VALUES (:log_id, :group_id, :nickname, :im_userid, :time, :message, :is_dice, :command_id, :command_info, :raw_msg_id, :user_uniform_id)`, args)
			}
			errExec = tx.Commit()
			if errExec != nil {
				_ = tx.Rollback()
			}
		}
	}

	log.Info("群组数量", len(groupIds))
	log.Info("log完成", times)
	log.Info("行数", itemNumber)

	err = dbSQL.Get(&num, "select count(id) from log_items")
	if err != nil {
		return err
	}
	log.Info("行数确认", num)

	_ = dbSQL.Close()
	return nil
}
