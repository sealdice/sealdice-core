package dice

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sealdice-core/message"
	"sealdice-core/utils/procs"
	"sealdice-core/utils/syncmap"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/sacOO7/gowebsocket"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// 0 默认 1登录中 2登录中-二维码 3登录中-滑条 4登录中-手机验证码 10登录成功 11登录失败

const (
	StateCodeInit                      = 0
	StateCodeInLogin                   = 1
	StateCodeInLoginQrCode             = 2
	GoCqhttpStateCodeInLoginBar        = 3
	GoCqhttpStateCodeInLoginVerifyCode = 6
	StateCodeInLoginDeviceLock         = 7
	StateCodeLoginSuccessed            = 10
	StateCodeLoginFailed               = 11
	GoCqhttpStateCodeClosed            = 20
)

type echoMapInfo struct {
	ch            chan string
	echoOverwrite any
	timeout       int64
}

type PlatformAdapterGocq struct {
	EndPoint *EndPointInfo `yaml:"-" json:"-"`
	Session  *IMSession    `yaml:"-" json:"-"`

	IsReverse   bool       `yaml:"isReverse" json:"isReverse" `
	ReverseAddr string     `yaml:"reverseAddr" json:"reverseAddr"`
	reverseApp  *echo.Echo `yaml:"-" json:"-"`

	Socket      *gowebsocket.Socket `yaml:"-" json:"-"`
	ConnectURL  string              `yaml:"connectUrl" json:"connectUrl"`   // 连接地址
	AccessToken string              `yaml:"accessToken" json:"accessToken"` // 访问令牌

	UseInPackClient bool   `yaml:"useInPackGoCqhttp" json:"useInPackGoCqhttp"` // 是否使用内置的gocqhttp
	BuiltinMode     string `yaml:"builtinMode" json:"builtinMode"`             // 分为 lagrange 和 gocq
	GoCqhttpState   int    `yaml:"-" json:"loginState"`                        // 当前状态
	CurLoginIndex   int    `yaml:"-" json:"curLoginIndex"`                     // 当前登录序号，如果正在进行的登录不是该Index，证明过时

	GoCqhttpProcess           *procs.Process `yaml:"-" json:"-"`
	GocqhttpLoginFailedReason string         `yaml:"-" json:"curLoginFailedReason"` // 当前登录失败原因

	GoCqhttpLoginCaptcha       string `yaml:"-" json:"goCqHttpLoginCaptcha"`
	GoCqhttpLoginVerifyCode    string `yaml:"-" json:"goCqHttpLoginVerifyCode"`
	GoCqhttpLoginDeviceLockURL string `yaml:"-" json:"goCqHttpLoginDeviceLockUrl"`
	GoCqhttpQrcodeData         []byte `yaml:"-" json:"-"` // 二维码数据
	GoCqhttpSmsNumberTip       string `yaml:"-" json:"goCqHttpSmsNumberTip"`

	GoCqLastAutoLoginTime      int64 `yaml:"inPackGoCqLastAutoLoginTime" json:"-"`                             // 上次自动重新登录的时间
	GoCqhttpLoginSucceeded     bool  `yaml:"inPackGoCqHttpLoginSucceeded" json:"-"`                            // 是否登录成功过
	GoCqhttpLastRestrictedTime int64 `yaml:"inPackGoCqHttpLastRestricted" json:"inPackGoCqHttpLastRestricted"` // 上次风控时间
	ForcePrintLog              bool  `yaml:"forcePrintLog" json:"forcePrintLog"`                               // 是否一定输出日志，隐藏配置项
	reconnectTimes             int   // 重连次数

	InPackGoCqhttpProtocol       int      `yaml:"inPackGoCqHttpProtocol" json:"inPackGoCqHttpProtocol"`
	InPackGoCqhttpAppVersion     string   `yaml:"inPackGoCqHttpAppVersion" json:"inPackGoCqHttpAppVersion"`
	InPackGoCqhttpPassword       string   `yaml:"inPackGoCqHttpPassword" json:"-"`
	diceServing                  bool     `yaml:"-"`                                              // 特指 diceServing 是否正在运行
	InPackGoCqhttpDisconnectedCH chan int `yaml:"-" json:"-"`                                     // 信号量，用于关闭连接
	IgnoreFriendRequest          bool     `yaml:"ignoreFriendRequest" json:"ignoreFriendRequest"` // 忽略好友请求处理开关

	customEcho     int64                                  `yaml:"-"` // 自定义返回标记
	echoMap        *syncmap.SyncMap[any, chan *MessageQQ] `yaml:"-"`
	echoMap2       *syncmap.SyncMap[any, *echoMapInfo]    `yaml:"-"`
	Implementation string                                 `yaml:"implementation" json:"implementation"`

	UseSignServer    bool              `yaml:"useSignServer" json:"useSignServer"`
	SignServerConfig *SignServerConfig `yaml:"signServerConfig" json:"signServerConfig"`
	ExtraArgs        string            `yaml:"extraArgs" json:"extraArgs"`

	riskAlertShieldCount int  // 风控警告屏蔽次数，一个临时变量
	useArrayMessage      bool `yaml:"-"` // 使用分段消息
	lagrangeRebootTimes  int
}

type Sender struct {
	Age      int32           `json:"age"`
	Card     string          `json:"card"`
	Nickname string          `json:"nickname"`
	Role     string          `json:"role"` // owner 群主
	UserID   json.RawMessage `json:"user_id"`
}

type OnebotUserInfo struct {
	// 个人信息
	Nickname string `json:"nickname"`
	UserID   string `json:"user_id"`

	// 群信息
	GroupID         string `json:"group_id"`          // 群号
	GroupCreateTime uint32 `json:"group_create_time"` // 群号
	MemberCount     int64  `json:"member_count"`
	GroupName       string `json:"group_name"`
	MaxMemberCount  int32  `json:"max_member_count"`
	Card            string `json:"card"`
}

type MessageQQBase struct {
	MessageID     int64           `json:"message_id"`   // QQ信息此类型为int64，频道中为string
	MessageType   string          `json:"message_type"` // Group
	Sender        *Sender         `json:"sender"`       // 发送者
	RawMessage    string          `json:"raw_message"`
	Time          int64           `json:"time"` // 发送时间
	MetaEventType string          `json:"meta_event_type"`
	OperatorID    json.RawMessage `json:"operator_id"`  // 操作者帐号
	GroupID       json.RawMessage `json:"group_id"`     // 群号
	PostType      string          `json:"post_type"`    // 上报类型，如group、notice
	RequestType   string          `json:"request_type"` // 请求类型，如group
	SubType       string          `json:"sub_type"`     // 子类型，如add invite
	Flag          string          `json:"flag"`         // 请求 flag, 在调用处理请求的 API 时需要传入
	NoticeType    string          `json:"notice_type"`
	UserID        json.RawMessage `json:"user_id"`
	SelfID        json.RawMessage `json:"self_id"`
	Duration      int64           `json:"duration"`
	Comment       string          `json:"comment"`
	TargetID      json.RawMessage `json:"target_id"`

	Data *struct {
		// 个人信息
		Nickname string          `json:"nickname"`
		UserID   json.RawMessage `json:"user_id"`

		// 群信息
		GroupID         json.RawMessage `json:"group_id"`          // 群号
		GroupCreateTime uint32          `json:"group_create_time"` // 群号
		MemberCount     int64           `json:"member_count"`
		GroupName       string          `json:"group_name"`
		MaxMemberCount  int32           `json:"max_member_count"`

		// 群成员信息
		Card string `json:"card"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	// Status string `json:"status"`
	Echo json.RawMessage `json:"echo"` // 声明类型而不是interface的原因是interface下数字不能正确转换

	Msg string `json:"msg"`
	// Status  interface{} `json:"status"`
	Wording string `json:"wording"`
}

type MessageQQ struct {
	MessageQQBase
	Message string `json:"message"` // 消息内容
}

// 注: 这部分对应了 onebot v11 的另一种消息格式
// https://github.com/botuniverse/onebot-11/blob/master/message/array.md

type OneBotV11MsgItemTextType struct {
	Text string `json:"text"`
}

type OneBotV11MsgItemImageType struct {
	File string `json:"file"`
}

type OneBotV11MsgItemFaceType struct {
	Id string `json:"id"`
}

type OneBotV11MsgItemRecordType struct {
	File string `json:"file"`
}

type OneBotV11MsgItemAtType struct {
	QQ string `json:"qq"`
}

type OneBotV11MsgItemPokeType struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type OneBotV11MsgItemReplyType struct {
	Id string `json:"id"`
}

type OneBotV11MsgItem struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type OneBotV11ArrMsgItem[T any] struct {
	Type string `json:"type"`
	Data T      `json:"data"`
}

type MessageQQArray struct {
	MessageQQBase
	Message []*OneBotV11MsgItem `json:"message"` // 消息内容
}

type LastWelcomeInfo struct {
	UserID  string
	GroupID string
	Time    int64
}

func (msgQQ *MessageQQ) toStdMessage() *Message {
	msg := new(Message)
	msg.Time = msgQQ.Time
	msg.MessageType = msgQQ.MessageType
	msg.Message = msgQQ.Message
	msg.Message = strings.ReplaceAll(msg.Message, "&#91;", "[")
	msg.Message = strings.ReplaceAll(msg.Message, "&#93;", "]")
	msg.Message = strings.ReplaceAll(msg.Message, "&amp;", "&")
	msg.RawID = msgQQ.MessageID
	msg.Platform = "QQ"

	if msg.MessageType == "" {
		msg.MessageType = "private"
	}

	if msgQQ.Data != nil && len(msgQQ.Data.GroupID) > 0 {
		msg.GroupID = FormatDiceIDQQGroup(string(msgQQ.Data.GroupID))
	}
	if string(msgQQ.GroupID) != "" {
		if msg.MessageType == "private" {
			msg.MessageType = "group"
		}
		msg.GroupID = FormatDiceIDQQGroup(string(msgQQ.GroupID))
	}
	if msgQQ.Sender != nil {
		msg.Sender.Nickname = msgQQ.Sender.Nickname
		if msgQQ.Sender.Card != "" {
			msg.Sender.Nickname = msgQQ.Sender.Card
		}
		msg.Sender.GroupRole = msgQQ.Sender.Role
		msg.Sender.UserID = FormatDiceIDQQ(string(msgQQ.Sender.UserID))
	}
	return msg
}

func FormatDiceIDQQ(diceQQ string) string {
	return fmt.Sprintf("QQ:%s", diceQQ)
}

func FormatDiceIDQQGroup(diceQQ string) string {
	return fmt.Sprintf("QQ-Group:%s", diceQQ)
}

func FormatDiceIDQQCh(userID string) string {
	return fmt.Sprintf("QQ-CH:%s", userID)
}

func FormatDiceIDQQChGroup(guildID, channelID string) string {
	return fmt.Sprintf("QQ-CH-Group:%s-%s", guildID, channelID)
}

func tryParseOneBot11ArrayMessage(log *zap.SugaredLogger, message string, writeTo *MessageQQ) error {
	msgQQType2 := new(MessageQQArray)
	err := json.Unmarshal([]byte(message), msgQQType2)

	if err != nil {
		log.Warn("无法解析 onebot11 字段:", message)
		return err
	}

	cqMessage := strings.Builder{}

	for _, i := range msgQQType2.Message {
		switch i.Type {
		case "text":
			cqMessage.WriteString(i.Data["text"].(string))
		case "image":
			cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", i.Data["file"]))
		case "face":
			// 兼容四叶草，移除 .(string)。自动获取的信息表示此类型为 float64，这是go解析的问题
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", i.Data["id"]))
		case "record":
			cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", i.Data["file"]))
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", i.Data["qq"]))
		case "poke":
			cqMessage.WriteString("[CQ:poke]")
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", i.Data["id"]))
		}
	}
	writeTo.MessageQQBase = msgQQType2.MessageQQBase
	writeTo.Message = cqMessage.String()
	return nil
}

func OneBot11CqMessageToArrayMessage(longText string) []interface{} {
	re := regexp.MustCompile(`\[CQ:.+?]`)
	m := re.FindAllStringIndex(longText, -1)

	newText := longText
	var arr []interface{}

	for i := len(m) - 1; i >= 0; i-- {
		p := m[i]
		cq := CQParse(longText[p[0]:p[1]])

		// 如果尾部有文本，将其拼入数组
		endText := newText[p[1]:]
		if len(endText) > 0 {
			i := OneBotV11ArrMsgItem[OneBotV11MsgItemTextType]{Type: "text", Data: OneBotV11MsgItemTextType{Text: endText}}
			arr = append(arr, i)
		}

		// 将 CQ 拼入数组
		switch cq.Type {
		case "image":
			i := OneBotV11ArrMsgItem[OneBotV11MsgItemImageType]{Type: "image", Data: OneBotV11MsgItemImageType{File: cq.Args["file"]}}
			arr = append(arr, i)
		case "record":
			i := OneBotV11ArrMsgItem[OneBotV11MsgItemRecordType]{Type: "record", Data: OneBotV11MsgItemRecordType{File: cq.Args["file"]}}
			arr = append(arr, i)
		case "at":
			// [CQ:at,qq=10001000]
			i := OneBotV11ArrMsgItem[OneBotV11MsgItemAtType]{Type: "at", Data: OneBotV11MsgItemAtType{QQ: cq.Args["qq"]}}
			arr = append(arr, i)
		default:
			data := make(map[string]interface{})
			for k, v := range cq.Args {
				data[k] = v
			}
			i := OneBotV11MsgItem{Type: cq.Type, Data: data}
			arr = append(arr, i)
		}

		newText = newText[:p[0]]
	}

	// 如果剩余有文本，将其拼入数组
	if len(newText) > 0 {
		i := OneBotV11ArrMsgItem[OneBotV11MsgItemTextType]{Type: "text", Data: OneBotV11MsgItemTextType{Text: newText}}
		arr = append(arr, i)
	}

	return lo.Reverse(arr)
}

func (pa *PlatformAdapterGocq) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterGocq) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterGocq) Serve() int {
	if pa.BuiltinMode == "lagrange" {
		pa.Implementation = "lagrange"
	} else {
		pa.Implementation = "gocq"
	}
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	dm := s.Parent.Parent
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	pa.InPackGoCqhttpDisconnectedCH = make(chan int, 1)
	session := s

	socket := gowebsocket.New(pa.ConnectURL)
	if pa.AccessToken != "" {
		socket.RequestHeader.Add("Authorization", "Bearer "+pa.AccessToken)
	}
	pa.Socket = &socket

	ep.State = 2
	socket.OnConnected = func(socket gowebsocket.Socket) {
		ep.State = 1
		if pa.IsReverse {
			log.Info("onebot v11 反向ws连接成功")
		} else {
			log.Info("onebot v11 连接成功")
		}
		pa.reconnectTimes = 0 // 重置连接重试次数
		pa.lagrangeRebootTimes = 0
		//  {"data":{"nickname":"闃斧鐗岃�佽檸鏈�","user_id":1001},"retcode":0,"status":"ok"}
		pa.GetLoginInfo()
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		// if CheckDialErr(err) != syscall.ECONNREFUSED {
		// refused 不算大事
		log.Error("onebot v11 connection error: ", err)
		log.Info("onebot wss connection addr: ", socket.Url)
		// }
		pa.InPackGoCqhttpDisconnectedCH <- 2
	}

	// {"channel_id":"3574366","guild_id":"51541481646552899","message":"说句话试试","message_id":"BAC3HLRYvXdDAAAAAAA2il4AAAAAAAAABA==","message_type":"guild","post_type":"mes
	// sage","self_id":2589922907,"self_tiny_id":"144115218748146488","sender":{"nickname":"木落","tiny_id":"222","user_id":222},"sub_type":"channel",
	// "time":1647386874,"user_id":"144115218731218202"}

	// 疑似消息发送成功？等等 是不是可以用来取一下log
	// {"data":{"message_id":-1541043078},"retcode":0,"status":"ok"}
	var lastWelcome *LastWelcomeInfo

	// 注意这几个不能轻易delete，最好整个替换
	tempInviteMap := map[string]int64{}
	tempInviteMap2 := map[string]string{}
	tempGroupEnterSpeechSent := map[string]int64{} // 记录入群致辞的发送时间 避免短时间重复
	tempFriendInviteSent := map[string]int64{}     // gocq会重新发送已经发过的邀请

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		// if strings.Contains(message, `.`) {
		//	log.Info("...", message)
		//}
		if strings.Contains(message, `"guild_id"`) {
			// log.Info("!!!", message, s.Parent.WorkInQQChannel)
			// 暂时忽略频道消息
			if s.Parent.WorkInQQChannel {
				pa.QQChannelTrySolve(message)
			}
			return
		}

		msgQQ := new(MessageQQ)
		err := json.Unmarshal([]byte(message), msgQQ)

		if err != nil {
			err = tryParseOneBot11ArrayMessage(log, message, msgQQ)

			if err != nil {
				log.Error("error" + err.Error())
				return
			}
			pa.useArrayMessage = true
		}

		// 心跳包，忽略
		if msgQQ.MetaEventType == "heartbeat" {
			return
		}
		if msgQQ.MetaEventType == "heartbeat" {
			return
		}

		if !ep.Enable {
			pa.InPackGoCqhttpDisconnectedCH <- 3
		}

		msg := msgQQ.toStdMessage()
		ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: session, Dice: session.Parent}

		if msg.Sender.UserID != "" {
			// 用户名缓存
			if msg.Sender.Nickname != "" {
				dm.UserNameCache.Set(msg.Sender.UserID, &GroupNameCacheItem{Name: msg.Sender.Nickname, time: time.Now().Unix()})
			}
		}

		// 获得用户信息
		if string(msgQQ.Echo) == "-1" {
			ep.Nickname = msgQQ.Data.Nickname
			ep.UserID = FormatDiceIDQQ(string(msgQQ.Data.UserID))

			log.Debug("骰子信息已刷新")
			ep.RefreshGroupNum()

			d := pa.Session.Parent
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
			return
		}

		// 自定义信息
		if pa.echoMap2 != nil && msgQQ.Echo != nil {
			if v, ok := pa.echoMap2.Load(string(msgQQ.Echo)); ok {
				v.ch <- message
				msgQQ.Echo = []byte(fmt.Sprintf("%v", v.echoOverwrite))
				return
			}

			now := time.Now().Unix()
			pa.echoMap2.Range(func(k any, v *echoMapInfo) bool {
				if v.timeout != 0 && now > v.timeout {
					v.ch <- ""
				}
				return true
			})
		}

		// 获得群信息
		if string(msgQQ.Echo) == "-2" { //nolint:nestif
			if msgQQ.Data != nil {
				groupID := FormatDiceIDQQGroup(string(msgQQ.Data.GroupID))
				dm.GroupNameCache.Set(groupID, &GroupNameCacheItem{
					Name: msgQQ.Data.GroupName,
					time: time.Now().Unix(),
				}) // 不论如何，先试图取一下群名

				group := session.ServiceAtNew[groupID]
				if group != nil {
					if msgQQ.Data.MaxMemberCount == 0 {
						diceID := ep.UserID
						if _, exists := group.DiceIDExistsMap.Load(diceID); exists {
							// 不在群里了，更新信息
							group.DiceIDExistsMap.Delete(diceID)
							group.UpdatedAtTime = time.Now().Unix()
						}
					} else if msgQQ.Data.GroupName != group.GroupName {
						// 更新群名
						group.GroupName = msgQQ.Data.GroupName
						group.UpdatedAtTime = time.Now().Unix()
					}

					// 处理被强制拉群的情况
					uid := group.InviteUserID
					banInfo, ok := ctx.Dice.BanList.GetByID(uid)
					if ok {
						if banInfo.Rank == BanRankBanned && ctx.Dice.BanList.BanBehaviorRefuseInvite {
							// 如果是被ban之后拉群，判定为强制拉群
							if group.EnteredTime > 0 && group.EnteredTime > banInfo.BanTime {
								text := fmt.Sprintf("本次入群为遭遇强制邀请，即将主动退群，因为邀请人%s正处于黑名单上。打扰各位还请见谅。感谢使用海豹核心。", group.InviteUserID)
								ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
								time.Sleep(1 * time.Second)
								pa.QuitGroup(ctx, groupID)
							}
							return
						}
					}

					// 强制拉群情况2 - 群在黑名单
					banInfo, ok = ctx.Dice.BanList.GetByID(groupID)
					if ok {
						if banInfo.Rank == BanRankBanned {
							// 如果是被ban之后拉群，判定为强制拉群
							if group.EnteredTime > 0 && group.EnteredTime > banInfo.BanTime {
								text := fmt.Sprintf("被群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", banInfo.toText(ctx.Dice))
								ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
								time.Sleep(1 * time.Second)
								pa.QuitGroup(ctx, groupID)
							}
							return
						}
					}
				} else {
					// TODO: 这玩意的创建是个专业活，等下来弄
					// session.ServiceAtNew[groupId] = GroupInfo{}
					fmt.Println("TODO create group")
				}
				// 这句话太吵了
				// log.Debug("群信息刷新: ", msgQQ.Data.GroupName)
			}
			return
		}

		// 自定义信息
		if pa.echoMap != nil && msgQQ.Echo != nil {
			if v, ok := pa.echoMap.Load(string(msgQQ.Echo)); ok {
				v <- msgQQ
				return
			}
		}

		// 处理加群请求
		if msgQQ.PostType == "request" && msgQQ.RequestType == "group" && msgQQ.SubType == "invite" {
			// {"comment":"","flag":"111","group_id":222,"post_type":"request","request_type":"group","self_id":333,"sub_type":"invite","time":1646782195,"user_id":444}
			ep.RefreshGroupNum()
			pa.GetGroupInfoAsync(msg.GroupID)
			time.Sleep(time.Duration((1.8 + rand.Float64()) * float64(time.Second))) // 稍作等待，也许能拿到群名

			uid := FormatDiceIDQQ(string(msgQQ.UserID))
			groupName := dm.TryGetGroupName(msg.GroupID)
			userName := dm.TryGetUserName(uid)
			txt := fmt.Sprintf("收到QQ加群邀请: 群组<%s>(%s) 邀请人:<%s>(%s)", groupName, msgQQ.GroupID, userName, msgQQ.UserID)
			log.Info(txt)
			ctx.Notice(txt)
			tempInviteMap[msg.GroupID] = time.Now().Unix()
			tempInviteMap2[msg.GroupID] = uid

			// 邀请人在黑名单上
			banInfo, ok := ctx.Dice.BanList.GetByID(uid)
			if ok {
				if banInfo.Rank == BanRankBanned && ctx.Dice.BanList.BanBehaviorRefuseInvite {
					pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, false, "黑名单")
					return
				}
			}

			// 信任模式，如果不是信任，又不是master则拒绝拉群邀请
			isMaster := ctx.Dice.IsMaster(uid)
			if ctx.Dice.TrustOnlyMode && ((banInfo != nil && banInfo.Rank != BanRankTrusted) && !isMaster) {
				pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, false, "只允许骰主设置信任的人拉群")
				return
			}

			// 群在黑名单上
			banInfo, ok = ctx.Dice.BanList.GetByID(msg.GroupID)
			if ok {
				if banInfo.Rank == BanRankBanned {
					pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, false, "群黑名单")
					return
				}
			}

			if ctx.Dice.RefuseGroupInvite {
				pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, false, "设置拒绝加群")
				return
			}

			// time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))
			pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, true, "")
			return
		}

		// 好友请求
		if msgQQ.PostType == "request" && msgQQ.RequestType == "friend" { //nolint:nestif
			// 有一个来自gocq的重发问题
			lastTime := tempFriendInviteSent[msgQQ.Flag]
			nowTime := time.Now().Unix()
			if nowTime-lastTime < 20*60 {
				// 保留20s
				return
			}
			tempFriendInviteSent[msgQQ.Flag] = nowTime

			// {"comment":"123","flag":"1647619872000000","post_type":"request","request_type":"friend","self_id":222,"time":1647619871,"user_id":111}
			var comment string
			if msgQQ.Comment != "" {
				comment = strings.TrimSpace(msgQQ.Comment)
				comment = strings.ReplaceAll(comment, "\u00a0", "")
			}

			toMatch := strings.TrimSpace(session.Parent.FriendAddComment)
			willAccept := comment == DiceFormat(ctx, toMatch)
			if toMatch == "" {
				willAccept = true
			}

			if !willAccept {
				// 如果是问题校验，只填写回答即可
				re := regexp.MustCompile(`\n回答:([^\n]+)`)
				m := re.FindAllStringSubmatch(comment, -1)

				var items []string
				for _, i := range m {
					items = append(items, i[1])
				}

				re2 := regexp.MustCompile(`\s+`)
				m2 := re2.Split(toMatch, -1)

				if len(m2) == len(items) {
					ok := true
					for i := 0; i < len(m2); i++ {
						if m2[i] != items[i] {
							ok = false
							break
						}
					}
					willAccept = ok
				}
			}

			if comment == "" {
				comment = "(无)"
			} else {
				comment = strconv.Quote(comment)
			}

			// 检查黑名单
			extra := ""
			uid := FormatDiceIDQQ(string(msgQQ.UserID))
			banInfo, ok := ctx.Dice.BanList.GetByID(uid)
			if ok {
				if banInfo.Rank == BanRankBanned && ctx.Dice.BanList.BanBehaviorRefuseInvite {
					if willAccept {
						extra = "。回答正确，但为被禁止用户，准备自动拒绝"
					} else {
						extra = "。回答错误，且为被禁止用户，准备自动拒绝"
					}
					willAccept = false
				}
			}

			if pa.IgnoreFriendRequest {
				extra += "。由于设置了忽略邀请，此信息仅为通报"
			}

			txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", msgQQ.UserID, comment, willAccept, extra)
			log.Info(txt)
			ctx.Notice(txt)

			// 忽略邀请
			if pa.IgnoreFriendRequest {
				return
			}

			time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

			if willAccept {
				pa.SetFriendAddRequest(msgQQ.Flag, true, "", "")
			} else {
				pa.SetFriendAddRequest(msgQQ.Flag, false, "", "验证信息不符")
			}
			return
		}

		// 好友通过后
		if msgQQ.NoticeType == "friend_add" && msgQQ.PostType == "notice" {
			// {"notice_type":"friend_add","post_type":"notice","self_id":222,"time":1648239248,"user_id":111}
			// 拉格兰目前会发送错误的记录，记录如下：
			// {"user_id":aaa,"notice_type":"friend_add","time":1708080896,"self_id":aaa,"post_type":"notice"}
			// 即用户id和self_id是同一个id的情况，这就会导致错误发送消息。
			if string(msgQQ.UserID) == string(msgQQ.SelfID) {
				return
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("好友致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
					}
				}()

				// 稍作等待后发好友致辞
				time.Sleep(2 * time.Second)

				msg.Sender.UserID = FormatDiceIDQQ(string(msgQQ.UserID))
				// 似乎这样会阻塞住，先不搞了
				// d := pa.GetStrangerInfo(msgQQ.UserId) // 先获取个人信息，避免不存在id
				ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
				// if ctx.Player.Name == "" {
				//	ctx.Player.Name = d.Nickname
				//	ctx.Player.UpdatedAtTime = time.Now().Unix()
				//}

				uid := FormatDiceIDQQ(string(msgQQ.UserID))

				welcome := DiceFormatTmpl(ctx, "核心:骰子成为好友")
				log.Infof("与 %s 成为好友，发送好友致辞: %s", uid, welcome)

				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Errorf("好友致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
						}
					}()

					// 这是一个polyfill，因为目前版本的lagrange会先发送friend_add事件，后成为好友
					// 而不是成为好友后，再发送friend_add事件(go-cqhttp行为)，导致好友致辞发不出去
					// 因此略作延迟，等上游修复后可以移除
					time.Sleep(5 * time.Second)

					for _, i := range ctx.SplitText(welcome) {
						doSleepQQ(ctx)
						pa.SendToPerson(ctx, uid, strings.TrimSpace(i), "")
					}
					if ctx.Session.ServiceAtNew[msg.GroupID] != nil {
						for _, i := range ctx.Session.ServiceAtNew[msg.GroupID].ActivatedExtList {
							if i.OnBecomeFriend != nil {
								i.callWithJsCheck(ctx.Dice, func() {
									i.OnBecomeFriend(ctx, msg)
								})
							}
						}
					}
				}()
			}()
			return
		}

		groupEnterFired := false
		groupEntered := func() {
			if groupEnterFired {
				return
			}
			groupEnterFired = true
			lastTime := tempGroupEnterSpeechSent[msg.GroupID]
			nowTime := time.Now().Unix()

			if nowTime-lastTime < 10 {
				// 10s内只发一次
				return
			}
			tempGroupEnterSpeechSent[msg.GroupID] = nowTime

			// 判断进群的人是自己，自动启动
			gi := SetBotOnAtGroup(ctx, msg.GroupID)
			// 获取邀请人ID
			if tempInviteMap2[msg.GroupID] != "" {
				// 设置邀请人
				gi.InviteUserID = tempInviteMap2[msg.GroupID]
			} else if string(msgQQ.OperatorID) != "" {
				// 适用场景: 受邀加入无需审核的群时邀请人显示未知的问题 (#710) - llob
				gi.InviteUserID = FormatDiceIDQQ(string(msgQQ.OperatorID))
			}
			gi.DiceIDExistsMap.Store(ep.UserID, true)
			gi.EnteredTime = nowTime // 设置入群时间
			gi.UpdatedAtTime = time.Now().Unix()
			// 立即获取群信息
			pa.GetGroupInfoAsync(msg.GroupID)
			// fmt.Sprintf("<%s>已经就绪。可通过.help查看指令列表", conn.Nickname)

			time.Sleep(2 * time.Second)
			groupName := dm.TryGetGroupName(msg.GroupID)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("入群致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
					}
				}()

				// 稍作等待后发送入群致词
				time.Sleep(1 * time.Second)

				ctx.Player = &GroupPlayerInfo{}
				log.Infof("发送入群致辞，群: <%s>(%d)", groupName, msgQQ.GroupID)
				text := DiceFormatTmpl(ctx, "核心:骰子进群")
				for _, i := range ctx.SplitText(text) {
					doSleepQQ(ctx)
					pa.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), "")
				}
			}()
			txt := fmt.Sprintf("加入QQ群组: <%s>(%s)", groupName, msgQQ.GroupID)
			log.Info(txt)
			ctx.Notice(txt)
			if ctx.Session.ServiceAtNew[msg.GroupID] != nil {
				for _, i := range ctx.Session.ServiceAtNew[msg.GroupID].ActivatedExtList {
					if i.OnGroupJoined != nil {
						i.callWithJsCheck(ctx.Dice, func() {
							i.OnGroupJoined(ctx, msg)
						})
					}
				}
			}
		}

		// 入群的另一种情况: 管理员审核
		group := s.ServiceAtNew[msg.GroupID]
		if group == nil && msg.GroupID != "" {
			now := time.Now().Unix()
			if tempInviteMap[msg.GroupID] != 0 && now > tempInviteMap[msg.GroupID] {
				delete(tempInviteMap, msg.GroupID)
				groupEntered()
			}
			// log.Infof("自动激活: 发现无记录群组(%s)，因为已是群成员，所以自动激活", group.GroupId)
		}

		// 入群后自动开启
		if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_increase" {
			// {"group_id":111,"notice_type":"group_increase","operator_id":0,"post_type":"notice","self_id":333,"sub_type":"approve","time":1646782012,"user_id":333}
			if string(msgQQ.UserID) == string(msgQQ.SelfID) {
				groupEntered()
			} else {
				group := session.ServiceAtNew[msg.GroupID]
				// 进群的是别人，是否迎新？
				// 这里很诡异，当手机QQ客户端审批进群时，入群后会有一句默认发言
				// 此时会收到两次完全一样的某用户入群信息，导致发两次欢迎词
				if group != nil && group.ShowGroupWelcome {
					isDouble := false
					if lastWelcome != nil {
						isDouble = string(msgQQ.GroupID) == lastWelcome.GroupID &&
							string(msgQQ.UserID) == lastWelcome.UserID &&
							msgQQ.Time == lastWelcome.Time
					}
					lastWelcome = &LastWelcomeInfo{
						GroupID: string(msgQQ.GroupID),
						UserID:  string(msgQQ.UserID),
						Time:    msgQQ.Time,
					}

					if !isDouble {
						func() {
							defer func() {
								if r := recover(); r != nil {
									log.Errorf("迎新致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
								}
							}()

							ctx.Player = &GroupPlayerInfo{}
							// VarSetValueStr(ctx, "$t新人昵称", "<"+msgQQ.Sender.Nickname+">")
							uidRaw := string(msgQQ.UserID)
							VarSetValueStr(ctx, "$t帐号ID_RAW", uidRaw)
							VarSetValueStr(ctx, "$t账号ID_RAW", uidRaw)
							stdID := FormatDiceIDQQ(string(msgQQ.UserID))
							VarSetValueStr(ctx, "$t帐号ID", stdID)
							VarSetValueStr(ctx, "$t账号ID", stdID)
							text := DiceFormat(ctx, group.GroupWelcomeMessage)
							for _, i := range ctx.SplitText(text) {
								doSleepQQ(ctx)
								pa.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), "")
							}
						}()
					}
				}
			}
			return
		}

		if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_decrease" && msgQQ.SubType == "kick_me" {
			// 被踢
			//  {"group_id":111,"notice_type":"group_decrease","operator_id":222,"post_type":"notice","self_id":333,"sub_type":"kick_me","time":1646689414 ,"user_id":333}
			if string(msgQQ.UserID) == string(msgQQ.SelfID) {
				opUID := FormatDiceIDQQ(string(msgQQ.OperatorID))
				groupName := dm.TryGetGroupName(msg.GroupID)
				userName := dm.TryGetUserName(opUID)

				skip := false
				skipReason := ""
				banInfo, ok := ctx.Dice.BanList.GetByID(opUID)
				if ok {
					if banInfo.Rank == 30 {
						skip = true
						skipReason = "信任用户"
					}
				}
				if ctx.Dice.IsMaster(opUID) {
					skip = true
					skipReason = "Master"
				}

				var extra string
				if skip {
					extra = fmt.Sprintf("\n取消处罚，原因为%s", skipReason)
				} else {
					ctx.Dice.BanList.AddScoreByGroupKicked(opUID, msg.GroupID, ctx)
				}

				txt := fmt.Sprintf("被踢出群: 在QQ群组<%s>(%s)中被踢出，操作者:<%s>(%s)%s", groupName, msgQQ.GroupID, userName, msgQQ.OperatorID, extra)
				log.Info(txt)
				ctx.Notice(txt)
			}
			return
		}

		if msgQQ.PostType == "notice" &&
			msgQQ.NoticeType == "group_decrease" &&
			msgQQ.SubType == "leave" &&
			string(msgQQ.OperatorID) == string(msgQQ.SelfID) {
			// 群解散
			// {"group_id":564808710,"notice_type":"group_decrease","operator_id":2589922907,"post_type":"notice","self_id":2589922907,"sub_type":"leave","time":1651584460,"user_id":2589922907}
			groupName := dm.TryGetGroupName(msg.GroupID)
			txt := fmt.Sprintf("离开群组或群解散: <%s>(%s)", groupName, msgQQ.GroupID)
			log.Info(txt)
			ctx.Notice(txt)
			return
		}

		if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_ban" && msgQQ.SubType == "ban" {
			// 禁言
			// {"duration":600,"group_id":111,"notice_type":"group_ban","operator_id":222,"post_type":"notice","self_id":333,"sub_type":"ban","time":1646689567,"user_id":333}
			if string(msgQQ.UserID) == string(msgQQ.SelfID) {
				opUID := FormatDiceIDQQ(string(msgQQ.OperatorID))
				groupName := dm.TryGetGroupName(msg.GroupID)
				userName := dm.TryGetUserName(opUID)

				ctx.Dice.BanList.AddScoreByGroupMuted(opUID, msg.GroupID, ctx)
				txt := fmt.Sprintf("被禁言: 在群组<%s>(%s)中被禁言，时长%d秒，操作者:<%s>(%s)", groupName, msgQQ.GroupID, msgQQ.Duration, userName, msgQQ.OperatorID)
				log.Info(txt)
				ctx.Notice(txt)
			}
			return
		}

		// 消息撤回
		if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_recall" {
			s.OnMessageDeleted(ctx, msg)
			return
		}

		if msgQQ.PostType == "" && msgQQ.Msg == "SEND_MSG_API_ERROR" && msgQQ.Retcode == 100 {
			// 群消息发送失败: 账号可能被风控，戳对方一下
			// {"data":null,"echo":0,"msg":"SEND_MSG_API_ERROR","retcode":100,"status":"failed","wording":"请参考 go-cqhttp 端输出"}
			// 但是这里没QQ号也没有消息ID，很麻烦
			if pa.riskAlertShieldCount > 0 {
				pa.riskAlertShieldCount--
			} else {
				fmt.Println("群消息发送失败: 账号可能被风控")
				_ = ctx.Dice.SendMail("群消息发送失败: 账号可能被风控", MailTypeCIAMLock)
			}
		}

		// 戳一戳
		if msgQQ.PostType == "notice" && msgQQ.SubType == "poke" {
			// {"post_type":"notice","notice_type":"notify","time":1672489767,"self_id":2589922907,"sub_type":"poke","group_id":131687852,"user_id":303451945,"sender_id":303451945,"target_id":2589922907}

			// 检查设置中是否开启
			if !ctx.Dice.QQEnablePoke {
				return
			}

			go func() {
				defer ErrorLogAndContinue(pa.Session.Parent)
				ctx := pa.packTempCtx(msgQQ, msg)

				if string(msgQQ.TargetID) == string(msgQQ.SelfID) {
					// 如果在戳自己
					text := DiceFormatTmpl(ctx, "其它:戳一戳")
					for _, i := range ctx.SplitText(text) {
						doSleepQQ(ctx)
						switch msg.MessageType {
						case "group":
							pa.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), "")
						case "private":
							pa.SendToPerson(ctx, msg.Sender.UserID, strings.TrimSpace(i), "")
						}
					}
				}
			}()
			return
		}

		// 处理命令
		if msgQQ.MessageType == "group" || msgQQ.MessageType == "private" {
			if msg.Sender.UserID == ep.UserID {
				// 以免私聊时自己发的信息被记录
				// 这里的私聊消息可能是自己发送的
				// 要是群发也可以被记录就好了
				// XXXX {"font":0,"message":"\u003c木落\u003e的今日人品为83","message_id":-358748624,"message_type":"private","post_type":"message_sent","raw_message":"\u003c木落\u003e的今日人
				// 品为83","self_id":2589922907,"sender":{"age":0,"nickname":"海豹一号机","sex":"unknown","user_id":2589922907},"sub_type":"friend","target_id":222,"time":1647760835,"use
				// r_id":2589922907}
				return
			}
			// fmt.Println("Recieved message1 " + message)
			// 拉格朗日无Sender缓存，采用最低修改的方式修补对接以解决私聊无法回复的问题
			if msgQQ.MessageType == "private" {
				if msg.Sender.UserID == "QQ:" {
					msg.Sender.UserID = "QQ:" + string(msgQQ.UserID)
				}
				if msg.Sender.Nickname == "" {
					msg.Sender.Nickname = "未知用户"
				}
			}
			session.Execute(ep, msg, false)
		} else {
			fmt.Println("Received message " + message)
		}
	}

	socket.OnBinaryMessage = func(data []byte, socket gowebsocket.Socket) {
		log.Debug("Recieved binary data ", data)
	}

	socket.OnPingReceived = func(data string, socket gowebsocket.Socket) {
		log.Debug("Recieved ping " + data)
	}

	socket.OnPongReceived = func(data string, socket gowebsocket.Socket) {
		log.Debug("Recieved pong " + data)
	}

	var lastDisconnect int64
	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		now := time.Now().Unix()
		if now-lastDisconnect < 2 {
			// 存在极端时间内触发两次的情况，且为同一个连接
			// 其他行为都是正常的，原因不明
			return
		}
		lastDisconnect = now

		log.Info("onebot 服务的连接被对方关闭")
		_ = pa.Session.Parent.SendMail("", MailTypeConnectClose)
		pa.InPackGoCqhttpDisconnectedCH <- 1
	}

	if pa.IsReverse {
		go func() {
			if pa.IsReverse && pa.reverseApp != nil {
				_ = pa.reverseApp.Close()
				pa.reverseApp = nil
			}

			e := echo.New()

			upgrader := websocket.Upgrader{}
			e.GET("/ws", func(c echo.Context) error {
				ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
				if err != nil {
					return err
				}
				defer ws.Close()

				socketClone := gowebsocket.New("")
				socketClone.OnDisconnected = socket.OnDisconnected
				socketClone.OnTextMessage = socket.OnTextMessage
				socketClone.OnBinaryMessage = socket.OnBinaryMessage
				socketClone.OnPingReceived = socket.OnPingReceived
				socketClone.OnPongReceived = socket.OnPongReceived
				socketClone.OnConnected = socket.OnConnected
				socketClone.OnConnectError = socket.OnConnectError
				// 注: 只能管一个socket，不过不管了
				pa.Socket = &socketClone

				pa.EndPoint.State = 1
				socketClone.NewClient(ws)
				return nil
			})

			pa.reverseApp = e
			log.Info("Onebot v11 反向WS服务启动，地址: ", pa.ReverseAddr)
			e.HideBanner = true
			err := e.Start(pa.ReverseAddr)
			if err != nil {
				log.Error("Onebot v11 反向WS服务关闭: ", err)
				pa.diceServing = false
			}
		}()
	} else {
		socket.Connect()
	}

	defer func() {
		// fmt.Println("socket close")
		go func() {
			defer func() {
				if r := recover(); r != nil { //nolint
					// 太频繁了 不输出了
					// fmt.Println("关闭连接时遭遇异常")
					// core.GetLogger().Error(r)
				}
			}()

			// 可能耗时好久
			socket.Close()
		}()
	}()

	for {
		select {
		case <-interrupt:
			log.Info("interrupt")
			pa.InPackGoCqhttpDisconnectedCH <- 0
			return 0
		case val := <-pa.InPackGoCqhttpDisconnectedCH:
			return val
		}
	}
}

func (pa *PlatformAdapterGocq) DoRelogin() bool {
	myDice := pa.Session.Parent
	ep := pa.EndPoint
	if pa.Socket != nil {
		go func() {
			defer func() {
				_ = recover()
			}()
			pa.Socket.Close()
			pa.diceServing = false
		}()
	}

	if pa.IsReverse {
		if pa.reverseApp != nil {
			_ = pa.reverseApp.Close()
			pa.reverseApp = nil
		}

		go pa.Serve()
		return true
	}

	if pa.UseInPackClient {
		if pa.InPackGoCqhttpDisconnectedCH != nil {
			pa.InPackGoCqhttpDisconnectedCH <- -1
		}
		if pa.BuiltinMode == "lagrange" {
			myDice.Logger.Infof("重新启动 lagrange 进程，对应账号: <%s>(%s)", ep.Nickname, ep.UserID)
			pa.CurLoginIndex++
			pa.GoCqhttpState = StateCodeInit
			ep.Enable = false // 拉格朗进程杀死前应先禁用账号，否则拉格朗会自动重启（该行为在LagrangeServe中）
			go BuiltinQQServeProcessKill(myDice, ep)
			time.Sleep(10 * time.Second)           // 上面那个清理有概率卡住，具体不懂，改成等5s -> 10s 超过一次重试间隔
			LagrangeServeRemoveSession(myDice, ep) // 删除 keystore
			pa.GoCqhttpLastRestrictedTime = 0      // 重置风控时间
			ep.Enable = true
			myDice.LastUpdatedTime = time.Now().Unix()
			myDice.Save(false)
			LagrangeServe(myDice, ep, LagrangeLoginInfo{
				IsAsyncRun: true,
			})
			return true
		} else {
			myDice.Logger.Infof("重新启动go-cqhttp进程，对应账号: <%s>(%s)", ep.Nickname, ep.UserID)
			pa.CurLoginIndex++
			pa.GoCqhttpState = StateCodeInit
			go BuiltinQQServeProcessKill(myDice, ep)
			time.Sleep(10 * time.Second)                // 上面那个清理有概率卡住，具体不懂，改成等5s -> 10s 超过一次重试间隔
			GoCqhttpServeRemoveSessionToken(myDice, ep) // 删除session.token
			pa.GoCqhttpLastRestrictedTime = 0           // 重置风控时间
			myDice.LastUpdatedTime = time.Now().Unix()
			myDice.Save(false)
			GoCqhttpServe(myDice, ep, GoCqhttpLoginInfo{
				Password:         pa.InPackGoCqhttpPassword,
				Protocol:         pa.InPackGoCqhttpProtocol,
				AppVersion:       pa.InPackGoCqhttpAppVersion,
				IsAsyncRun:       true,
				UseSignServer:    pa.UseSignServer,
				SignServerConfig: pa.SignServerConfig,
			})
			return true
		}
	}
	return false
}

func (pa *PlatformAdapterGocq) SetEnable(enable bool) {
	d := pa.Session.Parent
	c := pa.EndPoint
	if enable {
		c.Enable = true

		if pa.UseInPackClient {
			if pa.BuiltinMode == "lagrange" {
				BuiltinQQServeProcessKill(d, c)
				time.Sleep(1 * time.Second)
				LagrangeServe(d, c, LagrangeLoginInfo{
					IsAsyncRun: true,
				})
			} else {
				BuiltinQQServeProcessKill(d, c)
				time.Sleep(1 * time.Second)
				GoCqhttpServe(d, c, GoCqhttpLoginInfo{
					Password:         pa.InPackGoCqhttpPassword,
					Protocol:         pa.InPackGoCqhttpProtocol,
					AppVersion:       pa.InPackGoCqhttpAppVersion,
					IsAsyncRun:       true,
					UseSignServer:    pa.UseSignServer,
					SignServerConfig: pa.SignServerConfig,
				})
				go ServeQQ(d, c)
			}
		} else {
			pa.GoCqhttpState = StateCodeLoginSuccessed
			go ServeQQ(d, c)
		}
	} else {
		c.Enable = false
		if pa.UseInPackClient {
			BuiltinQQServeProcessKill(d, c)
		}
		if pa.IsReverse && pa.reverseApp != nil {
			_ = pa.reverseApp.Close()
			pa.reverseApp = nil
		}
	}

	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

func (pa *PlatformAdapterGocq) SetQQProtocol(protocol int) bool {
	// oldProtocol := pa.InPackGoCqHttpProtocol
	pa.InPackGoCqhttpProtocol = protocol

	// ep.Session.Parent.GetDiceDataPath(ep.RelWorkDir)
	workDir := filepath.Join(pa.Session.Parent.BaseConfig.DataDir, pa.EndPoint.RelWorkDir)
	deviceFilePath := filepath.Join(workDir, "device.json")
	if _, err := os.Stat(deviceFilePath); err == nil {
		configFile, _ := os.ReadFile(deviceFilePath)
		info := map[string]interface{}{}
		err = json.Unmarshal(configFile, &info)

		if err == nil {
			info["protocol"] = protocol
			data, err := json.Marshal(info)
			if err == nil {
				_ = os.WriteFile(deviceFilePath, data, 0644)
				return true
			}
		}
	}
	return false
}

func (pa *PlatformAdapterGocq) SetSignServer(signServerConfig *SignServerConfig) bool {
	workDir := filepath.Join(pa.Session.Parent.BaseConfig.DataDir, pa.EndPoint.RelWorkDir)
	configFilePath := filepath.Join(workDir, "config.yml")
	if _, err := os.Stat(configFilePath); err == nil {
		configFile, _ := os.ReadFile(configFilePath)
		info := map[string]interface{}{}
		err = yaml.Unmarshal(configFile, &info)

		if err == nil {
			if signServerConfig.SignServers != nil {
				mainServer := signServerConfig.SignServers[0]
				(info["account"]).(map[string]interface{})["sign-server"] = mainServer.URL
				(info["account"]).(map[string]interface{})["key"] = mainServer.Key
				(info["account"]).(map[string]interface{})["sign-servers"] = signServerConfig.SignServers
				(info["account"]).(map[string]interface{})["ruleChangeSignServer"] = signServerConfig.RuleChangeSignServer
				(info["account"]).(map[string]interface{})["maxCheckCount"] = signServerConfig.MaxCheckCount
				(info["account"]).(map[string]interface{})["signServerTimeout"] = signServerConfig.SignServerTimeout
				(info["account"]).(map[string]interface{})["autoRegister"] = signServerConfig.AutoRegister
				(info["account"]).(map[string]interface{})["autoRefreshToken"] = signServerConfig.AutoRefreshToken
				(info["account"]).(map[string]interface{})["refreshInterval"] = signServerConfig.RefreshInterval
			}
			data, err := yaml.Marshal(info)
			if err == nil {
				_ = os.WriteFile(configFilePath, data, 0644)
				return true
			}
		}
	}
	return false
}

func (pa *PlatformAdapterGocq) IsInLogin() bool {
	return pa.GoCqhttpState < StateCodeLoginSuccessed
}

func (pa *PlatformAdapterGocq) IsLoginSuccessed() bool {
	return pa.GoCqhttpState == StateCodeLoginSuccessed
}

func (pa *PlatformAdapterGocq) packTempCtx(msgQQ *MessageQQ, msg *Message) *MsgContext {
	ep := pa.EndPoint
	session := pa.Session

	ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: session, Dice: session.Parent}

	switch msg.MessageType {
	case "private":
		d := pa.GetStrangerInfo(string(msgQQ.UserID)) // 先获取个人信息，避免不存在id
		msg.Sender.UserID = FormatDiceIDQQ(string(msgQQ.UserID))
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		if ctx.Player.Name == "" {
			ctx.Player.Name = d.Nickname
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, ctx.Player.Name)
	case "group":
		d := pa.GetGroupMemberInfo(string(msgQQ.GroupID), string(msgQQ.UserID)) // 先获取个人信息，避免不存在id
		msg.Sender.UserID = FormatDiceIDQQ(string(msgQQ.UserID))
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		if ctx.Player.Name == "" {
			ctx.Player.Name = d.Card
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, ctx.Player.Name)
	}

	return ctx
}
