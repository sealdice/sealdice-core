package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"go.etcd.io/bbolt"
	"gopkg.in/yaml.v3"
)

type DeckInfo struct {
	Enable        bool                 `json:"enable" yaml:"enable"`
	Filename      string               `json:"filename" yaml:"filename"`
	Format        string               `json:"format" yaml:"format"`               // 几种：“SinaNya” ”Dice!“
	FormatVersion int64                `json:"formatVersion" yaml:"formatVersion"` // 格式版本，默认都是1
	FileFormat    string               `json:"fileFormat" yaml:"-" `               // json / yaml
	Name          string               `json:"name" yaml:"name"`
	Version       string               `json:"version" yaml:"-"`
	Author        string               `json:"author" yaml:"-"`
	Command       map[string]bool      `json:"command" yaml:"-"` // 牌堆命令名
	DeckItems     map[string][]string  `yaml:"-" json:"-"`
	Date          string               `json:"date" yaml:"-" `
	UpdateDate    string               `json:"updateDate" yaml:"-" `
	Desc          string               `yaml:"-" json:"desc"`
	Info          []string             `yaml:"-" json:"-"`
	RawData       *map[string][]string `yaml:"-" json:"-"`
}

type ExtDefaultSettingItem struct {
	Name            string          `yaml:"name" json:"name"`
	AutoActive      bool            `yaml:"autoActive" json:"autoActive"`                // 是否自动开启
	DisabledCommand map[string]bool `yaml:"disabledCommand,flow" json:"disabledCommand"` // 实际为set
	ExtItem         *ExtInfo        `yaml:"-" json:"-"`
}

type BanListInfo struct {
	BanBehaviorRefuseReply   bool  `yaml:"banBehaviorRefuseReply" json:"banBehaviorRefuseReply"`     // 拉黑行为: 拒绝回复
	BanBehaviorRefuseInvite  bool  `yaml:"banBehaviorRefuseInvite" json:"banBehaviorRefuseInvite"`   // 拉黑行为: 拒绝邀请
	BanBehaviorQuitLastPlace bool  `yaml:"banBehaviorQuitLastPlace" json:"banBehaviorQuitLastPlace"` // 拉黑行为: 退出事发群
	ThresholdWarn            int64 `yaml:"thresholdWarn" json:"thresholdWarn"`                       // 警告阈值
	ThresholdBan             int64 `yaml:"thresholdBan" json:"thresholdBan"`                         // 错误阈值
	AutoBanMinutes           int64 `yaml:"autoBanMinutes" json:"autoBanMinutes"`                     // 自动禁止时长

	ScoreReducePerMinute int64 `yaml:"scoreReducePerMinute" json:"scoreReducePerMinute"` // 每分钟下降
	ScoreGroupMuted      int64 `yaml:"scoreGroupMuted" json:"scoreGroupMuted"`           // 群组禁言
	ScoreGroupKicked     int64 `yaml:"scoreGroupKicked" json:"scoreGroupKicked"`         // 群组踢出
	ScoreTooManyCommand  int64 `yaml:"scoreTooManyCommand" json:"scoreTooManyCommand"`   // 刷指令

	JointScorePercentOfGroup   float64 `yaml:"jointScorePercentOfGroup" json:"jointScorePercentOfGroup"`     // 群组连带责任
	JointScorePercentOfInviter float64 `yaml:"jointScorePercentOfInviter" json:"jointScorePercentOfInviter"` // 邀请人连带责任
}

type EndPointInfoBase struct {
	Id                  string `yaml:"id" json:"id"` // uuid
	Nickname            string `yaml:"nickname" json:"nickname"`
	State               int    `yaml:"state" json:"state"` // 状态 0 断开 1已连接 2连接中 3连接失败
	UserId              string `yaml:"userId" json:"userId"`
	GroupNum            int64  `yaml:"groupNum" json:"groupNum"`                       // 拥有群数
	CmdExecutedNum      int64  `yaml:"cmdExecutedNum" json:"cmdExecutedNum"`           // 指令执行次数
	CmdExecutedLastTime int64  `yaml:"cmdExecutedLastTime" json:"cmdExecutedLastTime"` // 指令执行次数
	OnlineTotalTime     int64  `yaml:"onlineTotalTime" json:"onlineTotalTime"`         // 在线时长

	Platform     string `yaml:"platform" json:"platform"`     // 平台，如QQ等
	RelWorkDir   string `yaml:"relWorkDir" json:"relWorkDir"` // 工作目录
	Enable       bool   `yaml:"enable" json:"enable"`         // 是否启用
	ProtocolType string `yaml:"protocolType"`                 // 协议类型，如onebot、koishi等

	IsPublic bool `yaml:"isPublic"`
}

type EndPointInfo struct {
	EndPointInfoBase `yaml:"baseInfo"`

	// 下面这个能保留全部数据结构吗？
	Adapter interface{} `yaml:"adapter" json:"adapter"`
}

type IMSessionServe struct {
	EndPoints []*EndPointInfo `yaml:"endPoints"`
}

type DiceServe struct {
	ImSession               *IMSessionServe `yaml:"imSession" jsbind:"imSession"`
	ExtList                 []*ExtInfo      `yaml:"-"`
	CommandCompatibleMode   bool            `yaml:"commandCompatibleMode"`
	LastSavedTime           *time.Time      `yaml:"lastSavedTime"`
	LastUpdatedTime         *time.Time      `yaml:"-"`
	IsDeckLoading           bool            `yaml:"-"`                                    // 正在加载中
	DeckList                []*DeckInfo     `yaml:"deckList" jsbind:"deckList"`           // 牌堆信息
	CommandPrefix           []string        `yaml:"commandPrefix" jsbind:"commandPrefix"` // 指令前导
	DiceMasters             []string        `yaml:"diceMasters" jsbind:"diceMasters"`     // 骰主设置，需要格式: 平台:帐号
	NoticeIds               []string        `yaml:"noticeIds"`                            // 通知ID
	OnlyLogCommandInGroup   bool            `yaml:"onlyLogCommandInGroup"`                // 日志中仅记录命令
	OnlyLogCommandInPrivate bool            `yaml:"onlyLogCommandInPrivate"`              // 日志中仅记录命令
	VersionCode             int             `json:"versionCode"`                          // 版本ID(配置文件)
	MessageDelayRangeStart  float64         `yaml:"messageDelayRangeStart"`               // 指令延迟区间
	MessageDelayRangeEnd    float64         `yaml:"messageDelayRangeEnd"`
	WorkInQQChannel         bool            `yaml:"workInQQChannel"`
	QQChannelAutoOn         bool            `yaml:"QQChannelAutoOn"`     // QQ频道中自动开启(默认不开)
	QQChannelLogMessage     bool            `yaml:"QQChannelLogMessage"` // QQ频道中记录消息(默认不开)
	UILogLimit              int64           `yaml:"UILogLimit"`
	FriendAddComment        string          `yaml:"friendAddComment"` // 加好友验证信息
	MasterUnlockCode        string          `yaml:"-"`                // 解锁码，每20分钟变化一次，使用后立即变化
	MasterUnlockCodeTime    int64           `yaml:"-"`
	CustomReplyConfigEnable bool            `yaml:"customReplyConfigEnable"`
	AutoReloginEnable       bool            `yaml:"autoReloginEnable"` // 启用自动重新登录
	RefuseGroupInvite       bool            `yaml:"refuseGroupInvite"` // 拒绝加入新群
	UpgradeWindowId         string          `yaml:"upgradeWindowId"`   // 执行升级指令的窗口
	BotExtFreeSwitch        bool            `yaml:"botExtFreeSwitch"`  // 允许任意人员开关: 否则邀请者、群主、管理员、master有权限
	TrustOnlyMode           bool            `yaml:"trustOnlyMode"`     // 只有信任的用户/master可以拉群和使用
	AliveNoticeEnable       bool            `yaml:"aliveNoticeEnable"` // 定时通知
	AliveNoticeValue        string          `yaml:"aliveNoticeValue"`  // 定时通知间隔
	ReplyDebugMode          bool            `yaml:"replyDebugMode"`    // 回复调试

	DefaultCocRuleIndex int64 `yaml:"defaultCocRuleIndex" jsbind:"defaultCocRuleIndex"` // 默认coc index

	ExtDefaultSettings []*ExtDefaultSettingItem `yaml:"extDefaultSettings"` // 新群扩展按此顺序加载

	BanList *BanListInfo `yaml:"banList"` //

	RunAfterLoaded []func() `yaml:"-" json:"-"`

	LogSizeNoticeEnable bool `yaml:"logSizeNoticeEnable"` // 开启日志数量提示
	LogSizeNoticeCount  int  `yaml:"LogSizeNoticeCount"`  // 日志数量提示阈值，默认500

	IsAlreadyLoadConfig bool `yaml:"-"` // 如果在loads前崩溃，那么不写入配置，防止覆盖为空的
}

type BanRankType int

const (
	BanRankBanned  BanRankType = -30
	BanRankWarn    BanRankType = -10
	BanRankNormal  BanRankType = 0
	BanRankTrusted BanRankType = 30
)

type BanListInfoItem struct {
	ID      string      `json:"ID"`
	Name    string      `json:"name"`
	Score   int64       `json:"score"`
	Rank    BanRankType `json:"rank"`    // 0 没事 -10警告 -30禁止 30信任
	Times   []int64     `json:"times"`   // 事发时间
	Reasons []string    `json:"reasons"` // 拉黑原因
	Places  []string    `json:"places"`  // 发生地点
	BanTime int64       `json:"banTime"` // 上黑名单时间
}

func ConvertServe() error {
	data, err := os.ReadFile("./data/default/serve.yaml")
	if err != nil {
		return err
	}
	dbDataPath, _ := filepath.Abs("./data/default/data.db")
	dbSql, err := openDB(dbDataPath)
	if err != nil {
		return err
	}
	defer func(dbSql *sqlx.DB) {
		_ = dbSql.Close()
	}(dbSql)

	texts := []string{
		`
create table if not exists group_player_info
(
    id                     INTEGER
        primary key autoincrement,
    group_id               TEXT,
    user_id                TEXT,
    name                   TEXT,
    created_at             INTEGER,
    updated_at             INTEGER,
    last_command_time      INTEGER,
    auto_set_name_template TEXT,
    dice_side_num          TEXT
);`,
		`create index if not exists idx_group_player_info_group_id on group_player_info (group_id);`,
		`create index if not exists idx_group_player_info_user_id on group_player_info (user_id);`,
		`create unique index if not exists idx_group_player_info_group_user on group_player_info (group_id, user_id);`,
		`
create table if not exists group_info
(
    id         TEXT primary key,
    created_at INTEGER,
    updated_at INTEGER,
    data       BLOB
);`,

		`
create table if not exists attrs_group
(
    id         TEXT primary key,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_attrs_group_updated_at on attrs_group (updated_at);`,
		`create table if not exists attrs_group_user
(
    id         TEXT primary key,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_attrs_group_user_updated_at on attrs_group_user (updated_at);`,
		`create table if not exists attrs_user
(
    id         TEXT primary key,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_attrs_user_updated_at on attrs_user (updated_at);`,

		`
create table if not exists ban_info
(
    id         TEXT primary key,
    ban_updated_at INTEGER,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index idx_ban_info_updated_at on ban_info (updated_at);`,
		`create index idx_ban_info_ban_updated_at on ban_info (ban_updated_at);`,
	}

	for _, i := range texts {
		_, _ = dbSql.Exec(i)
	}
	now := time.Now()
	nowTimestamp := now.Unix()

	fmt.Println("处理serve.yaml")

	times := 0
	dNew := &Dice{}
	if yaml.Unmarshal(data, &dNew) == nil {
		tx := dbSql.MustBegin()

		for k, v := range dNew.ImSession.ServiceAtNew {
			fmt.Println("群组", k)
			times += len(v.Players)
			for _, playerInfo := range v.Players {
				args := map[string]interface{}{
					"group_id":               k,
					"user_id":                playerInfo.UserID,
					"created_at":             nowTimestamp,
					"name":                   playerInfo.Name,
					"last_command_time":      playerInfo.LastCommandTime,
					"auto_set_name_template": playerInfo.AutoSetNameTemplate,
					"dice_side_num":          int64(playerInfo.DiceSideNum),
				}
				_, _ = tx.NamedExec(`insert into group_player_info (group_id, user_id, created_at, name, last_command_time, auto_set_name_template, dice_side_num) VALUES (:group_id, :user_id, :created_at, :name, :last_command_time, :auto_set_name_template, :dice_side_num)`, args)
				// $group_id, $user_id, $created_at, $last_command_time, $auto_set_name_template, $dice_side_num
			}

			v.Players = nil
			v.DiceIDExistsMap = v.ActiveDiceIds // 暂时视为相同
			d, _ := json.Marshal(v)

			args := map[string]interface{}{
				"group_id":   k,
				"created_at": nowTimestamp,
				"data":       d,
			}

			_, _ = tx.NamedExec(`insert into group_info (id, created_at, data) VALUES (:group_id, :created_at, :data)`, args)
		}

		errTx := tx.Commit()
		if errTx != nil {
			fmt.Println("???", errTx)
			_ = tx.Rollback()
		}

		fmt.Println("群组信息处理完成")
		fmt.Println("群数量", len(dNew.ImSession.ServiceAtNew))
		fmt.Println("群成员数量", times)
	}

	_ = os.WriteFile("./data/default/serve.yaml.old", data, 0o644)

	// 处理attrs部分
	ctx := CreateFakeCtx()
	db := ctx.Dice.DB
	defer func(db *bbolt.DB) {
		_ = db.Close()
	}(db)

	fmt.Println("处理属性部分")
	copyByName := func(table string) {
		times = 0
		tx2 := dbSql.MustBegin()

		_ = db.View(func(tx *bbolt.Tx) error {
			logs := tx.Bucket([]byte(table))

			return logs.ForEach(func(k, v []byte) error {
				_, errExec := tx2.NamedExec(`INSERT INTO `+table+` (id, updated_at, data) VALUES (:id, :updated_at, :data)`, map[string]interface{}{
					"id":         string(k),
					"updated_at": nowTimestamp,
					"data":       v,
				})
				times += 1
				return errExec
			})
		})

		fmt.Println("条目数量"+table, times)

		if tx2.Commit() != nil {
			_ = tx2.Rollback()
			return
		}
	}

	copyByName("attrs_group")
	copyByName("attrs_user")
	copyByName("attrs_group_user")
	fmt.Println("完成")

	times = 0
	tx2 := dbSql.MustBegin()
	_ = db.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("common"))
		if b0 == nil {
			return nil
		}
		data = b0.Get([]byte("banMap"))

		dict := map[string]*BanListInfoItem{}
		errUnmarshal := json.Unmarshal(data, &dict)
		if errUnmarshal != nil {
			return errUnmarshal
		}

		for k, v := range dict {
			data, _ := json.Marshal(v)

			times += 1
			_, _ = tx2.NamedExec(`replace into ban_info (id, ban_updated_at, updated_at, data) VALUES (:id, :ban_updated_at, :updated_at, :data)`,
				map[string]interface{}{
					"id":             k,
					"ban_updated_at": v.BanTime,
					"updated_at":     v.BanTime,
					"data":           data,
				})
		}
		return nil
	})

	err = tx2.Commit()
	if err != nil {
		_ = tx2.Rollback()
	}

	fmt.Println("黑名单条目数量", times)
	fmt.Println("完成")

	return nil
}
