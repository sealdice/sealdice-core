package dice

import (
	"context"
	"strings"

	"github.com/PaienNate/SealSocketIO/socketio"
	"github.com/ThreeDotsLabs/watermill-bolt/pkg/bolt"
	waterMQ "github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/maypok86/otter"
	"github.com/orbs-network/govnr"
	"github.com/tidwall/gjson"

	"sealdice-core/message"
	log "sealdice-core/utils/kratos"
)

// PlatformAdapterPureOnebot11
// 个人思路上讲，不考虑将其序列化成对应的结构体，通过目前不同的实现可以发现，序列化的没有他们改的快。
// 转而是考虑使用gjson + sjson进行解析。
// 当需要开发新功能的时候，打断点或者直接输出展示或许是更快的选择。
// 预期解决如下ISSUE：
// 1. https://github.com/sealdice/sealdice-core/issues/78 控制加好友的速度 第一版采用5S加一个好友的设计
// 2. https://github.com/sealdice/sealdice-core/issues/950 骰娘是群主时，考虑不能退群
// 3. 私骰

type PlatformAdapterPureOnebot11 struct {
	govnr.TreeSupervisor `yaml:"-" json:"-"`
	Publisher            *bolt.Publisher          `yaml:"-" json:"-"`
	Subscriber           *bolt.Subscriber         `yaml:"-" json:"-"`
	MQRouter             *waterMQ.Router          `yaml:"-" json:"-"`
	Instance             *socketio.SocketInstance `yaml:"-" json:"-"`
	Logger               *log.Helper              `yaml:"-" json:"-"` // 自带的LOGGER
	GoVNRErrorLogger     *log.GoVNRErrorer        `yaml:"-" json:"-"` // 封装的GOVNR LOGGER
	// 特有的数据 -> 考虑一下，将它分开！
	ConnectURL                   string   `yaml:"connectUrl" json:"connectUrl"`   // 连接地址
	AccessToken                  string   `yaml:"accessToken" json:"accessToken"` // 访问令牌
	OnebotState                  int      `yaml:"-" json:"loginState"`            // 当前状态
	InPackGoCqhttpDisconnectedCH chan int `yaml:"-" json:"-"`                     // 信号量，用于关闭连接
	IsReverse                    bool     `yaml:"isReverse" json:"isReverse" `    // 是否是反向（服务端连接）
	ReverseAddr                  string   `yaml:"reverseAddr" json:"reverseAddr"` // 反向时，反向绑定的地址
	// 连接模式
	Mode LinkerMode `yaml:"-" json:"linkerMode"`
	// 发送使用的UUID TODO
	UUID string `yaml:"uuid" json:"uuid"`
	// App版本信息，连上获取。可以用来做判断用。不保证一定能拿到，没有就是空。
	AppVersion AppVersionInfo `yaml:"-" json:"appVersion"`
	// 群缓存信息 过期时间考虑设置为 5 分钟。
	GroupInfoCache *otter.Cache[string, gjson.Result] `yaml:"-" json:"-"`
	// 群Details信息 过期时间 5 分钟

	IgnoreFriendRequest bool `yaml:"ignoreFriendRequest" json:"ignoreFriendRequest"` // 忽略好友请求处理开关
	// 一般做处理用的到的公共数据 TMD，这玩意怎么在另外的地方赋值的？
	Session     *IMSession    `yaml:"-" json:"-"`
	EndPoint    *EndPointInfo `yaml:"-" json:"-"`
	DiceServing bool          `yaml:"-"`
}

// PreloadInstance 初始化Onebot适合的绑定关系
func (p *PlatformAdapterPureOnebot11) PreloadInstance() error {
	p.Instance = socketio.NewSocketInstance()
	// TODO: 连接绑定
	// 总接收事件
	p.Instance.On(socketio.EventMessage, p.serveOnebotEvent)
	// 分发会变成下面的消息
	// 上报类型为五类消息，绑定五种事件
	p.Instance.On(OnebotEventPostTypeMessage, p.onOnebotMessageEvent)
	// GOCQ 特有的自身消息，目前没啥用
	// p.Instance.On(OnebotEventPostTypeMessageSent, nil)
	p.Instance.On(OnebotEventPostTypeRequest, p.onOnebotRequestEvent)
	// p.Instance.On(OnebotEventPostTypeNotice, nil)
	// p.Instance.On(OnebotEventPostTypeMetaEvent, nil)
	// // 响应类型为一种事件 websocket可以拿到action
	p.Instance.On(OnebotReceiveMessage, p.onCustomReplyEvent)
	// TODO：针对断联的处理 -> 暂时先不做
	return nil
}

func (p *PlatformAdapterPureOnebot11) Serve() int {
	// 初始化bolt以及对应的Router
	err := p.PreloadInstance()
	if err != nil {
		return -1
	}
	// 添加处理器
	handleFunc := middleware.NewThrottle(1, 5).Middleware
	friendAddHandler := p.MQRouter.AddNoPublisherHandler(TopicHandleAddNewFriends, TopicHandleAddNewFriends, p.Subscriber, p.messageQueueOnFriendAdd)
	groupAddHandler := p.MQRouter.AddNoPublisherHandler(TopicHandleInviteToGroup, TopicHandleInviteToGroup, p.Subscriber, p.messageQueueOnGroupAdd)
	friendAddHandler.AddMiddleware(handleFunc)
	groupAddHandler.AddMiddleware(handleFunc)
	var handle *govnr.ForeverHandle
	// 如果是反向服务器的初始化
	if p.IsReverse {
		handle = govnr.Forever(context.Background(), "ReverseHttpServer", nil, func() {
			var upgrader = websocket.Upgrader{}
			e := echo.New()
			// TODO: 提供修改的渠道
			e.GET("/ws", func(c echo.Context) error {
				// Upgrade to WebSocket
				conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
				if err != nil {
					log.Error("upgrade:", err)
					return err
				}

				handler := p.Instance.New(func(kws *socketio.WebsocketWrapper) {
					// 获取连接的信息
					p.UUID = kws.UUID
					p.GetVersionInfo()
					p.GetLoginInfo()
					// 设置连接状态为 已连接
					p.EndPoint.State = 1
				}, conn)

				// Since Echo handles the request/response cycle differently,
				// we need to call the handler directly
				handler(c.Response(), c.Request())

				return nil
			})

			log.Fatal(e.Start("0.0.0.0:3002"))
		})
	} else {
		// TODO 正向TODO
	}
	// 管理连接的handle，并在Fatal时自动重新回复
	p.Supervise(handle)
	return 0
}

func (p *PlatformAdapterPureOnebot11) DoRelogin() bool {
	// 什么都不做，我们无法控制重新登录
	return true
}

func (p *PlatformAdapterPureOnebot11) SetEnable(enable bool) {
	// 设置是否启用 通过通断控制
}

func (p *PlatformAdapterPureOnebot11) QuitGroup(ctx *MsgContext, ID string) {
	// 退群
}
func (p *PlatformAdapterPureOnebot11) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
}

func (p *PlatformAdapterPureOnebot11) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	// 给群发消息
}

func (p *PlatformAdapterPureOnebot11) SetGroupCardName(ctx *MsgContext, name string) {
	// 发送群卡片消息？
}

func (p *PlatformAdapterPureOnebot11) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	// 什么J8？
}

func (p *PlatformAdapterPureOnebot11) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	// 什么J8？
}

func (p *PlatformAdapterPureOnebot11) SendFileToPerson(ctx *MsgContext, userID string, path string, flag string) {
	// 给人发文件
}

func (p *PlatformAdapterPureOnebot11) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	// 给组发文件
}

func (p *PlatformAdapterPureOnebot11) MemberBan(groupID string, userID string, duration int64) {
	// Ban人？
}

func (p *PlatformAdapterPureOnebot11) MemberKick(groupID string, userID string) {
	// Ban人？
}

func (p *PlatformAdapterPureOnebot11) GetGroupInfoAsync(groupID string) {
	// 第一步：切分，看看有没有需要发送的数据
	requireList := strings.Split(groupID, "@")
	topicName := ""
	// 切分出对应的Topic 什么的 放在 echo 里
	if len(requireList) == 2 {
		groupID = requireList[0]
		topicName = requireList[1]
	}
	type GroupMessageParams struct {
		GroupID int64 `json:"group_id"`
	}
	realGroupID, idType := p.mustExtractID(groupID)
	if idType != OBQQUidGroup {
		return
	}
	var echo = ""
	if topicName != "" {
		echo = GetGroupInfo + "@" + topicName
	} else {
		echo = GetGroupInfo
	}
	resp := oneBotCommand{
		GetGroupInfo,
		GroupMessageParams{
			realGroupID,
		},
		echo,
	}
	a, _ := sonic.Marshal(resp)
	err := p.Instance.EmitTo(p.UUID, a, socketio.TextMessage)
	if err != nil {
		p.Logger.Error("发送获取GroupInfo Onebot消息出现异常: %v", err)
		return
	}
}

func (p *PlatformAdapterPureOnebot11) EditMessage(ctx *MsgContext, msgID, message string) {
	// 改信息？
}

func (p *PlatformAdapterPureOnebot11) RecallMessage(ctx *MsgContext, msgID string) {
	// 重新发信息？
}
