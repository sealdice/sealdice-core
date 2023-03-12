package dice

import (
	"encoding/json"
	"errors"
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

/* 定义结构体 */

type PlatformAdapterWalleQ struct {
	EndPoint        *EndPointInfo       `yaml:"-" json:"-"`
	Session         *IMSession          `yaml:"-" json:"-"`
	Socket          *gowebsocket.Socket `yaml:"-" json:"-"`
	ConnectUrl      string              `yaml:"connectUrl" json:"connectUrl"`           // 连接地址
	UseInPackWalleQ bool                `yaml:"useInPackWalleQ" json:"useInPackWalleQ"` // 是否使用内置的WalleQ
	WalleQState     int                 `yaml:"-" json:"loginState"`                    // 当前状态
	CurLoginIndex   int                 `yaml:"-" json:"curLoginIndex"`                 // 当前登录序号，如果正在进行的登录不是该Index，证明过时

	WalleQProcess           *procs.Process `yaml:"-" json:"-"`
	WalleQLoginFailedReason string         `yaml:"-" json:"curLoginFailedReason"` // 当前登录失败原因

	WalleQLoginVerifyCode    string `yaml:"-" json:"WalleQLoginVerifyCode"`
	WalleQLoginDeviceLockUrl string `yaml:"-" json:"WalleQLoginDeviceLockUrl"`
	WalleQQrcodeData         []byte `yaml:"-" json:"-"` // 二维码数据

	WalleQLastAutoLoginTime  int64 `yaml:"inPackGoCqLastAutoLoginTime" json:"-"`                         // 上次自动重新登录的时间
	WalleQLoginSucceeded     bool  `yaml:"inPackWalleQLoginSucceeded" json:"-"`                          // 是否登录成功过
	WalleQLastRestrictedTime int64 `yaml:"inPackWalleQLastRestricted" json:"inPackWalleQLastRestricted"` // 上次风控时间

	InPackWalleQProtocol       int      `yaml:"inPackWalleQProtocol" json:"inPackWalleQProtocol"`
	InPackWalleQPassword       string   `yaml:"inPackWalleQPassword" json:"-"`
	DiceServing                bool     `yaml:"-"`                                              // 是否正在连接中
	InPackWalleQDisconnectedCH chan int `yaml:"-" json:"-"`                                     // 信号量，用于关闭连接
	IgnoreFriendRequest        bool     `yaml:"ignoreFriendRequest" json:"ignoreFriendRequest"` // 忽略好友请求处理开关

	echoMap        *SyncMap[string, chan *EventWalleQBase] `yaml:"-"`
	echoMap2       *SyncMap[string, *echoMapInfo]          `yaml:"-"`
	FileMap        *SyncMap[string, string]                // 记录上传文件后得到的 id
	Implementation string                                  `yaml:"implementation" json:"implementation"`
}

type EventWalleQBase struct {
	Id         string  `json:"id"`          // 事件唯一标识符
	Self       Self    `json:"self"`        // 机器人自身标识
	Time       float64 `json:"time"`        // 事件发生时间（Unix 时间戳），单位：秒
	Type       string  `json:"type"`        // meta、message、notice、request 中的一个，分别表示元事件、消息事件、通知事件和请求事件
	DetailType string  `json:"detail_type"` // 详细
	SubType    string  `json:"sub_type"`    // 子类型
	// 下面这些虽然不是共有字段 但基本也算
	UserId    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserCard  string `json:"user_card"` // 群名片
	GroupId   string `json:"group_id"`
	GroupName string `json:"group_name"`
	GuildId   string `json:"guild_id"`
	ChannelId string `json:"channel_id"`
}

type Bot struct {
	Self   Self `json:"self"`
	Online bool `json:"online"`
}

// EventWalleQMeta 元事件特有字段
type EventWalleQMeta struct {
	Version struct {
		Impl          string `json:"impl"`
		Version       string `json:"version"`
		OneBotVersion string `json:"onebot_version"`
	} `json:"version"`
	Status struct {
		Good bool  `json:"good"`
		Bots []Bot `json:"bots"`
	} `json:"status"`
}

// EventWalleQMsg 消息事件特有字段
type EventWalleQMsg struct {
	MessageId  string           `json:"message_id"`  // 消息id
	Message    []MessageSegment `json:"message"`     // 消息段
	AltMessage string           `json:"alt_message"` // 文本化
}

// EventWalleQNotice 通知事件特有字段
type EventWalleQNotice struct {
	MessageId  string `json:"message_id"`  // 消息id
	OperatorId string `json:"operator_id"` // 操作者账号
	ReceiverId string `json:"receiver_id"` // 戳一戳的接收者
	Duration   int64  `json:"duration"`
}

// EventWalleQReq 请求事件特有字段
type EventWalleQReq struct {
	Message     string `json:"message"`
	RequestId   int64  `json:"request_id"` // 请求者
	Suspicious  bool   `json:"suspicious"`
	InvitorId   string `json:"invitor_id"`   // 邀请者
	InvitorName string `json:"invitor_name"` // 邀请者名
}

type Self struct {
	Platform string `json:"platform"`
	UserId   string `json:"user_id"`
}

type MessageSegment struct {
	Type string `json:"type"`
	Data MSData `json:"data"`
}

type MSData struct {
	Id        int64   `json:"id,omitempty"`
	Text      string  `json:"text,omitempty"`
	UserId    string  `json:"user_id,omitempty"`
	UserName  string  `json:"user_name,omitempty"`
	Face      string  `json:"face,omitempty"`
	MessageId string  `json:"message_id,omitempty"`
	FileId    string  `json:"fileId,omitempty"`
	Time      float64 `json:"time,omitempty"`
	Url       string  `json:"url,omitempty"`
}

type EchoWalleQ struct {
	Status  string `json:"status"`
	RetCode int64  `json:"retcode"`
	// 先简单处理一下
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
	Echo    string                 `json:"echo"`
}

type LastWelcomeInfoWQ struct {
	UserId  string
	GroupId string
	Time    float64
}

type OneBotV12Command struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   string      `json:"echo"`
}

type OnebotV12UserInfo struct {
	// 个人信息
	Nickname string `json:"nickname"`
	UserId   string `json:"user_id"`

	// 群信息
	GroupId         string `json:"group_id"`          // 群号
	GroupCreateTime uint32 `json:"group_create_time"` // 群号
	MemberCount     int64  `json:"member_count"`
	GroupName       string `json:"group_name"`
	MaxMemberCount  int32  `json:"max_member_count"`
	Card            string `json:"card"`
}

/* 标准方法实现 */

func (pa *PlatformAdapterWalleQ) Serve() int {
	pa.Implementation = "walle-q"
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	dm := s.Parent.Parent
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	pa.InPackWalleQDisconnectedCH = make(chan int, 1)
	//session := s

	socket := gowebsocket.New(pa.ConnectUrl)
	pa.Socket = &socket

	socket.OnConnected = func(socket gowebsocket.Socket) {
		ep.State = 1
		log.Info("onebot 连接成功")
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		if CheckDialErr(err) != syscall.ECONNREFUSED {
			// refused 不算大事
			log.Info("Recieved connect error: ", err)
		}
		pa.InPackWalleQDisconnectedCH <- 2
	}
	var lastWelcome *LastWelcomeInfoWQ

	// 注意这几个不能轻易delete，最好整个替换
	tempInviteMap := map[string]int64{}
	tempInviteMap2 := map[string]string{}
	tempGroupEnterSpeechSent := map[string]int64{} // 记录入群致辞的发送时间 避免短时间重复
	//tempFriendInviteSent := map[string]int64{}     // gocq会重新发送已经发过的邀请

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		fmt.Println(message)
		event := new(EventWalleQBase)
		err := json.Unmarshal([]byte(message), event)
		if err != nil {
			log.Error("事件解析错误：", err.Error())
			return
		}

		msg := event.toMessageBase()
		ctx := &MsgContext{MessageType: event.DetailType, EndPoint: ep, Session: s, Dice: s.Parent}
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
				log.Infof("发送入群致辞，群: <%s>(%s)", groupName, event.GroupId)
				text := DiceFormatTmpl(ctx, "核心:骰子进群")
				for _, i := range strings.Split(text, "###SPLIT###") {
					doSleepQQ(ctx)
					pa.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), "")
				}
			}()
			txt := fmt.Sprintf("加入QQ群组: <%s>(%s)", groupName, event.GroupId)
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
		}

		if event.Type == "meta" {
			// {"id":"","time":1677991588.2299678,"type":"meta","detail_type":"status_update","sub_type":"","status":{"good":true,"bots":[{"self":{"platform":"qq","user_id":"2604200975"},"online":true}]}}
			meta := new(EventWalleQMeta)
			err = json.Unmarshal([]byte(message), meta)
			if err != nil {
				log.Error(err.Error())
				return
			}
			// 连接成功事件
			if event.DetailType == "connect" {
				ep.State = 1
				log.Info(meta.Version.Impl + "连接成功 >>> Walle-q 版本：" + meta.Version.Version + " | OneBot 协议版本：" + meta.Version.OneBotVersion)
			}

			if event.DetailType == "status_update" {
				if meta.Status.Good {
					log.Info("walle-q 运行正常")
				}
				if len(meta.Status.Bots) == 0 {
					log.Info("没有账号上线")
				}
				for _, i := range meta.Status.Bots {
					var t string
					if i.Online {
						t = "已经上线"
					} else {
						t = "未上线"
					}
					log.Info(i.Self, t)
				}
			}
			// 忽略心跳
			return
		}

		if event.Type == "message" {
			msgQQ := new(EventWalleQMsg)
			err = json.Unmarshal([]byte(message), msgQQ)
			if err != nil {
				log.Error(err.Error())
				return
			}

			msg.Message = MessageSegmentToText(msgQQ.Message)
			if msg.Sender.UserId != "" {
				if msg.Sender.Nickname != "" {
					dm.UserNameCache.Set(msg.Sender.UserId, &GroupNameCacheItem{Name: msg.Sender.Nickname, time: time.Now().Unix()})
				}
			}

			pa.Session.Execute(pa.EndPoint, msg, false) // wq 还没有频道支持，直接执行
		}

		if event.Type == "notice" {
			n := new(EventWalleQNotice)
			opUid := FormatDiceIdQQV12(n.OperatorId)
			groupName := dm.TryGetGroupName(msg.GroupId)
			userName := dm.TryGetUserName(opUid)
			switch event.DetailType {
			case "friend_poke":
				return
			case "friend_increase":
				return
			case "friend_decrease": // 好友被删，哀悼一下？
				return
			case "group_member_increase":
				//_ = session.ServiceAtNew[msg.GroupId]
				if event.UserId == event.Self.UserId {
					groupEntered()
				} else {
					group := s.ServiceAtNew[msg.GroupId]
					// 进群的是别人，是否迎新？
					// 这里很诡异，当手机QQ客户端审批进群时，入群后会有一句默认发言
					// 此时会收到两次完全一样的某用户入群信息，导致发两次欢迎词 // 如果是 TX BUG 这里就不改了
					if group != nil && group.ShowGroupWelcome {
						isDouble := false
						if lastWelcome != nil {
							isDouble = event.GroupId == lastWelcome.GroupId &&
								event.UserId == lastWelcome.UserId &&
								event.Time == lastWelcome.Time
						}
						lastWelcome = &LastWelcomeInfoWQ{
							GroupId: event.GroupId,
							UserId:  event.UserId,
							Time:    event.Time,
						}

						if !isDouble {
							func() {
								defer func() {
									if r := recover(); r != nil {
										log.Errorf("迎新致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
									}
								}()

								ctx.Player = &GroupPlayerInfo{}
								VarSetValueStr(ctx, "$t帐号ID_RAW", event.GroupId)
								VarSetValueStr(ctx, "$t账号ID_RAW", event.GroupId)
								stdId := FormatDiceIdQQV12(event.UserId)
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
			case "group_member_decrease": //  被提出
				if event.UserId == event.Self.UserId {
					skip := false
					skipReason := ""
					banInfo := ctx.Dice.BanList.GetById(opUid)
					if banInfo != nil {
						if banInfo.Rank == 30 {
							skip = true
							skipReason = "信任用户"
						}
					}
					if ctx.Dice.IsMaster(opUid) {
						skip = true
						skipReason = "Master"
					}

					var extra string
					if skip {
						extra = fmt.Sprintf("\n取消处罚，原因为%s", skipReason)
					} else {
						ctx.Dice.BanList.AddScoreByGroupKicked(opUid, msg.GroupId, ctx)
					}

					txt := fmt.Sprintf("被踢出群: 在QQ群组<%s>(%s)中被踢出，操作者:<%s>(%s)%s", groupName, event.GroupId, userName, n.OperatorId, extra)
					log.Info(txt)
					ctx.Notice(txt)
				}
			case "group_member_ban": // 被禁言
				if event.UserId == event.Self.UserId {
					ctx.Dice.BanList.AddScoreByGroupMuted(opUid, msg.GroupId, ctx)
					txt := fmt.Sprintf("被禁言: 在群组<%s>(%s)中被禁言，时长%d秒，操作者:<%s>(%s)", groupName, msg.GroupId, n.Duration, userName, n.OperatorId)
					log.Info(txt)
					ctx.Notice(txt)
				}
				return
			case "group_message_delete": // 消息撤回
				group := s.ServiceAtNew[msg.GroupId]
				if group != nil {
					if group.LogOn {
						_ = model.LogMarkDeleteByMsgId(ctx.Dice.DBLogs, group.GroupId, group.LogCurName, n.MessageId)
					}
				}
				return
			case "group_admin_set":
				return
			case "group_admin_unset":
				return
			case "group_name_update":
				return
			}
		}

		if event.Type == "request" {
			req := new(EventWalleQReq)
			err = json.Unmarshal([]byte(message), req)
			if err != nil {
				return
			}
			ctx := &MsgContext{MessageType: event.Type, EndPoint: ep, Session: s, Dice: s.Parent}
			switch event.DetailType {
			// 好友喜加一
			case "new_friend":
				// wq 没有重发问题，但是好像第一次接受好友请求时会初始化一遍好友列表……？
				// {'id': '00000000-0000-0000-1748-5e03b646be88', 'time': 1677694231.231515, 'type': 'request', 'detail_type': 'new_friend', 'sub_type': '', 'request_id': 1677694230000000, 'user_name': '冰块PSR', 'message': '问题1:你是谁？\n回答:我就是你\n问题2:你是哪里人？\n回答:这里人\n问题3:你多大了？\n回答:不大', 'user_id': '360326608', 'self': {'user_id': '2604200975', 'platform': 'qq'}}
				var comment string
				if req.Message != "" {
					comment = strings.TrimSpace(req.Message)
					comment = strings.ReplaceAll(comment, "\u00a0", "")
				}

				toMatch := strings.TrimSpace(s.Parent.FriendAddComment)
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
				uid := msg.Sender.UserId
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

				txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", event.UserId, comment, willAccept, extra)
				log.Info(txt)
				ctx.Notice(txt)

				// 忽略邀请
				if pa.IgnoreFriendRequest {
					return
				}

				time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

				if willAccept {
					pa.SetFriendAddRequest(req.RequestId, event.UserId, true)
				} else {
					pa.SetFriendAddRequest(req.RequestId, event.UserId, false)
				}
				return
			// 其他人的加群申请 // 管他呢
			case "join_group":
				return
			// 加群邀请
			case "group_invited":
				// {"comment":"","flag":"111","group_id":222,"post_type":"request","request_type":"group","self_id":333,"sub_type":"invite","time":1646782195,"user_id":444}
				ep.RefreshGroupNum()
				pa.GetGroupInfoAsync(event.GroupId)
				time.Sleep(time.Duration((1.8 + rand.Float64()) * float64(time.Second))) // 稍作等待，也许能拿到群名

				uid := FormatDiceIdQQV12(event.UserId)
				gid := FormatDiceIdQQGroupV12(event.GroupId)
				groupName := dm.TryGetGroupName(event.GroupId)
				userName := dm.TryGetUserName(uid)
				txt := fmt.Sprintf("收到QQ加群邀请: 群组<%s>(%s) 邀请人:<%s>(%s)", groupName, event.GroupId, userName, event.UserId)
				log.Info(txt)
				ctx.Notice(txt)
				//tempInviteMap[msg.GroupId] = time.Now().Unix()
				//tempInviteMap2[msg.GroupId] = uid

				// 邀请人在黑名单上
				banInfo := ctx.Dice.BanList.GetById(uid)
				if banInfo != nil {
					if banInfo.Rank == BanRankBanned && ctx.Dice.BanList.BanBehaviorRefuseInvite {
						pa.SetGroupAddRequest(req.RequestId, event.GroupId, false)
						return
					}
				}
				// 信任模式，如果不是信任，又不是master则拒绝拉群邀请
				isMaster := ctx.Dice.IsMaster(uid)
				if ctx.Dice.TrustOnlyMode && ((banInfo != nil && banInfo.Rank != BanRankTrusted) && !isMaster) {
					pa.SetGroupAddRequest(req.RequestId, event.GroupId, false)
					return
				}
				// 群在黑名单上
				banInfo = ctx.Dice.BanList.GetById(gid)
				if banInfo != nil {
					if banInfo.Rank == BanRankBanned {
						pa.SetGroupAddRequest(req.RequestId, event.GroupId, false)
						return
					}
				}
				// 拒绝加入新群
				if ctx.Dice.RefuseGroupInvite {
					pa.SetGroupAddRequest(req.RequestId, event.GroupId, false)
					return
				}

				pa.SetGroupAddRequest(req.RequestId, event.GroupId, true)
				return
			}
		}

		// 事件都有 ID，没有就是响应 but 有几个元事件 ID 是 "" ；把响应处理放到最后吧
		if event.Id == "" {
			fmt.Println(message)
			echo := new(EchoWalleQ)
			err = json.Unmarshal([]byte(message), echo)
			if err != nil {
				log.Error("响应解析错误", err.Error())
				return
			}
			if echo.Status != "ok" {
				log.Error("响应返回错误", echo.Message)
				return
			}

			m := echo.Data
			switch echo.Echo {
			case "send_message":
				if echo.Status != "ok" {
					log.Warn("消息发送失败。")
				}
				return
			case "get_self_info":
				ep.Nickname = m["user_name"].(string)
				ep.UserId = FormatDiceIdQQV12(m["user_id"].(string))
				d := pa.Session.Parent
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				return
			case "get_group_info":
				groupId := FormatDiceIdQQGroupV12(m["group_id"].(string))
				GroupName := m["group_name"].(string)
				ctx := &MsgContext{MessageType: "group", EndPoint: ep, Session: s, Dice: s.Parent}
				dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
					Name: GroupName,
					time: time.Now().Unix(),
				}) // 不论如何，先试图取一下群名

				group := s.ServiceAtNew[groupId]
				if group != nil {
					// 更新群名
					if GroupName != group.GroupName {
						group.GroupName = GroupName
						group.UpdatedAtTime = time.Now().Unix()
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
				return
			case "get_group_member_info":
				//pa.echoMap.Store("get_group_member_info", )

			}

			if pa.echoMap != nil {

			}
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
		pa.InPackWalleQDisconnectedCH <- 1
	}

	socket.Connect()
	defer func() {
		fmt.Println("socket close")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("关闭连接时遭遇异常")
				}
			}()
			socket.Close()
		}()
	}()

	for {
		select {
		case <-interrupt:
			log.Info("interrupt")
			pa.InPackWalleQDisconnectedCH <- 0
			return 0
		case val := <-pa.InPackWalleQDisconnectedCH:
			return val
		}
	}
}

func (pa *PlatformAdapterWalleQ) DoRelogin() bool {
	d := pa.Session.Parent
	ep := pa.EndPoint
	if pa.Socket != nil {
		go pa.Socket.Close()
		pa.Socket = nil
	}
	if pa.UseInPackWalleQ {
		if pa.InPackWalleQDisconnectedCH != nil {
			pa.InPackWalleQDisconnectedCH <- -1
		}
		d.Logger.Infof(fmt.Sprintf("重启 Walle-q，账号<%s>(%s)", ep.Nickname, ep.UserId))
		pa.CurLoginIndex += 1
		pa.WalleQState = WqStateCodeInit
		go WalleQServeProcessKill(d, ep)
		time.Sleep(10 * time.Second)
		WalleQServeRemoveSessionToken(d, ep)
		pa.WalleQLastRestrictedTime = 0
		WalleQServe(d, ep, pa.InPackWalleQPassword, pa.InPackWalleQProtocol, true)
		return true
	}
	return false
}

func (pa *PlatformAdapterWalleQ) SetEnable(enable bool) {
	d := pa.Session.Parent
	c := pa.EndPoint
	if enable {
		c.Enable = true
		pa.DiceServing = false

		if pa.UseInPackWalleQ {
			WalleQServeProcessKill(d, c)
			time.Sleep(1 * time.Second)
			WalleQServe(d, c, pa.InPackWalleQPassword, pa.InPackWalleQProtocol, true)
			go ServeQQ(d, c)
		} else {
			go ServeQQ(d, c)
		}
	} else {
		c.Enable = false
		pa.DiceServing = false
		if pa.UseInPackWalleQ {
			WalleQServeProcessKill(d, c)
		}
	}

	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

func (pa *PlatformAdapterWalleQ) GetGroupInfoAsync(id string) {
	type GroupMessageParams struct {
		GroupId string `json:"group_id"`
	}
	realGroupId, type_ := pa.mustExtractId(id)
	if type_ != QQUidGroup {
		return
	}

	a, _ := json.Marshal(OneBotV12Command{
		"get_group_info",
		GroupMessageParams{
			realGroupId,
		},
		"get_group_info",
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterWalleQ) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	rawId, type_ := pa.mustExtractId(userId)

	if type_ != QQUidPerson {
		return
	}

	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, "private", userId, text, flag)
			})
		}
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)
	for _, subText := range texts {
		pa.SendMessage(subText, "private", rawId, "")
		doSleepQQ(ctx)
	}
}

func (pa *PlatformAdapterWalleQ) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	if groupId == "" {
		return
	}
	rawId, type_ := pa.mustExtractId(groupId)
	if type_ == 0 {
		// pa.SendToChannelGroup(ctx, groupId, text, flag) wq 未实现
		return
	}

	if ctx.Session.ServiceAtNew[groupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, "group", groupId, text, flag)
				})
			}
		}
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)

	for index, subText := range texts {
		pa.SendMessage(subText, "group", rawId, "")
		if len(texts) > 1 && index != 0 {
			doSleepQQ(ctx)
		}
	}
}

func (pa *PlatformAdapterWalleQ) QuitGroup(ctx *MsgContext, id string) {
	groupId, type_ := pa.mustExtractId(id)
	if type_ != QQUidGroup {
		return
	}
	type GroupMessageParams struct {
		GroupId string `json:"group_id"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "leave_group",
		Params: GroupMessageParams{
			groupId,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterWalleQ) SetGroupCardName(groupId string, userId string, name string) {
	// wq 暂无该扩展 api
}

func (pa *PlatformAdapterWalleQ) MemberBan(groupId string, userId string, last int64) {
	type P struct {
		GroupId  string `json:"groupId"`
		UserId   string `json:"userId"`
		Duration int64  `json:"duration"`
	}
	a, _ := json.Marshal(&OneBotV12Command{
		Action: "ban_group_member",
		Params: P{
			groupId, userId, last,
		},
	})
	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterWalleQ) MemberKick(groupId string, userId string) {
	type P struct {
		GroupId string `json:"groupId"`
		UserId  string `json:"userId"`
	}
	a, _ := json.Marshal(&OneBotV12Command{
		Action: "kick_group_member",
		Params: P{
			groupId, userId,
		},
	})
	socketSendText(pa.Socket, string(a))
}

/* 扩展方法实现 */

func (pa *PlatformAdapterWalleQ) waitGroupMemberInfoEcho(echo string, beforeWait func()) *EventWalleQBase {
	//pa.echoList = append(pa.echoList, )
	ch := make(chan *EventWalleQBase, 1)

	if pa.echoMap == nil {
		pa.echoMap = new(SyncMap[string, chan *EventWalleQBase])
	}
	pa.echoMap.Store(echo, ch)

	beforeWait()
	return <-ch
}

func (pa *PlatformAdapterWalleQ) waitEcho2(echo string, value interface{}, beforeWait func(emi *echoMapInfo)) error {
	if pa.echoMap2 == nil {
		pa.echoMap2 = new(SyncMap[string, *echoMapInfo])
	}

	emi := &echoMapInfo{ch: make(chan string, 1)}
	beforeWait(emi)

	pa.echoMap2.Store(echo, emi)
	val := <-emi.ch
	if val == "" {
		return errors.New("超时")
	}
	return json.Unmarshal([]byte(val), value)
}

// SendMessage 原始的发消息 API
func (pa *PlatformAdapterWalleQ) SendMessage(text string, ty string, id string, cid string) {
	type Params struct {
		DetailType string           `json:"detail_type"`
		GroupId    string           `json:"group_id,omitempty"`
		UserId     string           `json:"user_id,omitempty"`
		GuildId    string           `json:"guild_id,omitempty"`
		ChannelId  string           `json:"channel_id,omitempty"`
		Message    []MessageSegment `json:"message"`
	}
	var (
		uid  string
		gid  string
		g2id string // 2级群组 wq 未实现
	)
	switch ty {
	case "private":
		uid = id
	case "group":
		gid = id
	case "channel":
		g2id = id
	}
	a, _ := json.Marshal(OneBotV12Command{
		Action: "send_message",
		Echo:   "send_message",
		Params: Params{
			DetailType: ty,
			GroupId:    gid,
			UserId:     uid,
			GuildId:    g2id,
			ChannelId:  cid,
			Message:    pa.TextToMessageSegment(text),
		},
	})
	socketSendText(pa.Socket, string(a))
}

// GetLoginInfo 获取登录号信息
func (pa *PlatformAdapterWalleQ) GetLoginInfo() {
	a, _ := json.Marshal(OneBotV12Command{
		Action: "get_self_info",
		Echo:   "get_self_info",
		Params: &struct{}{},
	})
	socketSendText(pa.Socket, string(a))
}

// GetGroupMemberInfo 获取群成员信息
func (pa *PlatformAdapterWalleQ) GetGroupMemberInfo(GroupId string, UserId string) *OnebotV12UserInfo {
	type DetailParams struct {
		GroupId string `json:"group_id"`
		UserId  string `json:"user_id"`
	}

	echo := "get_group_member_info"
	a, _ := json.Marshal(OneBotV12Command{
		Action: "get_group_member_info",
		Params: DetailParams{
			GroupId: GroupId,
			UserId:  UserId,
		},
		Echo: echo,
	})

	d := pa.waitGroupMemberInfoEcho(echo, func() {
		socketSendText(pa.Socket, string(a))
	})

	return &OnebotV12UserInfo{
		Nickname: d.UserName,
		UserId:   d.UserId,
		GroupId:  d.GroupId,
		Card:     d.UserCard,
	}
}

// SetGroupAddRequest 同意加群
func (pa *PlatformAdapterWalleQ) SetGroupAddRequest(rid int64, gid string, accept bool) {
	type DetailParams struct {
		ReqId  int64  `json:"request_id"`
		UserId string `json:"user_id"`
		Accept bool   `json:"accept"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "set_group_invited",
		Params: DetailParams{
			rid, gid, accept,
		},
	})

	socketSendText(pa.Socket, string(a))
}

// SetFriendAddRequest 同意好友
func (pa *PlatformAdapterWalleQ) SetFriendAddRequest(reqId int64, userId string, accept bool) {
	type DetailParams struct {
		ReqId  int64  `json:"request_id"`
		UserId string `json:"user_id"`
		Accept bool   `json:"accept"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_new_friend",
		DetailParams{
			reqId, userId, accept,
		},
	})

	socketSendText(pa.Socket, string(a))
}

// DeleteFriend 删除好友
func (pa *PlatformAdapterWalleQ) DeleteFriend(ctx *MsgContext, id string) {
	friendId, type_ := pa.mustExtractId(id)
	if type_ != QQUidPerson {
		return
	}
	type GroupMessageParams struct {
		UserId string `json:"user_id"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "delete_friend",
		Params: GroupMessageParams{
			friendId,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterWalleQ) UpVoiceFile(path string) {
	type P struct {
		Type     string `json:"type"`
		Path     string `json:"path,omitempty"`
		FileType string `json:"file_type"`
	}
	a, _ := json.Marshal(OneBotV12Command{
		Action: "upload_file",
		Echo:   "upload_file_voice",
		Params: P{"path", path, "voice"},
	})
	socketSendText(pa.Socket, string(a))
}

/** 格式化与反格式化 */

func MessageSegmentToText(ms []MessageSegment) string {
	var s string
	for _, i := range ms {
		switch i.Type {
		case "text":
			s = s + i.Data.Text
		case "mention":
			cq := CQCommand{
				Type: "at",
				Args: map[string]string{"qq": i.Data.UserId},
			}
			s = s + cq.Compile()
		case "mention_all":
			cq := CQCommand{
				Type: "at",
				Args: map[string]string{"qq": "all"},
			}
			s = s + cq.Compile()
		case "image":
			cq := CQCommand{
				Type: "image",
				Args: map[string]string{"file": i.Data.Url},
			}
			s = s + cq.Compile()
		case "voice":
			cq := CQCommand{
				Type: "voice",
				Args: map[string]string{"file": i.Data.FileId},
			}
			s = s + cq.Compile()
		case "audio": // 这个是音频文件 wq 未支持 or QQ 没有
		case "video":
			cq := CQCommand{
				Type: "voice",
				Args: map[string]string{"file": i.Data.FileId},
			}
			s = s + cq.Compile()
		case "file":
		case "location":
		case "reply":
			cq := CQCommand{
				Type: "reply",
				Args: map[string]string{"user_id": i.Data.UserId, "message_id": i.Data.MessageId},
			}
			s = s + cq.Compile()
		}
	}
	return s
}

func (pa *PlatformAdapterWalleQ) TextToMessageSegment(text string) []MessageSegment {
	arr := getCqCodePairs(text)
	var m []MessageSegment
	// 解析 cq 参数
	f := func(input string) map[string]string {
		pairs := strings.Split(input, ",")
		params := make(map[string]string)
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			params[kv[0]] = kv[1]
		}
		return params
	}
	// 路径转换
	f2 := func(path string) string {
		if strings.HasPrefix(path, "http") || strings.HasPrefix(path, "base64") || strings.HasPrefix(path, "file:///") {
			return path
		}
		if filepath.IsAbs(path) {
			return "file:///" + path
		} else {
			pa2, err := filepath.Abs(path)
			if err != nil {
				pa.Session.Parent.Logger.Info("路径转换错误，将使用原路径", err)
				pa2 = path
			}
			return pa2
		}
	}
	// 海豹码 转 cq 码 // 有点麻
	for _, i := range arr {
		if strings.HasPrefix(i, "[") { // 是 [ 开头的就转一下
			i = SealCodeToCqCode(i)
		}
		isCq := strings.HasPrefix(i, "[CQ")
		if isCq {
			i = i[4 : len(i)-1]
			j := strings.Index(i, ",")
			cqT := i[:j]
			i = i[:j]
			cqP := f(i)
			switch cqT {
			case "at":
				p := cqP["qq"]
				if p == "all" {
					m = append(m, MessageSegment{Type: "mention_all", Data: MSData{}})
				} else {
					m = append(m, MessageSegment{Type: "mention", Data: MSData{UserId: p}})
				}
				continue
			case "image":
				p := cqP["file"]
				p = f2(p)
				m = append(m, MessageSegment{"image", MSData{Url: p}})
				continue
			case "reply":
				p := cqP["id"]
				m = append(m, MessageSegment{"reply", MSData{MessageId: p}})
				continue
			case "record":
				continue
			}
		}
	}
	return m
}

func (event *EventWalleQBase) toMessageBase() *Message {
	msg := new(Message)
	msg.Time = int64(event.Time)
	msg.MessageType = event.DetailType
	msg.Platform = "QQ"
	switch event.DetailType {
	case "private":
		msg.Sender.UserId = FormatDiceIdQQV12(event.UserId)
		msg.Sender.Nickname = event.UserName
		msg.GroupId = "PG-QQ:" + event.UserId
	case "group":
		msg.Sender.UserId = FormatDiceIdQQV12(event.UserId)
		msg.Sender.Nickname = event.UserCard
		msg.GroupId = FormatDiceIdQQGroupV12(event.GroupId)
	case "channel": // wq 未实现
		msg.Sender.UserId = FormatDiceIdQQCh(event.UserId)
		msg.Sender.Nickname = event.UserName
		msg.GroupId = FormatDiceIdQQChGroup(event.ChannelId, event.GuildId) // v12 与海豹标准相反，好别扭啊
	}
	return msg
}

// FormatDiceIdQQV12 QQ:122
func FormatDiceIdQQV12(diceWalleQ string) string {
	return fmt.Sprintf("QQ:%s", diceWalleQ)
}

// FormatDiceIdQQGroupV12 QQ-Group:122
func FormatDiceIdQQGroupV12(diceWalleQ string) string {
	return fmt.Sprintf("QQ-Group:%s", diceWalleQ)
}

func (pa *PlatformAdapterWalleQ) mustExtractId(id string) (string, QQUidType) {
	if strings.HasPrefix(id, "QQ:") {
		return id[len("QQ:"):], QQUidPerson
	}
	if strings.HasPrefix(id, "QQ-Group:") {
		return id[len("QQ-Group:"):], QQUidGroup
	}
	if strings.HasPrefix(id, "PG-QQ:") {
		return id[len("PG-QQ:"):], QQUidPerson
	}
	return "", 0
}

func getCqCodePairs(text string) []string {
	var res []string // 我感觉木落写过类似的函数，比如牌堆里，但是我没找到（
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '[' {
			if start != i {
				res = append(res, text[start:i])
			}
			end := strings.Index(text[i:], "]")
			if end == -1 {
				break
			}
			res = append(res, text[i:i+end+1])
			i += end
			start = i + 1
		}
	}
	if start < len(text) {
		res = append(res, text[start:])
	}
	return res
}
