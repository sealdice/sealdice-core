package dice

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PaienNate/SealSocketIO/socketio"
	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"sealdice-core/message"
	log "sealdice-core/utils/kratos"
)

// 个人思路上讲，不考虑将其序列化成对应的结构体，通过目前不同的实现可以发现，序列化的没有他们改的快。
// 转而是考虑使用gjson + sjson进行解析。
// 当需要开发新功能的时候，打断点或者直接输出展示或许是更快的选择。

type PlatformAdapterPureOnebot11 struct {
	// TODO: 缺少TreeSupervisor和GracefulShutdown的实现，得等govnr
	Instance *socketio.SocketInstance `yaml:"-" json:"-"`
	Logger   *log.Helper              `yaml:"-" json:"-"` // 自带的LOGGER
	// 特有的数据 -> 考虑一下，将它分开！
	ConnectURL                   string   `yaml:"connectUrl" json:"connectUrl"`   // 连接地址
	AccessToken                  string   `yaml:"accessToken" json:"accessToken"` // 访问令牌
	OnebotState                  int      `yaml:"-" json:"loginState"`            // 当前状态
	InPackGoCqhttpDisconnectedCH chan int `yaml:"-" json:"-"`                     // 信号量，用于关闭连接
	IsReverse                    bool     `yaml:"isReverse" json:"isReverse" `    // 是否是反向（服务端连接）
	ReverseAddr                  string   `yaml:"reverseAddr" json:"reverseAddr"` // 反向时，反向绑定的地址
	// 连接模式
	Mode LinkerMode `yaml:"-" json:"linkerMode"`
	// App版本信息，连上获取。可以用来做判断用。不保证一定能拿到，没有就是空。
	AppVersion AppVersionInfo `yaml:"-" json:"appVersion"`

	IgnoreFriendRequest bool `yaml:"ignoreFriendRequest" json:"ignoreFriendRequest"` // 忽略好友请求处理开关
	// 一般做处理用的到的公共数据 TMD，这玩意怎么在另外的地方赋值的？
	Session     *IMSession    `yaml:"-" json:"-"`
	EndPoint    *EndPointInfo `yaml:"-" json:"-"`
	DiceServing bool          `yaml:"-"`
	FuckYouFlag string        `yaml:"FuckYouFlag" json:"FuckYouFlag"`
	// 一些本来在MsgContext里，但是实在不知道这有个什么B用的东西
}

// TODO: 最后需要做耗子大搬家活动

type AppVersionInfo struct {
	AppName         string `json:"app_name"`
	ProtocolVersion string `json:"protocol_version"`
	AppVersion      string `json:"app_version"`
}

type LinkerMode string

// linkermode分为string和array
const (
	LinkerModeString LinkerMode = "string"
	LinkerModeArray  LinkerMode = "array"
)

// 这里需要一个ONEBOT的事件对应关系码表
// OnebotEventPostTypeCode
const (
	// 这里有一个算是坑的地方吧，他们的数据默认的什么ping啊，pong啊居然是明文的……这个感觉得改改。
	OnebotEventPostTypeMessage = "onebot_message"
	// OnebotEventPostTypeCodeMessageSent 是bot发出的消息
	OnebotEventPostTypeMessageSent = "onebot_message_sent"
	OnebotEventPostTypeRequest     = "onebot_request"
	OnebotEventPostTypeNotice      = "onebot_notice"
	OnebotEventPostTypeMetaEvent   = "onebot_meta_event"
)

const (
	OnebotReceiveMessage = "onebot_echo"
)

// 定义参数
// 原则上说，echo信息应该都有。
const (
	// GetLoginInfo 获取登录信息
	GetLoginInfo = "get_login_info"
	// GetGroupInfo 获取群信息
	GetGroupInfo = "get_group_info"
	// 获取版本信息
	GetVersionInfo = "get_version_info"
)

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
	//p.Instance.On(OnebotEventPostTypeRequest, nil)
	//p.Instance.On(OnebotEventPostTypeNotice, nil)
	//p.Instance.On(OnebotEventPostTypeMetaEvent, nil)
	//// 响应类型为一种事件
	//p.Instance.On(OnebotReceiveMessage, nil)
	// TODO：针对断联的处理 -> 暂时先不做
	return nil
}

// 需要获取Echo数据的函数
func (pa *PlatformAdapterPureOnebot11) GetLoginInfo() {
	a, _ := sonic.Marshal(struct {
		Action string `json:"action"`
		Echo   string `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   GetLoginInfo,
	})
	fmt.Println(a)
	// socketSendText(pa.Socket, string(a))
}

// 消息处理函数们

// request 类型
func (p *PlatformAdapterPureOnebot11) onOnebotRequestEvent(ep *socketio.EventPayload) {
	// 请求分为好友请求和加群/邀请请求
	// TODO: 以此为例，我们考虑一下，要不要err?
	req := gjson.ParseBytes(ep.Data)
	switch req.Get("request_type").String() {
	case "friend":
		_ = p.handleReqFriendAction(req, ep)
	case "group":
		_ = p.handleReqGroupAction(req, ep)
	}
}

// 编写一个检查是否在黑名单的函数体
func (p *PlatformAdapterPureOnebot11) checkPassBlackList(userId string, ctx *MsgContext) bool {
	uid := FormatDiceIDQQ(userId)
	banInfo, ok := ctx.Dice.Config.BanList.GetByID(uid)
	if ok {
		if banInfo.Rank == BanRankBanned && ctx.Dice.Config.BanList.BanBehaviorRefuseInvite {
			return false
		}
	}
	return true
}

// 编写一个多验证消息检查函数
func (p *PlatformAdapterPureOnebot11) checkMultiFriendAddVerify(comment string, toMatch string) bool {
	// 根据GPT的描述，这里干的事情是：从评论中提取回答内容，并与目标字符串进行逐项匹配，最终决定是否接受。
	// 我只是从木落那里拆了过来，太热闹了。
	var willAccept bool
	re := regexp.MustCompile(`\n回答:([^\n]+)`)
	m := re.FindAllStringSubmatch(comment, -1)
	// 要匹配的是空，说明不验证
	if toMatch == "" {
		willAccept = true
		return willAccept
	}
	var items []string
	for _, i := range m {
		items = append(items, i[1])
	}

	re2 := regexp.MustCompile(`\s+`)
	m2 := re2.Split(toMatch, -1)

	if len(m2) == len(items) {
		ok := true
		for i := range m2 {
			if m2[i] != items[i] {
				ok = false
				break
			}
		}
		willAccept = ok
	}
	return willAccept
}

func (p *PlatformAdapterPureOnebot11) handleReqFriendAction(req gjson.Result, ep *socketio.EventPayload) error {
	// 只有一种情况 就是好友添加
	// 获取请求详情
	var comment string
	if req.Get("comment").Exists() {
		comment = strings.TrimSpace(req.Get("comment").String())
		comment = strings.ReplaceAll(comment, "\u00a0", "")
	}
	// 将匹配的验证问题
	toMatch := strings.TrimSpace(p.Session.Parent.Config.FriendAddComment)
	// 创建虚构MsgContext
	ctx := &MsgContext{MessageType: req.Get("message_type").String(), EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	var extra string
	// 匹配验证问题检查
	var passQuestion bool
	var passblackList bool
	if comment != DiceFormat(ctx, toMatch) {
		passQuestion = p.checkMultiFriendAddVerify(comment, toMatch)
	}
	// 匹配黑名单检查
	passblackList = p.checkPassBlackList(req.Get("user_id").String(), ctx)
	// 格式化请求的数据
	comment = strconv.Quote(comment)
	if comment == "" {
		comment = "(无)"
	}
	if !passQuestion {
		extra = "。回答错误"
	} else {
		extra = "。回答正确"
	}
	if !passblackList {
		extra += "。（被禁止用户）"
	}
	if p.IgnoreFriendRequest {
		extra += "。由于设置了忽略邀请，此信息仅为通报"
	}

	txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", req.Get("user_id").String(), comment, passQuestion && passblackList, extra)
	log.Info(txt)
	ctx.Notice(txt)
	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))
	if passQuestion && passblackList {
		p.SetFriendAddRequest(req.Get("flag").String(), true, "", "", ep)
	} else {
		p.SetFriendAddRequest(req.Get("flag").String(), false, "", "验证信息不符或在黑名单", ep)
	}
	return nil
}

func (p *PlatformAdapterPureOnebot11) handleReqGroupAction(req gjson.Result, ep *socketio.EventPayload) error {
	return nil
}

// MESSAGE 类型
func (p *PlatformAdapterPureOnebot11) onOnebotMessageEvent(ep *socketio.EventPayload) {
	// 进行进一步的下一层解析，从而获取值
	// 如果是普通消息(群消息/私聊消息)
	// Note(Pinenutn): 我真是草了，这个Execute又是依托答辩，代码长的完全不忍阅读
	// 总之分发到这里的数据肯定都能序列化成Message的罢，对的罢？
	msg := p.convertStringMessage(gjson.ParseBytes(ep.Data))
	if msg.MessageType == "private" || msg.MessageType == "group" {
		msg.UUID = ep.SocketUUID
		p.Session.Execute(p.EndPoint, msg, false)
	}
	// DO NOTHING
}

func (p *PlatformAdapterPureOnebot11) serveOnebotEvent(ep *socketio.EventPayload) {
	fmt.Printf("Message event - User: %s - Message: %s", ep.Kws.GetStringAttribute("user_id"), string(ep.Data))
	var err error
	if !gjson.ValidBytes(ep.Data) {
		// TODO：错误处理
		return
	}
	resp := gjson.ParseBytes(ep.Data)
	// 解析是string还是array
	// TODO: 逻辑正确吗？
	switch resp.Get("message").Type {
	case gjson.String:
		p.Mode = LinkerModeString
	case gjson.JSON:
		p.Mode = LinkerModeArray
	default:
		p.Mode = LinkerModeString
	}
	if p.Mode == LinkerModeArray {
		resp, err = p.parseOB11ArrayToStringMessage(resp)
		if err != nil {
			// 日志
			return
		}
	}
	// 解析终了，进行分发
	eventType := resp.Get("post_type").String()
	if eventType != "" {
		// 分发事件
		eventType = fmt.Sprintf("onebot_%s", eventType)
		ep.Kws.Fire(eventType, []byte(resp.String()))
	} else {
		// 如果没有post_type，说明不是上报信息，而是API的返回信息
		ep.Kws.Fire(OnebotReceiveMessage, []byte(resp.String()))
	}
	// 完活
}

// 操作性质的函数们
func (p *PlatformAdapterPureOnebot11) SetFriendAddRequest(flag string, approve bool, remark string, reason string, ep *socketio.EventPayload) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		Remark  string `json:"remark"` // 备注名
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	msg, _ := sonic.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_friend_add_request",
		DetailParams{
			Flag:    flag,
			Approve: approve,
			Remark:  remark,
			Reason:  reason,
		},
	})
	// 发送数据
	ep.Kws.Emit(msg, socketio.TextMessage)
}

// UTILS函数们

// 完全重写的Message转换逻辑，采用gjson实现
func (p *PlatformAdapterPureOnebot11) convertStringMessage(operator gjson.Result) *Message {
	msg := new(Message)

	msg.Time = operator.Get("time").Int()
	msg.MessageType = operator.Get("message_type").String()
	msg.Message = operator.Get("message").String()
	// 看上去就是原本的替换策略
	msg.Message = strings.ReplaceAll(msg.Message, "&#91;", "[")
	msg.Message = strings.ReplaceAll(msg.Message, "&#93;", "]")
	msg.Message = strings.ReplaceAll(msg.Message, "&amp;", "&")
	msg.RawID = operator.Get("message_id").String()
	msg.Platform = "QQ"

	if msg.MessageType == "" {
		msg.MessageType = "private"
	}

	// 这两段代码什么情况？尝试取值？
	if operator.Get("data").Exists() && operator.Get("data.group_id").Exists() {
		msg.GroupID = FormatDiceIDQQGroup(operator.Get("data.group_id").String())
	}
	if operator.Get("group_id").Exists() {
		if msg.MessageType == "private" {
			msg.MessageType = "group"
		}
		msg.GroupID = FormatDiceIDQQGroup(operator.Get("group_id").String())
	}
	sender := operator.Get("sender")
	if sender.Exists() {
		msg.Sender.Nickname = sender.Get("nickname").String()
		if sender.Get("card").Exists() {
			msg.Sender.Nickname = sender.Get("card").String()
		}
		msg.Sender.GroupRole = sender.Get("role").String()
		msg.Sender.UserID = FormatDiceIDQQ(sender.Get("user_id").String())
	}
	return msg
}

func (p *PlatformAdapterPureOnebot11) parseOB11ArrayToStringMessage(parseContent gjson.Result) (gjson.Result, error) {
	arrayContent := parseContent.Get("message").Array()
	cqMessage := strings.Builder{}

	for _, i := range arrayContent {
		// 使用String()方法，如果为空，会自动产生空字符串
		typeStr := i.Get("type").String()
		dataObj := i.Get("data")
		switch typeStr {
		case "text":
			cqMessage.WriteString(dataObj.Get("text").String())
		case "image":
			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
			}
		case "face":
			// 兼容四叶草，移除 .(string)。自动获取的信息表示此类型为 float64，这是go解析的问题
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
		case "record":
			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
			}
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
		case "poke":
			cqMessage.WriteString("[CQ:poke]")
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
		}
	}
	// 赋值对应的Message
	tempStr, err := sjson.Set(parseContent.String(), "message", cqMessage.String())
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Parse(tempStr), nil
}

// 编写一堆handle，用来管理六种事件

func (p *PlatformAdapterPureOnebot11) Serve() int {
	// TODO: seal的SocketIO整体需要补充逻辑！
	// 先做服务器模式，客户端模式需要移植部分 gowebsocket 的代码
	err := p.PreloadInstance()
	if err != nil {
		return -1
	}
	// 如果是反向服务器的初始化
	if p.IsReverse {
		var upgrader = websocket.Upgrader{}
		// 启动反向服务器
		go func() {
			e := echo.New()

			e.GET("/echo", func(c echo.Context) error {
				// Upgrade to WebSocket
				conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
				if err != nil {
					log.Error("upgrade:", err)
					return err
				}

				handler := p.Instance.New(func(kws *socketio.WebsocketWrapper) {
					// Broadcast to all connected users about the newcomer
					kws.Broadcast([]byte(fmt.Sprintf("New user connected: and UUID: %s", kws.UUID)), true, socketio.TextMessage)
					// Write welcome message
					kws.Emit([]byte(fmt.Sprintf("Hello user: with UUID: %s", kws.UUID)), socketio.TextMessage)
				}, conn)

				// Since Echo handles the request/response cycle differently,
				// we need to call the handler directly
				handler(c.Response(), c.Request())

				return nil
			})

			log.Fatal(e.Start("0.0.0.0:3002"))
		}()

	}
	// TODO: 缺少代码

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

// 这下上下文真有用了，需要上下文里面带上客户端/服务端的UUID。
// 给倒霉蛋发消息 这里就能拿上UUID给用户发数据
// btw：虽然但是真有反向连接一次连俩的情况？似乎是个伪命题
// TODO: 考虑移除多连接，连接数 > 1时，自动踹掉前面那个连接。
func (p *PlatformAdapterPureOnebot11) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
	//
	//rawID, idType := pa.mustExtractID(userID)
	//
	//if idType != QQUidPerson {
	//	return
	//}
	//
	//for _, i := range ctx.Dice.ExtList {
	//	if i.OnMessageSend != nil {
	//		i.callWithJsCheck(ctx.Dice, func() {
	//			i.OnMessageSend(ctx, &Message{
	//				Message:     text,
	//				MessageType: "private",
	//				Platform:    pa.EndPoint.Platform,
	//				Sender: SenderBase{
	//					Nickname: pa.EndPoint.Nickname,
	//					UserID:   pa.EndPoint.UserID,
	//				},
	//			}, flag)
	//		})
	//	}
	//}
	//
	//type GroupMessageParams struct {
	//	MessageType string `json:"message_type"`
	//	UserID      int64  `json:"user_id"`
	//	Message     string `json:"message"`
	//}
	//
	//text = textAssetsConvert(text)
	//texts := textSplit(text)
	//
	//for index, subText := range texts {
	//	re := regexp.MustCompile(`\[CQ:poke,qq=(\d+)\]`)
	//
	//	if re.MatchString(subText) {
	//		re = regexp.MustCompile(`\d+`)
	//		qq := re.FindStringSubmatch(subText)
	//		pa.FriendPoke(qq[0])
	//		texts = append(texts[:index], texts[index+1:]...)
	//	}
	//}
	//
	//for _, subText := range texts {
	//	a, _ := json.Marshal(oneBotCommand{
	//		Action: "send_msg",
	//		Params: GroupMessageParams{
	//			MessageType: "private",
	//			UserID:      rawID,
	//			Message:     subText,
	//		},
	//	})
	//	doSleepQQ(ctx)
	//	socketSendText(pa.Socket, string(a))
	//}
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
	// 获取群信息
}

func (p *PlatformAdapterPureOnebot11) EditMessage(ctx *MsgContext, msgID, message string) {
	// 改信息？
}

func (p *PlatformAdapterPureOnebot11) RecallMessage(ctx *MsgContext, msgID string) {
	// 重新发信息？
}
