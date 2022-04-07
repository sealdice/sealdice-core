package dice

import (
	"github.com/fy0/lockfree"
	"github.com/fy0/procs"
	"github.com/sacOO7/gowebsocket"
)

// PlayerInfo 群内玩家信息
type PlayerInfo struct {
	UserId int64  `yaml:"userId"`
	UID    string `yaml:"uid"`
	Name   string // 玩家昵称
	//ValueNumMap    map[string]int64  `yaml:"valueNumMap"`
	//ValueStrMap    map[string]string `yaml:"valueStrMap"`
	LastUpdateTime int64 `yaml:"lastUpdateTime"`
	InGroup        bool  `yaml:"inGroup"`

	// level int 权限
	DiceSideNum    int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	TempValueAlias *map[string][]string `yaml:"-"`

	ValueMap     map[string]*VMValue `yaml:"-"`
	ValueMapTemp map[string]*VMValue `yaml:"-"`
}

type ServiceAtItem struct {
	Active           bool                  `json:"active" yaml:"active"` // 需要能记住配置，故有此选项
	ActivatedExtList []*ExtInfo            `yaml:"activatedExtList"`     // 当前群开启的扩展列表
	Players          map[int64]*PlayerInfo // 群员角色数据
	NotInGroup       bool                  // 是否已经离开群

	LogCurName  string          `yaml:"logCurFile"`
	LogOn       bool            `yaml:"logOn"`
	GroupId     int64           `yaml:"groupId"`
	GroupName   string          `yaml:"groupName"`
	Platform    string          `yaml:"platform"` // 默认为QQ（为空）
	DiceIds     map[string]bool `yaml:"diceIds"`  // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	DiceSideNum int64           `yaml:"diceSideNum"`
	BotList     map[string]bool `yaml:"botList"` // 其他骰子列表

	ValueMap     map[string]*VMValue `yaml:"-"`
	CocRuleIndex int                 `yaml:"cocRuleIndex"`
	HelpPackages []string            `yaml:"-"`

	// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
	//RuleCriticalSuccessValue *int64 // 大成功值，1默认
	//RuleFumbleValue *int64 // 大失败值 96默认
}

type PlayerVariablesItem struct {
	Loaded        bool             `yaml:"-"`
	ValueMap      lockfree.HashMap `yaml:"-"`
	LastWriteTime int64            `yaml:"lastUsedTime"`
	//ValueMap            map[string]*VMValue `yaml:"-"`
}

type ConnectInfoItem struct {
	//InPackGoCqHttpPassword            string              `yaml:"password" json:"password"`
	Socket              *gowebsocket.Socket `yaml:"-" json:"-"`
	Id                  string              `yaml:"id" json:"id"` // uuid
	Nickname            string              `yaml:"nickname" json:"nickname"`
	State               int                 `yaml:"state" json:"state"` // 状态 0 断开 1已连接 2连接中
	UserId              int64               `yaml:"userId" json:"userId"`
	UniformID           string              `yaml:"uid" json:"uid"`
	GroupNum            int64               `yaml:"groupNum" json:"groupNum"`                       // 拥有群数
	CmdExecutedNum      int64               `yaml:"cmdExecutedNum" json:"cmdExecutedNum"`           // 指令执行次数
	CmdExecutedLastTime int64               `yaml:"cmdExecutedLastTime" json:"cmdExecutedLastTime"` // 指令执行次数
	OnlineTotalTime     int64               `yaml:"onlineTotalTime" json:"onlineTotalTime"`         // 在线时长
	ConnectUrl          string              `yaml:"connectUrl" json:"connectUrl"`                   // 连接地址

	Platform          string `yaml:"platform" json:"platform"`                   // 平台，如QQ等
	RelWorkDir        string `yaml:"relWorkDir" json:"relWorkDir"`               // 工作目录
	Enable            bool   `yaml:"enable" json:"enable"`                       // 是否启用
	Type              string `yaml:"type" json:"type"`                           // 协议类型，如onebot、koishi等
	UseInPackGoCqhttp bool   `yaml:"useInPackGoCqhttp" json:"useInPackGoCqhttp"` // 是否使用内置的gocqhttp

	InPackGoCqHttpProcess            *procs.Process `yaml:"-" json:"-"`
	InPackGoCqHttpLoginSuccess       bool           `yaml:"-" json:"inPackGoCqHttpLoginSuccess"`   // 是否登录成功
	InPackGoCqHttpLoginSucceeded     bool           `yaml:"inPackGoCqHttpLoginSucceeded" json:"-"` // 是否登录成功过
	InPackGoCqHttpRunning            bool           `yaml:"-" json:"inPackGoCqHttpRunning"`        // 是否仍在运行
	InPackGoCqHttpQrcodeReady        bool           `yaml:"-" json:"inPackGoCqHttpQrcodeReady"`    // 二维码已就绪
	InPackGoCqHttpNeedQrCode         bool           `yaml:"-" json:"inPackGoCqHttpNeedQrCode"`     // 是否需要二维码
	InPackGoCqHttpQrcodeData         []byte         `yaml:"-" json:"-"`                            // 二维码数据
	InPackGoCqHttpLoginDeviceLockUrl string         `yaml:"-" json:"inPackGoCqHttpLoginDeviceLockUrl"`
	InPackGoCqHttpLastRestrictedTime int64          `yaml:"inPackGoCqHttpLastRestricted" json:"inPackGoCqHttpLastRestricted"` // 上次风控时间
	InPackGoCqHttpProtocol           int            `yaml:"inPackGoCqHttpProtocol" json:"inPackGoCqHttpProtocol"`
	InPackGoCqHttpPassword           string         `yaml:"inPackGoCqHttpPassword" json:"-"`
	DiceServing                      bool           `yaml:"-"` // 是否正在连接中
}

type IMSessionLegacy struct {
	Conns          []*ConnectInfoItem             `yaml:"connections"`
	Parent         *Dice                          `yaml:"-"`
	ServiceAt      map[int64]*ServiceAtItem       `json:"serviceAt" yaml:"serviceAt"`
	PlayerVarsData map[int64]*PlayerVariablesItem `yaml:"PlayerVarsData"`

	//CommandIndex int64                    `yaml:"-"`
	//GroupId int64 `json:"group_id"`
}

var curCommandId uint64 = 0

func getNextCommandId() uint64 {
	curCommandId += 1
	return curCommandId
}
