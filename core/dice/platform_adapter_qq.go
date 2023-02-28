package dice

import (
	"encoding/json"
	"fmt"
	"github.com/fy0/procs"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sealdice-core/dice/model"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// 0 默认 1登录中 2登录中-二维码 3登录中-滑条 4登录中-手机验证码 10登录成功 11登录失败

const (
	GoCqHttpStateCodeInit              = 0
	GoCqHttpStateCodeInLogin           = 1
	GoCqHttpStateCodeInLoginQrCode     = 2
	GoCqHttpStateCodeInLoginBar        = 3
	GoCqHttpStateCodeInLoginVerifyCode = 6
	GoCqHttpStateCodeInLoginDeviceLock = 7
	GoCqHttpStateCodeLoginSuccessed    = 10
	GoCqHttpStateCodeLoginFailed       = 11
	GoCqHttpStateCodeClosed            = 20
)

type echoMapInfo struct {
	ch            chan string
	echoOverwrite int64
	timeout       int64
}

type PlatformAdapterQQOnebot struct {
	EndPoint *EndPointInfo `yaml:"-" json:"-"`
	Session  *IMSession    `yaml:"-" json:"-"`

	Socket     *gowebsocket.Socket `yaml:"-" json:"-"`
	ConnectUrl string              `yaml:"connectUrl" json:"connectUrl"` // 连接地址

	UseInPackGoCqhttp bool `yaml:"useInPackGoCqhttp" json:"useInPackGoCqhttp"` // 是否使用内置的gocqhttp
	GoCqHttpState     int  `yaml:"-" json:"goCqHttpState"`                     // 当前状态
	CurLoginIndex     int  `yaml:"-" json:"curLoginIndex"`                     // 当前登录序号，如果正在进行的登录不是该Index，证明过时

	GoCqHttpProcess           *procs.Process `yaml:"-" json:"-"`
	GocqhttpLoginFailedReason string         `yaml:"-" json:"curLoginFailedReason"` // 当前登录失败原因

	GoCqHttpLoginVerifyCode    string `yaml:"-" json:"goCqHttpLoginVerifyCode"`
	GoCqHttpLoginDeviceLockUrl string `yaml:"-" json:"goCqHttpLoginDeviceLockUrl"`
	GoCqHttpQrcodeData         []byte `yaml:"-" json:"-"` // 二维码数据

	GoCqLastAutoLoginTime      int64 `yaml:"inPackGoCqLastAutoLoginTime" json:"-"`                             // 上次自动重新登录的时间
	GoCqHttpLoginSucceeded     bool  `yaml:"inPackGoCqHttpLoginSucceeded" json:"-"`                            // 是否登录成功过
	GoCqHttpLastRestrictedTime int64 `yaml:"inPackGoCqHttpLastRestricted" json:"inPackGoCqHttpLastRestricted"` // 上次风控时间

	InPackGoCqHttpProtocol       int      `yaml:"inPackGoCqHttpProtocol" json:"inPackGoCqHttpProtocol"`
	InPackGoCqHttpPassword       string   `yaml:"inPackGoCqHttpPassword" json:"-"`
	DiceServing                  bool     `yaml:"-"`                                              // 是否正在连接中
	InPackGoCqHttpDisconnectedCH chan int `yaml:"-" json:"-"`                                     // 信号量，用于关闭连接
	IgnoreFriendRequest          bool     `yaml:"ignoreFriendRequest" json:"ignoreFriendRequest"` // 忽略好友请求处理开关

	customEcho int64                            `yaml:"-"` // 自定义返回标记
	echoMap    *SyncMap[int64, chan *MessageQQ] `yaml:"-"`
	echoMap2   *SyncMap[int64, *echoMapInfo]    `yaml:"-"`
}

type Sender struct {
	Age      int32  `json:"age"`
	Card     string `json:"card"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"` // owner 群主
	UserId   int64  `json:"user_id"`
}

type OnebotUserInfo struct {
	// 个人信息
	Nickname string `json:"nickname"`
	UserId   int64  `json:"user_id"`

	// 群信息
	GroupId         int64  `json:"group_id"`          // 群号
	GroupCreateTime uint32 `json:"group_create_time"` // 群号
	MemberCount     int64  `json:"member_count"`
	GroupName       string `json:"group_name"`
	MaxMemberCount  int32  `json:"max_member_count"`
	Card            string `json:"card"`
}

type MessageQQ struct {
	MessageId     int64   `json:"message_id"`   // QQ信息此类型为int64，频道中为string
	MessageType   string  `json:"message_type"` // Group
	Sender        *Sender `json:"sender"`       // 发送者
	RawMessage    string  `json:"raw_message"`
	Message       string  `json:"message"` // 消息内容
	Time          int64   `json:"time"`    // 发送时间
	MetaEventType string  `json:"meta_event_type"`
	OperatorId    int64   `json:"operator_id"`  // 操作者帐号
	GroupId       int64   `json:"group_id"`     // 群号
	PostType      string  `json:"post_type"`    // 上报类型，如group、notice
	RequestType   string  `json:"request_type"` // 请求类型，如group
	SubType       string  `json:"sub_type"`     // 子类型，如add invite
	Flag          string  `json:"flag"`         // 请求 flag, 在调用处理请求的 API 时需要传入
	NoticeType    string  `json:"notice_type"`
	UserId        int64   `json:"user_id"`
	SelfId        int64   `json:"self_id"`
	Duration      int64   `json:"duration"`
	Comment       string  `json:"comment"`
	TargetId      int64   `json:"target_id"`

	Data *struct {
		// 个人信息
		Nickname string `json:"nickname"`
		UserId   int64  `json:"user_id"`

		// 群信息
		GroupId         int64  `json:"group_id"`          // 群号
		GroupCreateTime uint32 `json:"group_create_time"` // 群号
		MemberCount     int64  `json:"member_count"`
		GroupName       string `json:"group_name"`
		MaxMemberCount  int32  `json:"max_member_count"`

		// 群成员信息
		Card string `json:"card"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	//Status string `json:"status"`
	Echo int64 `json:"echo"` //声明类型而不是interface的原因是interface下数字不能正确转换

	Msg string `json:"msg"`
	//Status  interface{} `json:"status"`
	Wording string `json:"wording"`
}

type LastWelcomeInfo struct {
	UserId  int64
	GroupId int64
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
	msg.RawId = msgQQ.MessageId
	msg.Platform = "QQ"

	if msg.MessageType == "" {
		msg.MessageType = "private"
	}

	if msgQQ.Data != nil && msgQQ.Data.GroupId != 0 {
		msg.GroupId = FormatDiceIdQQGroup(msgQQ.Data.GroupId)
	}
	if msgQQ.GroupId != 0 {
		if msg.MessageType == "private" {
			msg.MessageType = "group"
		}
		msg.GroupId = FormatDiceIdQQGroup(msgQQ.GroupId)
	}
	if msgQQ.Sender != nil {
		msg.Sender.Nickname = msgQQ.Sender.Nickname
		if msgQQ.Sender.Card != "" {
			msg.Sender.Nickname = msgQQ.Sender.Card
		}
		msg.Sender.GroupRole = msgQQ.Sender.Role
		msg.Sender.UserId = FormatDiceIdQQ(msgQQ.Sender.UserId)
	}
	return msg
}

func FormatDiceIdQQ(diceQQ int64) string {
	return fmt.Sprintf("QQ:%s", strconv.FormatInt(diceQQ, 10))
}

func FormatDiceIdQQGroup(diceQQ int64) string {
	return fmt.Sprintf("QQ-Group:%s", strconv.FormatInt(diceQQ, 10))
}

func FormatDiceIdQQCh(userId string) string {
	return fmt.Sprintf("QQ-CH:%s", userId)
}

func FormatDiceIdQQChGroup(GuildId, ChannelId string) string {
	return fmt.Sprintf("QQ-CH-Group:%s-%s", GuildId, ChannelId)
}

func (pa *PlatformAdapterQQOnebot) Serve() int {
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	dm := s.Parent.Parent
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	pa.InPackGoCqHttpDisconnectedCH = make(chan int, 1)
	session := s

	socket := gowebsocket.New(pa.ConnectUrl)
	pa.Socket = &socket

	socket.OnConnected = func(socket gowebsocket.Socket) {
		ep.State = 1
		log.Info("onebot 连接成功")
		//  {"data":{"nickname":"闃斧鐗岃�佽檸鏈�","user_id":1001},"retcode":0,"status":"ok"}
		pa.GetLoginInfo()
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		if CheckDialErr(err) != syscall.ECONNREFUSED {
			// refused 不算大事
			log.Info("Recieved connect error: ", err)
		}
		pa.InPackGoCqHttpDisconnectedCH <- 2
	}

	// {"channel_id":"3574366","guild_id":"51541481646552899","message":"说句话试试","message_id":"BAC3HLRYvXdDAAAAAAA2il4AAAAAAAAABA==","message_type":"guild","post_type":"mes
	//sage","self_id":2589922907,"self_tiny_id":"144115218748146488","sender":{"nickname":"木落","tiny_id":"222","user_id":222},"sub_type":"channel",
	//"time":1647386874,"user_id":"144115218731218202"}

	// 疑似消息发送成功？等等 是不是可以用来取一下log
	// {"data":{"message_id":-1541043078},"retcode":0,"status":"ok"}
	var lastWelcome *LastWelcomeInfo

	// 注意这几个不能轻易delete，最好整个替换
	tempInviteMap := map[string]int64{}
	tempInviteMap2 := map[string]string{}
	tempGroupEnterSpeechSent := map[string]int64{} // 记录入群致辞的发送时间 避免短时间重复
	tempFriendInviteSent := map[string]int64{}     // gocq会重新发送已经发过的邀请

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		//if strings.Contains(message, `.`) {
		//	log.Info("...", message)
		//}
		if strings.Contains(message, `"guild_id"`) {
			//log.Info("!!!", message, s.Parent.WorkInQQChannel)
			// 暂时忽略频道消息
			if s.Parent.WorkInQQChannel {
				pa.QQChannelTrySolve(message)
			}
			return
		}

		msgQQ := new(MessageQQ)
		err := json.Unmarshal([]byte(message), msgQQ)

		if err == nil {
			// 心跳包，忽略
			if msgQQ.MetaEventType == "heartbeat" {
				return
			}
			if msgQQ.MetaEventType == "heartbeat" {
				return
			}

			if !ep.Enable {
				pa.InPackGoCqHttpDisconnectedCH <- 3
			}

			msg := msgQQ.toStdMessage()
			ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: session, Dice: session.Parent}

			if msg.Sender.UserId != "" {
				// 用户名缓存
				dm.UserNameCache.Set(msg.Sender.UserId, &GroupNameCacheItem{Name: msg.Sender.Nickname, time: time.Now().Unix()})
			}

			// 获得用户信息
			if msgQQ.Echo == -1 {
				ep.Nickname = msgQQ.Data.Nickname
				ep.UserId = FormatDiceIdQQ(msgQQ.Data.UserId)

				log.Debug("骰子信息已刷新")
				ep.RefreshGroupNum()
				return
			}

			// 自定义信息
			if pa.echoMap2 != nil {
				if v, ok := pa.echoMap2.Load(msgQQ.Echo); ok {
					v.ch <- message
					msgQQ.Echo = v.echoOverwrite
					return
				}

				now := time.Now().Unix()
				pa.echoMap2.Range(func(k int64, v *echoMapInfo) bool {
					if v.timeout != 0 && now > v.timeout {
						v.ch <- ""
					}
					return true
				})
			}

			// 获得群信息
			if msgQQ.Echo == -2 {
				if msgQQ.Data != nil {
					groupId := FormatDiceIdQQGroup(msgQQ.Data.GroupId)
					dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
						Name: msgQQ.Data.GroupName,
						time: time.Now().Unix(),
					}) // 不论如何，先试图取一下群名

					group := session.ServiceAtNew[groupId]
					if group != nil {
						if msgQQ.Data.MaxMemberCount == 0 {
							diceId := ep.UserId
							if _, exists := group.DiceIdExistsMap.Load(diceId); exists {
								// 不在群里了，更新信息
								group.DiceIdExistsMap.Delete(diceId)
								group.UpdatedAtTime = time.Now().Unix()
							}
						} else {
							// 更新群名
							if msgQQ.Data.GroupName != group.GroupName {
								group.GroupName = msgQQ.Data.GroupName
								group.UpdatedAtTime = time.Now().Unix()
							}
						}

						// 处理被强制拉群的情况
						uid := group.InviteUserId
						banInfo := ctx.Dice.BanList.GetById(uid)
						if banInfo != nil {
							if banInfo.Rank == BanRankBanned && ctx.Dice.BanList.BanBehaviorRefuseInvite {
								// 如果是被ban之后拉群，判定为强制拉群
								if group.EnteredTime > 0 && group.EnteredTime > banInfo.BanTime {
									text := fmt.Sprintf("本次入群为遭遇强制邀请，即将主动退群，因为邀请人%s正处于黑名单上。打扰各位还请见谅。感谢使用海豹核心。", group.InviteUserId)
									ReplyGroupRaw(ctx, &Message{GroupId: groupId}, text, "")
									time.Sleep(1 * time.Second)
									pa.QuitGroup(ctx, groupId)
								}
								return
							}
						}

						// 强制拉群情况2 - 群在黑名单
						banInfo = ctx.Dice.BanList.GetById(groupId)
						if banInfo != nil {
							if banInfo.Rank == BanRankBanned {
								// 如果是被ban之后拉群，判定为强制拉群
								if group.EnteredTime > 0 && group.EnteredTime > banInfo.BanTime {
									text := fmt.Sprintf("被群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", banInfo.toText(ctx.Dice))
									ReplyGroupRaw(ctx, &Message{GroupId: groupId}, text, "")
									time.Sleep(1 * time.Second)
									pa.QuitGroup(ctx, groupId)
								}
								return
							}
						}

					} else {
						// TODO: 这玩意的创建是个专业活，等下来弄
						//session.ServiceAtNew[groupId] = GroupInfo{}
					}
					// 这句话太吵了
					//log.Debug("群信息刷新: ", msgQQ.Data.GroupName)
				}
				return
			}

			// 自定义信息
			if pa.echoMap != nil {
				if v, ok := pa.echoMap.Load(msgQQ.Echo); ok {
					v <- msgQQ
					return
				}
			}

			// 处理加群请求
			if msgQQ.PostType == "request" && msgQQ.RequestType == "group" && msgQQ.SubType == "invite" {
				// {"comment":"","flag":"111","group_id":222,"post_type":"request","request_type":"group","self_id":333,"sub_type":"invite","time":1646782195,"user_id":444}
				ep.RefreshGroupNum()
				pa.GetGroupInfoAsync(msg.GroupId)
				time.Sleep(time.Duration((1.8 + rand.Float64()) * float64(time.Second))) // 稍作等待，也许能拿到群名

				uid := FormatDiceIdQQ(msgQQ.UserId)
				groupName := dm.TryGetGroupName(msg.GroupId)
				userName := dm.TryGetUserName(uid)
				txt := fmt.Sprintf("收到QQ加群邀请: 群组<%s>(%d) 邀请人:<%s>(%d)", groupName, msgQQ.GroupId, userName, msgQQ.UserId)
				log.Info(txt)
				ctx.Notice(txt)
				tempInviteMap[msg.GroupId] = time.Now().Unix()
				tempInviteMap2[msg.GroupId] = uid

				// 邀请人在黑名单上
				banInfo := ctx.Dice.BanList.GetById(uid)
				if banInfo != nil {
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
				banInfo = ctx.Dice.BanList.GetById(msg.GroupId)
				if banInfo != nil {
					if banInfo.Rank == BanRankBanned {
						pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, false, "群黑名单")
						return
					}
				}

				if ctx.Dice.RefuseGroupInvite {
					pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, false, "设置拒绝加群")
					return
				}

				//time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))
				pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, true, "")
				return
			}

			// 好友请求
			if msgQQ.PostType == "request" && msgQQ.RequestType == "friend" {
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
				uid := FormatDiceIdQQ(msgQQ.UserId)
				banInfo := ctx.Dice.BanList.GetById(uid)
				if banInfo != nil {
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

				txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%d, 验证信息: %s, 是否自动同意: %t%s", msgQQ.UserId, comment, willAccept, extra)
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
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Errorf("好友致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
						}
					}()

					// 稍作等待后发好友致辞
					time.Sleep(2 * time.Second)

					msg.Sender.UserId = FormatDiceIdQQ(msgQQ.UserId)
					ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
					uid := FormatDiceIdQQ(msgQQ.UserId)

					welcome := DiceFormatTmpl(ctx, "核心:骰子成为好友")
					log.Infof("与 %s 成为好友，发送好友致辞: %s", uid, welcome)

					for _, i := range strings.Split(welcome, "###SPLIT###") {
						doSleepQQ(ctx)
						pa.SendToPerson(ctx, uid, strings.TrimSpace(i), "")
					}
				}()
				return
			}

			groupEnterFired := false
			groupEntered := func() {
				if groupEnterFired {
					return
				}
				groupEnterFired = true
				lastTime := tempGroupEnterSpeechSent[msg.GroupId]
				nowTime := time.Now().Unix()

				if nowTime-lastTime < 10 {
					// 10s内只发一次
					return
				}
				tempGroupEnterSpeechSent[msg.GroupId] = nowTime

				// 判断进群的人是自己，自动启动
				gi := SetBotOnAtGroup(ctx, msg.GroupId)
				if tempInviteMap2[msg.GroupId] != "" {
					// 设置邀请人
					gi.InviteUserId = tempInviteMap2[msg.GroupId]
				}
				gi.DiceIdExistsMap.Store(msg.GroupId, true)
				gi.EnteredTime = nowTime // 设置入群时间
				gi.UpdatedAtTime = time.Now().Unix()
				// 立即获取群信息
				pa.GetGroupInfoAsync(msg.GroupId)
				// fmt.Sprintf("<%s>已经就绪。可通过.help查看指令列表", conn.Nickname)

				time.Sleep(2 * time.Second)
				groupName := dm.TryGetGroupName(msg.GroupId)
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Errorf("入群致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
						}
					}()

					// 稍作等待后发送入群致词
					time.Sleep(1 * time.Second)

					ctx.Player = &GroupPlayerInfo{}
					log.Infof("发送入群致辞，群: <%s>(%d)", groupName, msgQQ.GroupId)
					text := DiceFormatTmpl(ctx, "核心:骰子进群")
					for _, i := range strings.Split(text, "###SPLIT###") {
						doSleepQQ(ctx)
						pa.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), "")
					}
				}()
				txt := fmt.Sprintf("加入QQ群组: <%s>(%d)", groupName, msgQQ.GroupId)
				log.Info(txt)
				ctx.Notice(txt)
			}

			// 入群的另一种情况: 管理员审核
			group := s.ServiceAtNew[msg.GroupId]
			if group == nil && msg.GroupId != "" {
				now := time.Now().Unix()
				if tempInviteMap[msg.GroupId] != 0 && now > tempInviteMap[msg.GroupId] {
					delete(tempInviteMap, msg.GroupId)
					groupEntered()
				}
				//log.Infof("自动激活: 发现无记录群组(%s)，因为已是群成员，所以自动激活", group.GroupId)
			}

			// 入群后自动开启
			if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_increase" {
				//{"group_id":111,"notice_type":"group_increase","operator_id":0,"post_type":"notice","self_id":333,"sub_type":"approve","time":1646782012,"user_id":333}
				if msgQQ.UserId == msgQQ.SelfId {
					groupEntered()
				} else {
					group := session.ServiceAtNew[msg.GroupId]
					// 进群的是别人，是否迎新？
					// 这里很诡异，当手机QQ客户端审批进群时，入群后会有一句默认发言
					// 此时会收到两次完全一样的某用户入群信息，导致发两次欢迎词
					if group != nil && group.ShowGroupWelcome {
						isDouble := false
						if lastWelcome != nil {
							isDouble = msgQQ.GroupId == lastWelcome.GroupId &&
								msgQQ.UserId == lastWelcome.UserId &&
								msgQQ.Time == lastWelcome.Time
						}
						lastWelcome = &LastWelcomeInfo{
							GroupId: msgQQ.GroupId,
							UserId:  msgQQ.UserId,
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
								//VarSetValueStr(ctx, "$t新人昵称", "<"+msgQQ.Sender.Nickname+">")
								uidRaw := strconv.FormatInt(msgQQ.UserId, 10)
								VarSetValueStr(ctx, "$t帐号ID_RAW", uidRaw)
								VarSetValueStr(ctx, "$t账号ID_RAW", uidRaw)
								stdId := FormatDiceIdQQ(msgQQ.UserId)
								VarSetValueStr(ctx, "$t帐号ID", stdId)
								VarSetValueStr(ctx, "$t账号ID", stdId)
								text := DiceFormat(ctx, group.GroupWelcomeMessage)
								for _, i := range strings.Split(text, "###SPLIT###") {
									doSleepQQ(ctx)
									pa.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), "")
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
				if msgQQ.UserId == msgQQ.SelfId {
					opUid := FormatDiceIdQQ(msgQQ.OperatorId)
					groupName := dm.TryGetGroupName(msg.GroupId)
					userName := dm.TryGetUserName(opUid)

					ctx.Dice.BanList.AddScoreByGroupKicked(opUid, msg.GroupId, ctx)
					txt := fmt.Sprintf("被踢出群: 在QQ群组<%s>(%d)中被踢出，操作者:<%s>(%d)", groupName, msgQQ.GroupId, userName, msgQQ.OperatorId)
					log.Info(txt)
					ctx.Notice(txt)
				}
				return
			}

			if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_decrease" && msgQQ.SubType == "leave" && msgQQ.OperatorId == msgQQ.SelfId {
				// 群解散
				// {"group_id":564808710,"notice_type":"group_decrease","operator_id":2589922907,"post_type":"notice","self_id":2589922907,"sub_type":"leave","time":1651584460,"user_id":2589922907}
				groupName := dm.TryGetGroupName(msg.GroupId)
				txt := fmt.Sprintf("离开群组或群解散: <%s>(%d)", groupName, msgQQ.GroupId)
				log.Info(txt)
				ctx.Notice(txt)
				return
			}

			if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_ban" && msgQQ.SubType == "ban" {
				// 禁言
				// {"duration":600,"group_id":111,"notice_type":"group_ban","operator_id":222,"post_type":"notice","self_id":333,"sub_type":"ban","time":1646689567,"user_id":333}
				if msgQQ.UserId == msgQQ.SelfId {
					opUid := FormatDiceIdQQ(msgQQ.OperatorId)
					groupName := dm.TryGetGroupName(msg.GroupId)
					userName := dm.TryGetUserName(opUid)

					ctx.Dice.BanList.AddScoreByGroupMuted(opUid, msg.GroupId, ctx)
					txt := fmt.Sprintf("被禁言: 在群组<%s>(%d)中被禁言，时长%d秒，操作者:<%s>(%d)", groupName, msgQQ.GroupId, msgQQ.Duration, userName, msgQQ.OperatorId)
					log.Info(txt)
					ctx.Notice(txt)
				}
				return
			}

			// 消息撤回
			if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_recall" {
				group := s.ServiceAtNew[msg.GroupId]
				if group != nil {
					if group.LogOn {
						_ = model.LogMarkDeleteByMsgId(ctx.Dice.DBLogs, group.GroupId, group.LogCurName, msgQQ.MessageId)
					}
				}
				return
			}

			if msgQQ.PostType == "" && msgQQ.Msg == "SEND_MSG_API_ERROR" && msgQQ.Retcode == 100 {
				// 群消息发送失败: 账号可能被风控，戳对方一下
				// {"data":null,"echo":0,"msg":"SEND_MSG_API_ERROR","retcode":100,"status":"failed","wording":"请参考 go-cqhttp 端输出"}
				// 但是这里没QQ号也没有消息ID，很麻烦
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

					if msgQQ.TargetId == msgQQ.SelfId {
						// 如果在戳自己
						text := DiceFormatTmpl(ctx, "其它:戳一戳")
						for _, i := range strings.Split(text, "###SPLIT###") {
							doSleepQQ(ctx)
							switch msg.MessageType {
							case "group":
								pa.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), "")
							case "private":
								pa.SendToPerson(ctx, msg.Sender.UserId, strings.TrimSpace(i), "")
							}
						}
					}
				}()
				return
			}

			// 处理命令
			if msgQQ.MessageType == "group" || msgQQ.MessageType == "private" {
				if msg.Sender.UserId == ep.UserId {
					// 以免私聊时自己发的信息被记录
					// 这里的私聊消息可能是自己发送的
					// 要是群发也可以被记录就好了
					// XXXX {"font":0,"message":"\u003c木落\u003e的今日人品为83","message_id":-358748624,"message_type":"private","post_type":"message_sent","raw_message":"\u003c木落\u003e的今日人
					//品为83","self_id":2589922907,"sender":{"age":0,"nickname":"海豹一号机","sex":"unknown","user_id":2589922907},"sub_type":"friend","target_id":222,"time":1647760835,"use
					//r_id":2589922907}
					return
				}

				//fmt.Println("Recieved message1 " + message)
				session.Execute(ep, msg, false)
			} else {
				fmt.Println("Recieved message " + message)
			}
		} else {
			log.Error("error" + err.Error())
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

	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Info("onebot 服务的连接被对方关闭 ")
		pa.InPackGoCqHttpDisconnectedCH <- 1
	}

	socket.Connect()
	defer func() {
		//fmt.Println("socket close")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 太频繁了 不输出了
					//fmt.Println("关闭连接时遭遇异常")
					//core.GetLogger().Error(r)
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
			pa.InPackGoCqHttpDisconnectedCH <- 0
			return 0
		case val := <-pa.InPackGoCqHttpDisconnectedCH:
			return val
		}
	}
}

func (pa *PlatformAdapterQQOnebot) DoRelogin() bool {
	myDice := pa.Session.Parent
	ep := pa.EndPoint
	if pa.Socket != nil {
		go pa.Socket.Close()
		pa.Socket = nil
	}
	if pa.UseInPackGoCqhttp {
		if pa.InPackGoCqHttpDisconnectedCH != nil {
			pa.InPackGoCqHttpDisconnectedCH <- -1
		}
		myDice.Logger.Infof("重新启动go-cqhttp进程，对应账号: <%s>(%s)", ep.Nickname, ep.UserId)
		pa.CurLoginIndex += 1
		pa.GoCqHttpState = GoCqHttpStateCodeInit
		go GoCqHttpServeProcessKill(myDice, ep)
		time.Sleep(10 * time.Second)                // 上面那个清理有概率卡住，具体不懂，改成等5s -> 10s 超过一次重试间隔
		GoCqHttpServeRemoveSessionToken(myDice, ep) // 删除session.token
		pa.GoCqHttpLastRestrictedTime = 0           // 重置风控时间
		GoCqHttpServe(myDice, ep, pa.InPackGoCqHttpPassword, pa.InPackGoCqHttpProtocol, true)
		return true
	}
	return false
}

func (pa *PlatformAdapterQQOnebot) SetEnable(enable bool) {
	d := pa.Session.Parent
	c := pa.EndPoint
	if enable {
		c.Enable = true
		pa.DiceServing = false

		if pa.UseInPackGoCqhttp {
			GoCqHttpServeProcessKill(d, c)
			time.Sleep(1 * time.Second)
			GoCqHttpServe(d, c, pa.InPackGoCqHttpPassword, pa.InPackGoCqHttpProtocol, true)
			go ServeQQ(d, c)
		} else {
			go ServeQQ(d, c)
		}
	} else {
		c.Enable = false
		pa.DiceServing = false
		if pa.UseInPackGoCqhttp {
			GoCqHttpServeProcessKill(d, c)
		}
	}
}

func (pa *PlatformAdapterQQOnebot) SetQQProtocol(protocol int) bool {
	//oldProtocol := pa.InPackGoCqHttpProtocol
	pa.InPackGoCqHttpProtocol = protocol

	//ep.Session.Parent.GetDiceDataPath(ep.RelWorkDir)
	workDir := filepath.Join(pa.Session.Parent.BaseConfig.DataDir, pa.EndPoint.RelWorkDir)
	deviceFilePath := filepath.Join(workDir, "device.json")
	if _, err := os.Stat(deviceFilePath); err == nil {
		configFile, err := os.ReadFile(deviceFilePath)
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

func (pa *PlatformAdapterQQOnebot) IsInLogin() bool {
	return pa.GoCqHttpState < GoCqHttpStateCodeLoginSuccessed
}

func (pa *PlatformAdapterQQOnebot) IsLoginSuccessed() bool {
	return pa.GoCqHttpState == GoCqHttpStateCodeLoginSuccessed
}

func (pa *PlatformAdapterQQOnebot) packTempCtx(msgQQ *MessageQQ, msg *Message) *MsgContext {
	ep := pa.EndPoint
	session := pa.Session

	ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: session, Dice: session.Parent}

	switch msg.MessageType {
	case "private":
		d := pa.GetStrangerInfo(msgQQ.UserId) // 先获取个人信息，避免不存在id
		msg.Sender.UserId = FormatDiceIdQQ(msgQQ.UserId)
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		if ctx.Player.Name == "" {
			ctx.Player.Name = d.Nickname
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, ctx.Player.Name)
	case "group":
		d := pa.GetGroupMemberInfo(msgQQ.GroupId, msgQQ.UserId) // 先获取个人信息，避免不存在id
		msg.Sender.UserId = FormatDiceIdQQ(msgQQ.UserId)
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		if ctx.Player.Name == "" {
			ctx.Player.Name = d.Card
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, ctx.Player.Name)
	}

	return ctx
}
