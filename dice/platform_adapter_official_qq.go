package dice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sealdice/botgo/event"

	qqbot "github.com/sealdice/botgo"
	"github.com/sealdice/botgo/dto"
	qqapi "github.com/sealdice/botgo/openapi"
	qqtoken "github.com/sealdice/botgo/token"
	"golang.org/x/oauth2"

	"sealdice-core/message"
)

type PlatformAdapterOfficialQQ struct {
	Session     *IMSession    `json:"-" yaml:"-"`
	EndPoint    *EndPointInfo `json:"-" yaml:"-"`
	DiceServing bool          `yaml:"-"`

	AppID       string `json:"appID"       yaml:"appID"`
	AppSecret   string `json:"appSecret"   yaml:"appSecret"`
	Token       string `json:"token"       yaml:"token"`
	OnlyQQGuild bool   `json:"onlyQQGuild" yaml:"onlyQQGuild"`
	
	// Webhook配置
	UseWebhook    bool   `json:"useWebhook"    yaml:"useWebhook"`       // 是否使用Webhook模式
	WebhookPath   string `json:"webhookPath"   yaml:"webhookPath"`      // Webhook路径，默认 /webhook/qq
	WebhookPort   int    `json:"webhookPort"   yaml:"webhookPort"`      // Webhook端口，默认 8099
	WebhookSecret string `json:"webhookSecret" yaml:"webhookSecret"`    // Webhook密钥（可选）

	Api            qqapi.OpenAPI        `json:"-" yaml:"-"`
	SessionManager qqbot.SessionManager `json:"-" yaml:"-"`
	Ctx            context.Context      `json:"-" yaml:"-"`
	CancelFunc     context.CancelFunc   `json:"-" yaml:"-"`
	tokenSource    oauth2.TokenSource   `json:"-" yaml:"-"`
	
	// Webhook服务器
	webhookServer *http.Server `json:"-" yaml:"-"`
}

// WSGroupMessageData 群聊普通消息数据（非@消息），对应 GROUP_MESSAGE_CREATE 事件
// botgo dto 包中未包含此类型，在此补充定义
type WSGroupMessageData struct {
	ID          string              `json:"id"`
	Content     string              `json:"content"`
	Timestamp   dto.Timestamp       `json:"timestamp"`
	Author      *GroupMessageAuthor  `json:"author"`
	GroupOpenID string              `json:"group_openid"`
}

type GroupMessageAuthor struct {
	ID           string `json:"id"`            // 用户ID（如果有）
	MemberOpenID string `json:"member_openid"` // 成员OpenID
}

// WSGroupMemberAddData 群成员增加事件数据，对应 GROUP_MEMBER_ADD 事件
type WSGroupMemberAddData struct {
	GroupOpenID    string `json:"group_openid"`     // 群OpenID
	MemberOpenID   string `json:"member_openid"`    // 新成员OpenID
	OpMemberOpenID string `json:"op_member_openid"` // 操作者OpenID
	Timestamp      int64  `json:"timestamp"`        // 时间戳（秒）
}

// WSGroupMemberRemoveData 群成员减少事件数据，对应 GROUP_MEMBER_REMOVE 事件
type WSGroupMemberRemoveData struct {
	GroupOpenID    string `json:"group_openid"`     // 群OpenID
	MemberOpenID   string `json:"member_openid"`    // 被移除成员OpenID
	OpMemberOpenID string `json:"op_member_openid"` // 操作者OpenID
	Timestamp      int64  `json:"timestamp"`        // 时间戳（秒）
}

type GroupMessageEventHandler func(event *dto.WSPayload, data *WSGroupMessageData) error
type GroupMemberAddEventHandler func(event *dto.WSPayload, data *WSGroupMemberAddData) error
type GroupMemberRemoveEventHandler func(event *dto.WSPayload, data *WSGroupMemberRemoveData) error

func (pa *PlatformAdapterOfficialQQ) Serve() int {
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	d := pa.Session.Parent

	if pa.Ctx != nil {
		log.Info("official qq session already running, skip Serve")
		return 0
	}

	pa.AppID = strings.TrimSpace(pa.AppID)
	pa.AppSecret = strings.TrimSpace(pa.AppSecret)
	pa.Token = strings.TrimSpace(pa.Token)

	log.Debug("official qq server")
	qqbot.SetLogger(NewDummyLogger())

	// 初始化 OAuth2 token source
	pa.tokenSource = qqtoken.NewQQBotTokenSource(&qqtoken.QQBotCredentials{
		AppID:     pa.AppID,
		AppSecret: pa.AppSecret,
	})

	ctx, cancel := context.WithCancel(context.Background())
	pa.Ctx, pa.CancelFunc = ctx, cancel

	// 启动 token 自动刷新
	if err := qqtoken.StartRefreshAccessToken(ctx, pa.tokenSource); err != nil {
		log.Error("official qq 启动 token 刷新失败: ", err)
		ep.State = 3
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
		pa.Api = nil
		pa.Ctx = nil
		pa.CancelFunc = nil
		return 1
	}

	// 创建 OpenAPI 客户端
	pa.Api = qqbot.NewOpenAPI(pa.AppID, pa.tokenSource).WithTimeout(3 * time.Second)

	// 注册事件处理器
	event.RegisterHandlers(
		pa.makeHandlers()...,
	)

	// 获取机器人信息
	botInfo, err := pa.Api.Me(ctx)
	if err != nil {
		log.Error("official qq 获取机器人信息失败: ", err)
		ep.State = 3
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
		pa.Api = nil
		pa.Ctx = nil
		pa.CancelFunc = nil
		return 1
	}

	ep.UserID = formatDiceIDOfficialQQ(botInfo.ID)
	ep.Nickname = botInfo.Username

	// 区分 Webhook 还是 WebSocket 模式
	if pa.UseWebhook {
		// 启动 webhook 服务器
		if pa.WebhookPath == "" {
			pa.WebhookPath = "/webhook/qq"
		}
		if pa.WebhookPort == 0 {
			pa.WebhookPort = 8099
		}

		mux := http.NewServeMux()
		mux.HandleFunc(pa.WebhookPath, pa.handleWebhookCallback)

		addr := fmt.Sprintf(":%d", pa.WebhookPort)
		pa.webhookServer = &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		go func() {
			log.Infof("official qq webhook: 监听地址 %s%s", addr, pa.WebhookPath)
			if err := pa.webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("official qq webhook服务器启动失败: ", err)
				if pa.Ctx == ctx {
					ep.State = 3
					ep.Enable = false
				}
			}
		}()

		ep.State = 1
		ep.Enable = true
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		log.Info("official qq webhook模式启动成功")
	} else {
		// WebSocket 模式
		pa.SessionManager = qqbot.NewSessionManager()

		ep.State = 2
		log.Debug("official qq connecting")
		ws, err := pa.Api.WS(ctx, nil, "")
		if err != nil || ws == nil {
			log.Error("official qq 获取 ws 接入点失败: ", err)
			log.Error("official qq 提示：请确认在机器人后台配置了 IP 白名单，并检查 AppID/AppSecret 是否正确")
			ep.State = 3
			if pa.CancelFunc != nil {
				pa.CancelFunc()
			}
			pa.Api = nil
			pa.SessionManager = nil
			pa.Ctx = nil
			pa.CancelFunc = nil
			return 1
		}
		// 极端情况下 shards 为 0 会导致 session manager 阻塞在 channel range 上
		if ws.Shards == 0 {
			ws.Shards = 1
		}
		// 频控不满足时，botgo 会直接返回错误；这里提前检查避免在 goroutine 内“静默失败”
		if ws.Shards > ws.SessionStartLimit.Remaining {
			log.Errorf(
				"official qq session limited: shards=%d remaining=%d resetAfter=%d maxConcurrency=%d",
				ws.Shards, ws.SessionStartLimit.Remaining, ws.SessionStartLimit.ResetAfter, ws.SessionStartLimit.MaxConcurrency,
			)
			ep.State = 3
			if pa.CancelFunc != nil {
				pa.CancelFunc()
			}
			pa.Api = nil
			pa.SessionManager = nil
			pa.Ctx = nil
			pa.CancelFunc = nil
			return 1
		}

		var intent dto.Intent
		// 文字子频道at消息
		intent = intent | dto.IntentGuildAtMessage
		// 频道私信
		intent = intent | dto.IntentDirectMessages

		if !pa.OnlyQQGuild {
			// 群聊@消息、单聊、好友关系事件、进入AIO等
			intent = intent | dto.IntentGroupMessages
			intent = intent | dto.IntentEnterAIO
		}

		go func() {
			currentCtx := ctx
			defer func() {
				isCurrent := pa.Ctx == currentCtx
				// 防止崩掉进程
				if r := recover(); r != nil {
					log.Error("official qq 启动失败: ", r)
					if isCurrent {
						ep.State = 3
						ep.Enable = false
					}
				}
				if isCurrent {
					pa.Ctx = nil
					pa.CancelFunc = nil
					pa.SessionManager = nil
				}
			}()
			if startErr := pa.SessionManager.Start(currentCtx, ws, pa.tokenSource, &intent); startErr != nil {
				log.Error("official qq session manager 启动失败: ", startErr)
				if pa.Ctx == currentCtx {
					ep.State = 3
					ep.Enable = false
				}
			}
		}()

		ep.State = 1
		ep.Enable = true
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		log.Info("official qq 连接成功")
	}

	return 0
}

// makeHandlers 构造事件处理器列表
func (pa *PlatformAdapterOfficialQQ) makeHandlers() []interface{} {
	handlers := []interface{}{
		// 频道@消息
		event.ATMessageEventHandler(pa.ChannelAtMessageReceive),
		// 频道私信
		event.DirectMessageEventHandler(pa.GuildDirectMessageReceive),
	}

	if !pa.OnlyQQGuild {
		handlers = append(handlers,
			// 群聊@消息
			event.GroupATMessageEventHandler(pa.GroupAtMessageReceive),
			// 单聊消息
			event.C2CMessageEventHandler(pa.C2CMessageReceiveFromEvent),
		)
	}

	return handlers
}

func (pa *PlatformAdapterOfficialQQ) ChannelAtMessageReceive(event *dto.WSPayload, data *dto.WSATMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到文字频道消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.channelMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) channelMsgToStdMsg(msgQQ *dto.WSATMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQCH"
	msg.GuildID = formatDiceIDOfficialQQChGuild(msgQQ.GuildID)
	channelID := formatDiceIDOfficialQQChannel(msgQQ.GuildID, msgQQ.ChannelID)
	msg.GroupID = channelID
	msg.ChannelID = channelID
	if msgQQ.Author != nil {
		msg.Sender.Nickname = msgQQ.Author.Username
		msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
	}
	return msg
}

func (pa *PlatformAdapterOfficialQQ) GuildDirectMessageReceive(event *dto.WSPayload, data *dto.WSDirectMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到频道私信消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.guildDirectMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) guildDirectMsgToStdMsg(msgQQ *dto.WSDirectMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "private"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQCH"
	// 频道私信需要私信频道的 guild_id 和 channel_id
	channelID := formatDiceIDOfficialQQChannel(msgQQ.GuildID, msgQQ.ChannelID)
	msg.GroupID = channelID
	msg.ChannelID = channelID
	if msgQQ.Author != nil {
		msg.Sender.Nickname = msgQQ.Author.Username
		msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
	}
	return msg
}

func (pa *PlatformAdapterOfficialQQ) GroupAtMessageReceive(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群聊消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.groupMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) groupMsgToStdMsg(msgQQ *dto.WSGroupATMessageData) *Message {
	appID := pa.AppID
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	msg.GroupID = formatDiceIDOfficialQQGroupOpenID(appID, msgQQ.GroupOpenID)
	if msgQQ.Author != nil {
		// FIXME: 我要用户名啊kora
		msg.Sender.Nickname = "用户" + msgQQ.Author.MemberOpenID[len(msgQQ.Author.MemberOpenID)-4:]
		msg.Sender.UserID = formatDiceIDOfficialQQMemberOpenID(appID, msgQQ.GroupOpenID, msgQQ.Author.MemberOpenID)
	}
	return msg
}

// GroupMessageReceive 处理群聊普通消息（非@）
func (pa *PlatformAdapterOfficialQQ) GroupMessageReceive(event *dto.WSPayload, data *WSGroupMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群聊普通消息：%v, %v", event, data)

	msg := pa.groupNormalMsgToStdMsg(data)
	s.Execute(pa.EndPoint, msg, false)
	return nil
}

// groupNormalMsgToStdMsg 将群聊普通消息转换为标准消息
func (pa *PlatformAdapterOfficialQQ) groupNormalMsgToStdMsg(msgQQ *WSGroupMessageData) *Message {
	appID := pa.AppID
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	msg.GroupID = formatDiceIDOfficialQQGroupOpenID(appID, msgQQ.GroupOpenID)
	if msgQQ.Author != nil {
		msg.Sender.Nickname = "用户" + msgQQ.Author.MemberOpenID[len(msgQQ.Author.MemberOpenID)-4:]
		msg.Sender.UserID = formatDiceIDOfficialQQMemberOpenID(appID, msgQQ.GroupOpenID, msgQQ.Author.MemberOpenID)
	}
	return msg
}

// C2CMessageReceiveFromEvent 处理单聊消息（使用 botgo dto 内置类型）
func (pa *PlatformAdapterOfficialQQ) C2CMessageReceiveFromEvent(payload *dto.WSPayload, data *dto.WSC2CMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到单聊消息: %v", data)

	msg := pa.c2cMsgToStdMsg(data)
	s.Execute(pa.EndPoint, msg, false)
	return nil
}

// c2cMsgToStdMsg 将单聊消息转换为标准消息
func (pa *PlatformAdapterOfficialQQ) c2cMsgToStdMsg(msgQQ *dto.WSC2CMessageData) *Message {
	appID := pa.AppID
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "private"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	if msgQQ.Author != nil {
		userOpenID := msgQQ.Author.UserOpenID
		if len(userOpenID) >= 4 {
			msg.Sender.Nickname = "用户" + userOpenID[len(userOpenID)-4:]
		} else {
			msg.Sender.Nickname = "用户"
		}
		msg.Sender.UserID = formatDiceIDOfficialQQUserOpenID(appID, userOpenID)
	}
	return msg
}

// GroupMemberAddReceive 处理群成员增加事件
func (pa *PlatformAdapterOfficialQQ) GroupMemberAddReceive(event *dto.WSPayload, data *WSGroupMemberAddData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群成员增加事件：%v, %v", event, data)

	appID := pa.AppID
	groupID := formatDiceIDOfficialQQGroupOpenID(appID, data.GroupOpenID)
	userID := formatDiceIDOfficialQQMemberOpenID(appID, data.GroupOpenID, data.MemberOpenID)

	// 如果是机器人自己加入群
	if userID == pa.EndPoint.UserID {
		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		ctx.Group = SetBotOnAtGroup(ctx, groupID)
		ctx.Group.DiceIDExistsMap.Store(ctx.EndPoint.UserID, true)
		ctx.Group.EnteredTime = time.Now().Unix()
		ctx.Group.MarkDirty(ctx.Dice)

		log.Infof("official qq: 机器人加入群 %s", groupID)

		// 发送入群致辞
		go func() {
			time.Sleep(2 * time.Second)
			ctx.Player = &GroupPlayerInfo{}
			text := DiceFormatTmpl(ctx, "核心:骰子进群")
			for _, i := range ctx.SplitText(text) {
				pa.SendToGroup(ctx, groupID, strings.TrimSpace(i), "")
			}
		}()
	}

	return nil
}

// GroupMemberRemoveReceive 处理群成员减少事件
func (pa *PlatformAdapterOfficialQQ) GroupMemberRemoveReceive(event *dto.WSPayload, data *WSGroupMemberRemoveData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群成员减少事件：%v, %v", event, data)

	appID := pa.AppID
	groupID := formatDiceIDOfficialQQGroupOpenID(appID, data.GroupOpenID)
	userID := formatDiceIDOfficialQQMemberOpenID(appID, data.GroupOpenID, data.MemberOpenID)

	// 如果是机器人自己被移出群
	if userID == pa.EndPoint.UserID {
		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		groupName := s.Parent.Parent.TryGetGroupName(groupID)

		txt := fmt.Sprintf("official qq: 离开群组: <%s>(%s)", groupName, groupID)
		log.Info(txt)

		group, exists := s.ServiceAtNew.Load(groupID)
		if exists {
			group.DiceIDExistsMap.Delete(pa.EndPoint.UserID)
			group.MarkDirty(s.Parent)
		}

		ctx.Notice(txt)
	}

	return nil
}
func (pa *PlatformAdapterOfficialQQ) DoRelogin() bool {
	if pa.CancelFunc != nil {
		pa.CancelFunc()
	}
	pa.Session.Parent.Logger.Infof("正在启用 official qq 服务")
	pa.EndPoint.State = 0
	pa.EndPoint.Enable = false
	pa.Api = nil
	pa.Ctx = nil
	pa.CancelFunc = nil
	pa.tokenSource = nil
	if pa.webhookServer != nil {
		pa.webhookServer.Close()
		pa.webhookServer = nil
	}
	return pa.Serve() == 0
}

func (pa *PlatformAdapterOfficialQQ) SetEnable(enable bool) {
	d := pa.Session.Parent
	ep := pa.EndPoint
	if enable {
		if pa.Ctx == nil {
			ep.Enable = false
			pa.DiceServing = false
			ep.State = 2
			ServerOfficialQQ(d, ep)
		} else {
			ep.Enable = true
			ep.State = 1
		}
	} else {
		ep.State = 0
		ep.Enable = false
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
		if pa.webhookServer != nil {
			pa.webhookServer.Close()
			pa.webhookServer = nil
		}
		pa.CancelFunc = nil
		pa.Ctx = nil
		pa.tokenSource = nil
	}
	d.LastUpdatedTime = time.Now().Unix()
}

func (pa *PlatformAdapterOfficialQQ) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterOfficialQQ) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterOfficialQQ) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	userID, idType := pa.mustExtractID(uid)

	if idType == OpenQQUserOpenid {
		// 单聊消息
		rowID, ok := VarGetValueStr(ctx, "$tMsgID")
		if !ok {
			pa.Session.Parent.Logger.Error("official qq 发送单聊消息失败：无法获取消息ID")
			return
		}
		pa.sendC2CMsgRaw(ctx, rowID, userID, text)
		return
	}

	if idType != OpenQQCHUser {
		// 说明不是频道信息
		pa.Session.Parent.Logger.Error("official qq 发送私聊消息失败：不支持该功能")
		return
	}
	channelID, guildID, _ := pa.mustExtractTwoID(ctx.Group.ChannelID)
	rowID, ok := VarGetValueStr(ctx, "$tMsgID")
	if !ok || ctx.MessageType == "group" {
		// 需要主动发起私聊
		g, c, err := pa.createQQGuildDirectChannel(ctx, guildID, userID)
		if err != nil {
			pa.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：", err.Error())
			return
		}
		guildID = g
		channelID = c
	}
	pa.sendQQGuildDirectMsgRaw(ctx, rowID, guildID, channelID, text)
}

func (pa *PlatformAdapterOfficialQQ) createQQGuildDirectChannel( /* ctx */ _ *MsgContext, guildID, userID string) (string, string, error) {
	if guildID == "" || userID == "" {
		err := errors.New("创建私信频道的参数不全")
		pa.Session.Parent.Logger.Error("official qq 创建私信频道失败：" + err.Error())
		return "", "", err
	}
	qctx := context.Background()
	toCreate := &dto.DirectMessageToCreate{
		SourceGuildID: guildID,
		RecipientID:   userID,
	}
	info, err := pa.Api.CreateDirectMessage(qctx, toCreate)
	if err != nil {
		pa.Session.Parent.Logger.Error("official qq 创建私信频道失败：" + err.Error())
		return "", "", err
	}
	return info.GuildID, info.ChannelID, nil
}

func (pa *PlatformAdapterOfficialQQ) sendQQGuildDirectMsgRaw( /* ctx */ _ *MsgContext, rowMsgID string, guildID, channelID string, text string) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content  string
		toCreate *dto.MessageToCreate
	)

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			content += e.Content
		case *message.ImageElement:
		}
	}

	dMsg := &dto.DirectMessage{
		GuildID:   guildID,
		ChannelID: channelID,
	}
	toCreate = &dto.MessageToCreate{
		Content: content,
		MsgType: 0,
		MsgID:   rowMsgID,
	}
	if _, err := pa.Api.PostDirectMessage(qctx, dMsg, toCreate); err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：" + err.Error())
	}
}

// sendC2CMsgRaw 发送单聊消息（使用msg_id被动回复）
func (pa *PlatformAdapterOfficialQQ) sendC2CMsgRaw( /* ctx */ _ *MsgContext, rowMsgID, userOpenID string, text string) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var content string

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			// QQ官方API中不能发送链接，所以全部进行转写绕过
			content += textLinkStrip(e.Content)
		case *message.ImageElement:
			// 单聊暂不支持图片，跳过
		}
	}

	toCreate := &dto.MessageToCreate{
		Content: content,
		MsgType: 0,
		MsgID:   rowMsgID,
	}

	if _, err := pa.Api.PostC2CMessage(qctx, userOpenID, toCreate); err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送单聊消息失败：" + err.Error())
	}
}

func (pa *PlatformAdapterOfficialQQ) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	rowID, ok := VarGetValueStr(ctx, "$tMsgID")
	if !ok {
		// TODO：允许主动消息发送，并校验频率
		pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：无法直接发送消息")
		return
	}
	groupId, idType := pa.mustExtractID(uid)
	switch idType {
	case OpenQQGroupOpenid:
		pa.sendQQGroupMsgRaw(ctx, rowID, groupId, text)
	case OpenQQCHChannel:
		pa.sendQQChannelMsgRaw(ctx, rowID, groupId, text)
	default:
		pa.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", uid, idType)
		return
	}
}

func (pa *PlatformAdapterOfficialQQ) sendQQGroupMsgRaw( /* ctx */ _ *MsgContext, rowMsgID, groupID string, text string) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content  string
		toCreate *dto.MessageToCreate
	)

	toCreate = &dto.MessageToCreate{
		MsgID: rowMsgID,
	}

	for _, element := range elems {
		switch elem := element.(type) {
		case *message.TextElement:
			// QQ官方API中不能发送链接，所以全部进行转写绕过
			content += textLinkStrip(elem.Content)
		case *message.AtElement:
			pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持 AT 他人，跳过该部分")
		case *message.ImageElement:
			url := elem.File.URL
			// 目前不支持本地发送，检查一下url
			if url == "" ||
				strings.Contains(url, "localhost") ||
				strings.Contains(url, "127.0.0.1") {
				pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持发送本地图片，跳过该部分")
			}
			fMsg := &dto.MessageMediaToCreate{
				FileType:   1,
				URL:        url,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备图片信息失败：" + err.Error())
				continue
			}

			toCreate.MsgType = 7
			toCreate.Media = &dto.MediaInfo{
				FileInfo: []byte(media.FileInfo),
			}
		case *message.RecordElement:
			url := elem.File.URL
			// 目前不支持本地发送，检查一下url
			if url == "" ||
				strings.Contains(url, "localhost") ||
				strings.Contains(url, "127.0.0.1") {
				pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持发送本地语音，跳过该部分")
			}
			fMsg := &dto.MessageMediaToCreate{
				FileType:   3,
				URL:        url,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备语音信息失败：" + err.Error())
				continue
			}

			toCreate.MsgType = 7
			toCreate.Media = &dto.MediaInfo{
				FileInfo: []byte(media.FileInfo),
			}
		}
	}

	toCreate.Content = content

	if _, err := pa.Api.PostGroupMessage(qctx, groupID, toCreate); err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：" + err.Error())
	}
}

func (pa *PlatformAdapterOfficialQQ) sendQQChannelMsgRaw( /* ctx */ _ *MsgContext, rowMsgID, channelID string, text string) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content  string
		toCreate *dto.MessageToCreate
	)

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			// QQ官方API中不能发送链接，所以全部进行转写绕过
			content += textLinkStrip(e.Content)
		case *message.AtElement:
			if e.Target == "all" {
				content += "@everyone"
			} else {
				content += fmt.Sprintf("<@%s>", e.Target)
			}
		case *message.ImageElement:
		}
	}

	toCreate = &dto.MessageToCreate{
		Content: content,
		MsgType: 0,
		MsgID:   rowMsgID,
	}
	if _, err := pa.Api.PostMessage(qctx, channelID, toCreate); err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送频道消息失败：" + err.Error())
	}
}

func (pa *PlatformAdapterOfficialQQ) GetGroupInfoAsync(groupID string) {
	// 警告太频繁了，拿掉
	// pa.Session.Parent.Logger.Infof("official qq 更新群信息失败：不支持该功能")
}

func formatDiceIDOfficialQQCh(userID string) string {
	return fmt.Sprintf("OpenQQCH:%s", userID)
}

func formatDiceIDOfficialQQChGuild(guildID string) string {
	return fmt.Sprintf("OpenQQCH-Guild:%s", guildID)
}

func formatDiceIDOfficialQQChannel(guildID, channelID string) string {
	return fmt.Sprintf("OpenQQCH-Channel:%s-%s", guildID, channelID)
}

func formatDiceIDOfficialQQ(userUnionID string) string {
	return fmt.Sprintf("OpenQQ:%s", userUnionID)
}

func formatDiceIDOfficialQQGroupOpenID(botID, groupOpenID string) string {
	// 在没有qq_unionid时的临时方案
	return fmt.Sprintf("OpenQQ-Group-T:%s-%s", botID, groupOpenID)
}

func formatDiceIDOfficialQQMemberOpenID(botID, groupOpenID, memberOpenID string) string {
	// 在没有qq_unionid时的临时方案
	return fmt.Sprintf("OpenQQ-Member-T:%s-%s-%s", botID, groupOpenID, memberOpenID)
}

func formatDiceIDOfficialQQUserOpenID(botID, userOpenID string) string {
	// 单聊用户OpenID格式
	return fmt.Sprintf("OpenQQ-User-T:%s-%s", botID, userOpenID)
}

type OpenQQIDType = int

const (
	OpenQQUnknown OpenQQIDType = iota

	OpenQQUser
	OpenQQGroupOpenid
	OpenQQGroupMemberOpenid
	OpenQQUserOpenid

	OpenQQCHUser
	OpenQQCHGuild
	OpenQQCHChannel
)

func (pa *PlatformAdapterOfficialQQ) mustExtractID(text string) (string, OpenQQIDType) {
	id, _, idType := pa.mustExtractTwoID(text)
	return id, idType
}

func (pa *PlatformAdapterOfficialQQ) mustExtractTwoID(text string) (string, string, OpenQQIDType) {
	if strings.HasPrefix(text, "OpenQQ:") {
		return text[len("OpenQQ:"):], "", OpenQQUser
	}
	if strings.HasPrefix(text, "OpenQQ-Group-T:") {
		temp := text[len("OpenQQ-Group-T:"):]
		lst := strings.Split(temp, "-")
		return lst[1], "", OpenQQGroupOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-Member-T:") {
		temp := text[len("OpenQQ-Member-T:"):]
		lst := strings.Split(temp, "-")
		return lst[2], lst[1], OpenQQGroupMemberOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-User-T:") {
		temp := text[len("OpenQQ-User-T:"):]
		lst := strings.Split(temp, "-")
		return lst[1], "", OpenQQUserOpenid
	}
	if strings.HasPrefix(text, "OpenQQCH:") {
		return text[len("OpenQQCH:"):], "", OpenQQCHUser
	}
	if strings.HasPrefix(text, "OpenQQCH-Guild:") {
		return text[len("OpenQQCH-Guild:"):], "", OpenQQCHGuild
	}
	if strings.HasPrefix(text, "OpenQQCH-Channel:") {
		temp := text[len("OpenQQCH-Channel:"):]
		lst := strings.Split(temp, "-")
		return lst[1], lst[0], OpenQQCHChannel
	}
	return "", "", OpenQQUnknown
}

func (pa *PlatformAdapterOfficialQQ) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterOfficialQQ) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterOfficialQQ) QuitGroup(_ *MsgContext, _ string) {
	pa.Session.Parent.Logger.Error("official qq 退出群组失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) SetGroupCardName(_ *MsgContext, _ string) {
	pa.Session.Parent.Logger.Error("official qq 修改名片失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) MemberBan(_ string, _ string, _ int64) {
	pa.Session.Parent.Logger.Error("official qq 禁言用户失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) MemberKick(_ string, _ string) {
	pa.Session.Parent.Logger.Error("official qq 踢出用户失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterOfficialQQ) RecallMessage(_ *MsgContext, _ string) {}

// ServeWebhook 启动Webhook模式（已整合到 Serve 中，保留作为兼容接口）
func (pa *PlatformAdapterOfficialQQ) ServeWebhook() int {
	return pa.Serve()
}

// handleWebhookCallback 处理Webhook回调
func (pa *PlatformAdapterOfficialQQ) handleWebhookCallback(w http.ResponseWriter, r *http.Request) {
	s := pa.Session
	log := s.Parent.Logger

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("official qq webhook: 读取请求失败: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Debugf("official qq webhook: 收到请求 %s", string(body))

	// 解析事件
	var payload dto.WSPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error("official qq webhook: 解析事件失败: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 保存原始消息数据供 ParseAndHandle 使用
	payload.RawMessage = body

	// 响应确认
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"op": 12, // HTTP回调确认
	}
	json.NewEncoder(w).Encode(response)

	// 异步处理事件
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Errorf("official qq webhook: 处理事件异常: %v", rec)
			}
		}()
		if err := event.ParseAndHandle(&payload); err != nil {
			log.Errorf("official qq webhook: 事件处理失败: %v", err)
		}
	}()
}

// handleWebhookEvent 处理Webhook事件（保留用于手动分发场景）
func (pa *PlatformAdapterOfficialQQ) handleWebhookEvent(payload *dto.WSPayload) {
	s := pa.Session
	log := s.Parent.Logger

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("official qq webhook: 处理事件异常: %v", r)
		}
	}()

	// 优先使用 event.ParseAndHandle 进行自动分发
	if err := event.ParseAndHandle(payload); err != nil {
		log.Errorf("official qq webhook: 事件处理失败: %v", err)
	}
}

// verifyWebhookSignature 验证Webhook签名
// 注意：QQ官方使用 Ed25519 签名，这里暂时用简化实现
// 后续可根据 botgo fork 中的 webhook 包替换为完整实现
func (pa *PlatformAdapterOfficialQQ) verifyWebhookSignature(_ []byte, _, _ string) bool {
	// TODO: 实现 Ed25519 签名验证
	return true
}
