package dice

import (
	"encoding/json"
	"fmt"
	"github.com/fy0/procs"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type PlatformAdapterQQOnebot struct {
	EndPoint *EndPointInfo `yaml:"-" json:"-"`
	Session  *IMSession    `yaml:"-" json:"-"`

	Socket     *gowebsocket.Socket `yaml:"-" json:"-"`
	ConnectUrl string              `yaml:"connectUrl" json:"connectUrl"` // 连接地址

	UseInPackGoCqhttp                bool           `yaml:"useInPackGoCqhttp" json:"useInPackGoCqhttp"` // 是否使用内置的gocqhttp
	InPackGoCqLastAutoLoginTime      int64          `yaml:"inPackGoCqLastAutoLoginTime" json:"-"`       // 上次自动重新登录的时间
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

type Sender struct {
	Age      int32  `json:"age"`
	Card     string `json:"card"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"` // owner 群主
	UserId   int64  `json:"user_id"`
}

type MessageQQ struct {
	MessageId     int64   `json:"message_id"`   // QQ信息此类型为int64，频道中为string
	MessageType   string  `json:"message_type"` // Group
	Sender        *Sender `json:"sender"`       // 发送者
	RawMessage    string  `json:"raw_message"`
	Message       string  `json:"message"` // 消息内容
	Time          int64   `json:"time"`    // 发送时间
	MetaEventType string  `json:"meta_event_type"`
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
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	//Status string `json:"status"`
	Echo int `json:"echo"`
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
	msg.RawId = msgQQ.MessageId
	msg.Platform = "QQ"

	if msgQQ.Data != nil && msgQQ.Data.GroupId != 0 {
		msg.GroupId = FormatDiceIdQQGroup(msgQQ.Data.GroupId)
	}
	if msgQQ.GroupId != 0 {
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
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	disconnected := make(chan int, 1)
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
		log.Info("Recieved connect error: ", err)
		fmt.Println("连接失败")
		disconnected <- 2
	}

	// {"channel_id":"3574366","guild_id":"51541481646552899","message":"说句话试试","message_id":"BAC3HLRYvXdDAAAAAAA2il4AAAAAAAAABA==","message_type":"guild","post_type":"mes
	//sage","self_id":2589922907,"self_tiny_id":"144115218748146488","sender":{"nickname":"木落","tiny_id":"222","user_id":222},"sub_type":"channel",
	//"time":1647386874,"user_id":"144115218731218202"}

	// 疑似消息发送成功？等等 是不是可以用来取一下log
	// {"data":{"message_id":-1541043078},"retcode":0,"status":"ok"}
	var lastWelcome *LastWelcomeInfo

	tempInviteMap := map[string]int64{}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		//fmt.Println("!!!", message)
		if strings.Contains(message, `"channel_id"`) {
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
				disconnected <- 3
			}

			msg := msgQQ.toStdMessage()
			ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: session, Dice: session.Parent}

			// 获得用户信息
			if msgQQ.Echo == -1 {
				ep.Nickname = msgQQ.Data.Nickname
				ep.UserId = FormatDiceIdQQ(msgQQ.Data.UserId)

				log.Debug("骰子信息已刷新")
				ep.GroupNum = int64(len(session.ServiceAtNew))
				return
			}

			// 获得群信息
			if msgQQ.Echo == -2 {
				if msgQQ.Data != nil {
					groupId := FormatDiceIdQQGroup(msgQQ.Data.GroupId)
					group := session.ServiceAtNew[groupId]
					if group != nil {
						if msgQQ.Data.MaxMemberCount == 0 {
							// 试图删除自己
							diceId := ep.UserId
							if _, exists := group.DiceIds[diceId]; exists {
								// 删除自己的登记信息
								delete(group.DiceIds, diceId)

								if len(group.DiceIds) == 0 {
									// 如果该群所有账号都被删除了，那么也删掉整条记录
									// TODO: 该群下的用户信息实际没有被删除
									delete(session.ServiceAtNew, msg.GroupId)
								}
							}
						} else {
							// 更新群名
							group.GroupName = msgQQ.Data.GroupName
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

			// 处理加群请求
			if msgQQ.PostType == "request" && msgQQ.RequestType == "group" && msgQQ.SubType == "invite" {
				// {"comment":"","flag":"111","group_id":222,"post_type":"request","request_type":"group","self_id":333,"sub_type":"invite","time":1646782195,"user_id":444}
				ep.GroupNum = int64(len(session.ServiceAtNew))

				txt := fmt.Sprintf("收到QQ加群邀请: 群组(%d) 邀请人:%d", msgQQ.GroupId, msgQQ.UserId)
				log.Info(txt)
				ctx.Notice(txt)
				tempInviteMap[msg.GroupId] = time.Now().Unix()
				time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))
				pa.SetGroupAddRequest(msgQQ.Flag, msgQQ.SubType, true, "")
				return
			}

			// 好友请求
			if msgQQ.PostType == "request" && msgQQ.RequestType == "friend" {
				// {"comment":"123","flag":"1647619872000000","post_type":"request","request_type":"friend","self_id":222,"time":1647619871,"user_id":111}
				var comment string
				if msgQQ.Comment != "" {
					comment = strings.TrimSpace(msgQQ.Comment)
					comment = strings.ReplaceAll(comment, "\u00a0", "")
				}

				willAccept := comment == DiceFormat(ctx, strings.TrimSpace(session.Parent.FriendAddComment))

				if comment == "" {
					comment = "(无)"
				} else {
					comment = strconv.Quote(comment)
				}

				txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%d, 验证信息: %s, 是否自动同意: %t", msgQQ.UserId, comment, willAccept)
				log.Info(txt)
				ctx.Notice(txt)
				time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

				if willAccept {
					pa.SetFriendAddRequest(msgQQ.Flag, true, "", "")
				} else {
					pa.SetFriendAddRequest(msgQQ.Flag, false, "", "验证信息不符")
				}
				return
			}

			// 好友通过后
			if msgQQ.NoticeType == "friend_add" && msgQQ.PostType == "post_type" {
				// {"notice_type":"friend_add","post_type":"notice","self_id":222,"time":1648239248,"user_id":111}
				go func() {
					// 稍作等待后发送入群致词
					time.Sleep(2 * time.Second)
					pa.SendToPerson(ctx, msg.Sender.UserId, DiceFormatTmpl(ctx, "核心:骰子成为好友"), "")
				}()
				return
			}

			groupEntered := func() {
				// 判断进群的人是自己，自动启动
				SetBotOnAtGroup(ctx, msg.GroupId)
				// 立即获取群信息
				pa.GetGroupInfoAsync(msg.GroupId)
				// fmt.Sprintf("<%s>已经就绪。可通过.help查看指令列表", conn.Nickname)
				go func() {
					// 稍作等待后发送入群致词
					time.Sleep(2 * time.Second)
					pa.SendToGroup(ctx, msg.GroupId, DiceFormatTmpl(ctx, "核心:骰子进群"), "")
				}()
				txt := fmt.Sprintf("加入QQ群组: (%d)", msgQQ.GroupId)
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
							//VarSetValueStr(ctx, "$t新人昵称", "<"+msgQQ.Sender.Nickname+">")
							pa.SendToGroup(ctx, msg.GroupId, DiceFormat(ctx, group.GroupWelcomeMessage), "")
						}
					}
				}
				return
			}

			if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_decrease" && msgQQ.SubType == "kick_me" {
				// 被踢
				//  {"group_id":111,"notice_type":"group_decrease","operator_id":222,"post_type":"notice","self_id":333,"sub_type":"kick_me","time":1646689414 ,"user_id":333}
				if msgQQ.UserId == msgQQ.SelfId {
					txt := fmt.Sprintf("被踢出群: 在QQ群组(%d)中被踢出，操作者:(%d)", msgQQ.GroupId, msgQQ.UserId)
					log.Info(txt)
					ctx.Notice(txt)
				}
				return
			}

			if msgQQ.PostType == "notice" && msgQQ.NoticeType == "group_ban" && msgQQ.SubType == "ban" {
				// 禁言
				// {"duration":600,"group_id":111,"notice_type":"group_ban","operator_id":222,"post_type":"notice","self_id":333,"sub_type":"ban","time":1646689567,"user_id":333}
				if msgQQ.UserId == msgQQ.SelfId {
					txt := fmt.Sprintf("被禁言: 在群组(%d)中被禁言，时长%d秒，操作者:(%d)", msgQQ.GroupId, msgQQ.Duration, msgQQ.UserId)
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
						_ = LogMarkDeleteByMsgId(ctx, group, msgQQ.MessageId)
					}
				}
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
		disconnected <- 1
	}

	socket.Connect()
	defer func() {
		fmt.Println("socket close")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("关闭连接时遭遇异常")
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
			disconnected <- 0
			return 0
		case val := <-disconnected:
			return val
		}
	}
}

func (pa *PlatformAdapterQQOnebot) DoRelogin() bool {
	myDice := pa.Session.Parent
	ep := pa.EndPoint
	if pa.UseInPackGoCqhttp {
		myDice.Logger.Infof("重新启动go-cqhttp进程，对应账号: <%s>(%s)", ep.Nickname, ep.UserId)
		go GoCqHttpServeProcessKill(myDice, ep)
		time.Sleep(5 * time.Second)                 // 上面那个清理有概率卡住，具体不懂，改成等5s
		GoCqHttpServeRemoveSessionToken(myDice, ep) // 删除session.token
		pa.InPackGoCqHttpLastRestrictedTime = 0     // 重置风控时间
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
			go DiceServe(d, c)
		} else {
			go DiceServe(d, c)
		}
	} else {
		c.Enable = false
		pa.DiceServing = false
		if pa.UseInPackGoCqhttp {
			GoCqHttpServeProcessKill(d, c)
		}
	}
}
