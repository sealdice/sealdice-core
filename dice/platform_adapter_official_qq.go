package dice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sealdice/botgo/event"
	"github.com/sealdice/botgo/interaction/signature"

	qqbot "github.com/sealdice/botgo"
	"github.com/sealdice/botgo/dto"
	"github.com/sealdice/botgo/dto/keyboard"
	qqapi "github.com/sealdice/botgo/openapi"
	qqtoken "github.com/sealdice/botgo/token"
	"golang.org/x/oauth2"

	"sealdice-core/message"
	"sealdice-core/utils"
)

type PaginationItem struct {
	Pages     []string
	CreatedAt time.Time
}

type PlatformAdapterOfficialQQ struct {
	Session		*IMSession			`json:"-" yaml:"-"`
	EndPoint	*EndPointInfo			`json:"-" yaml:"-"`
	DiceServing	bool				`yaml:"-"`

	AppID		string				`json:"appID"       yaml:"appID"`
	AppSecret	string				`json:"appSecret"   yaml:"appSecret"`
	Token		string				`json:"token"       yaml:"token"`
	OnlyQQGuild	bool				`json:"onlyQQGuild" yaml:"onlyQQGuild"`

	// Webhook配置
	UseWebhook	bool				`json:"useWebhook"    yaml:"useWebhook"`	 // 是否使用Webhook模式
	WebhookPath	string				`json:"webhookPath"   yaml:"webhookPath"`	 // Webhook路径，默认 /webhook
	WebhookPort	int				`json:"webhookPort"   yaml:"webhookPort"`	 // Webhook端口，默认 8099

	Api		qqapi.OpenAPI			`json:"-" yaml:"-"`
	SessionManager	qqbot.SessionManager		`json:"-" yaml:"-"`
	Ctx		context.Context			`json:"-" yaml:"-"`
	CancelFunc	context.CancelFunc		`json:"-" yaml:"-"`
	tokenSource	oauth2.TokenSource		`json:"-" yaml:"-"`

	// Webhook服务器
	webhookServer	*http.Server			`json:"-" yaml:"-"`

	paginationCache	map[string]*PaginationItem	`json:"-" yaml:"-"`
	paginationMu	sync.Mutex			`json:"-" yaml:"-"`
}




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
	event.RegisterHandlersByAppID(
		pa.AppID,
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
			pa.WebhookPath = "/webhook"
		}
		if pa.WebhookPort == 0 {
			pa.WebhookPort = 8099
		}

		mux := http.NewServeMux()
		mux.HandleFunc(pa.WebhookPath, pa.handleWebhookCallback)

		addr := fmt.Sprintf(":%d", pa.WebhookPort)
		pa.webhookServer = &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 3 * time.Second,
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
		intent |= dto.IntentGuildAtMessage
		// 频道私信
		intent |= dto.IntentDirectMessages
		// 互动事件
		intent |= dto.IntentInteraction

		if !pa.OnlyQQGuild {
			// 群聊@消息、单聊、好友关系事件
			intent |= dto.IntentGroupMessages
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
		// 互动事件 (按钮点击)
		event.InteractionEventHandler(pa.InteractionReceive),
	}

	if !pa.OnlyQQGuild {
		handlers = append(handlers,
			// 群聊@消息
			event.GroupATMessageEventHandler(pa.GroupAtMessageReceive),
			// 群聊普通消息 (非@)
			event.GroupMessageEventHandler(pa.GroupMessageReceive),
			// 单聊消息
			event.C2CMessageEventHandler(pa.C2CMessageReceiveFromEvent),
			// 好友关系事件
			event.C2CFriendEventHandler(pa.C2CFriendReceive),
			// 机器人加入群聊
			event.GroupAddRobotEventHandler(pa.GroupAddRobotReceive),
			// 机器人退出群聊
			event.GroupDelRobotEventHandler(pa.GroupDelRobotReceive),
			// 群成员加入
			event.GroupMemberAddEventHandler(pa.GroupMemberAddReceive),
			// 群成员退出
			event.GroupMemberRemoveEventHandler(pa.GroupMemberRemoveReceive),
		)
	}

	return handlers
}

func (pa *PlatformAdapterOfficialQQ) InteractionReceive(eventRaw *dto.WSPayload, data *dto.WSInteractionData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到互动事件: %v, %v", eventRaw, data)

	qctx := context.Background()
	// 首先响应这个 interaction，让客户端停止 loading
	if err := pa.Api.PutInteraction(qctx, data.ID, `{"code":0}`); err != nil {
		log.Errorf("official qq 响应互动事件失败: %v", err)
	}

	if data.Data == nil || data.Data.Type != dto.InteractionDataTypeInlineKeyboardClick {
		return nil
	}

	var resolved dto.Resolved
	if err := json.Unmarshal(data.Data.Resolved, &resolved); err != nil {
		log.Errorf("official qq 解析互动事件 Resolved 失败: %v", err)
		return nil
	}

	buttonData := resolved.ButtonData
	if !strings.HasPrefix(buttonData, "pg:") {
		return nil
	}

	// 格式为 pg:<cacheID>:<pageIndex>
	parts := strings.Split(buttonData, ":")
	if len(parts) != 3 {
		return nil
	}

	cacheID := parts[1]
	pageIndexVal := parts[2]
	var pageIndex int
	if _, err := fmt.Sscanf(pageIndexVal, "%d", &pageIndex); err != nil {
		return err
	}

	pa.paginationMu.Lock()
	item, ok := pa.paginationCache[cacheID]
	pa.paginationMu.Unlock()

	if !ok {
		log.Warnf("official qq 翻页失败：未找到缓存的翻页消息ID %s", cacheID)
		return nil
	}

	if pageIndex < 0 || pageIndex >= len(item.Pages) {
		log.Warnf("official qq 翻页失败：页码 %d 越界 (总数 %d)", pageIndex, len(item.Pages))
		return nil
	}

	text := item.Pages[pageIndex]

	toCreate := &dto.MessageToCreate{
		MsgSeq:  rand.Uint32()%10000000 + 1,
	}

	if eventRaw != nil && eventRaw.EventID != "" {
		toCreate.EventID = eventRaw.EventID
	} else {
		toCreate.EventID = data.ID
	}

	keyboardObj := pa.buildPaginationKeyboard(cacheID, pageIndex, len(item.Pages))

	if pa.Session.Parent.Config.OfficialQQUseMarkdown {
		toCreate.MsgType = 2
		toCreate.Markdown = &dto.Markdown{
			Content: text,
		}
		if keyboardObj != nil {
			toCreate.Keyboard = keyboardObj
		}
	} else {
		toCreate.MsgType = 0
		toCreate.Content = text
	}

	ctx := &MsgContext{
		EndPoint: pa.EndPoint,
		Session:  s,
		Dice:     s.Parent,
	}

	// 根据 chat_type 发送
	switch data.ChatType {
	case 0: // 频道
		msg, err := pa.Api.PostMessage(qctx, data.ChannelID, toCreate)
		if err != nil {
			log.Errorf("official qq 翻页发送频道消息失败: %v", err)
		} else if msg != nil {
			ctx.MessageType = "group"
			pa.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "group",
				Message:     text,
				GroupID:     data.ChannelID,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, "")
		}
	case 1: // 群
		msg, err := pa.Api.PostGroupMessage(qctx, data.GroupOpenID, toCreate)
		if err != nil {
			log.Errorf("official qq 翻页发送群聊消息失败: %v", err)
		} else if msg != nil {
			ctx.MessageType = "group"
			appID := pa.AppID
			groupID := formatDiceIDOfficialQQGroupOpenID(appID, data.GroupOpenID)
			pa.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "group",
				Message:     text,
				GroupID:     groupID,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, "")
		}
	case 2: // C2C
		msg, err := pa.Api.PostC2CMessage(qctx, data.UserOpenID, toCreate)
		if err != nil {
			log.Errorf("official qq 翻页发送私聊消息失败: %v", err)
		} else if msg != nil {
			ctx.MessageType = "private"
			pa.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "private",
				Message:     text,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, "")
		}
	default:
		switch data.Scene {
		case "group":
			msg, err := pa.Api.PostGroupMessage(qctx, data.GroupOpenID, toCreate)
			if err != nil {
				log.Errorf("official qq 翻页发送群聊消息失败: %v", err)
			} else if msg != nil {
				ctx.MessageType = "group"
				appID := pa.AppID
				groupID := formatDiceIDOfficialQQGroupOpenID(appID, data.GroupOpenID)
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     text,
					GroupID:     groupID,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, "")
			}
		case "c2c":
			msg, err := pa.Api.PostC2CMessage(qctx, data.UserOpenID, toCreate)
			if err != nil {
				log.Errorf("official qq 翻页发送私聊消息失败: %v", err)
			} else if msg != nil {
				ctx.MessageType = "private"
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "private",
					Message:     text,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, "")
			}
		default:
			if data.ChannelID != "" {
				msg, err := pa.Api.PostMessage(qctx, data.ChannelID, toCreate)
				if err != nil {
					log.Errorf("official qq 翻页发送频道消息失败: %v", err)
				} else if msg != nil {
					ctx.MessageType = "group"
					pa.Session.OnMessageSend(ctx, &Message{
						Platform:    "QQ",
						MessageType: "group",
						Message:     text,
						GroupID:     data.ChannelID,
						Sender: SenderBase{
							UserID:   pa.EndPoint.UserID,
							Nickname: pa.EndPoint.Nickname,
						},
						RawID: msg.ID,
					}, "")
				}
			}
		}
	}

	return nil
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
func (pa *PlatformAdapterOfficialQQ) GroupMessageReceive(event *dto.WSPayload, data *dto.WSGroupMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群聊普通消息：%v, %v", event, data)

	msg := pa.groupNormalMsgToStdMsg(data)
	s.Execute(pa.EndPoint, msg, false)
	return nil
}

// groupNormalMsgToStdMsg 将群聊普通消息转换为标准消息
func (pa *PlatformAdapterOfficialQQ) groupNormalMsgToStdMsg(msgQQ *dto.WSGroupMessageData) *Message {
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
func (pa *PlatformAdapterOfficialQQ) GroupMemberAddReceive(event *dto.WSPayload, data *dto.WSGroupMemberAddData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群成员增加事件：%v, %v", event, data)

	appID := pa.AppID
	groupID := formatDiceIDOfficialQQGroupOpenID(appID, data.GroupOpenID)
	userID := formatDiceIDOfficialQQMemberOpenID(appID, data.GroupOpenID, data.MemberOpenID)

	// 如果是机器人自己加入群
	if userID == pa.EndPoint.UserID || data.MemberOpenID == "" || data.MemberOpenID == "BOT" {
		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		ctx.Group = SetBotOnAtGroup(ctx, groupID)
		ctx.Group.DiceIDExistsMap.Store(ctx.EndPoint.UserID, true)
		ctx.Group.EnteredTime = time.Now().Unix()
		ctx.Group.MarkDirty(ctx.Dice)

		if event != nil && event.EventID != "" {
			VarSetValueStr(ctx, "$tEventID", event.EventID)
		}

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
	} else {
		// 普通成员进群
		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		if event != nil && event.EventID != "" {
			VarSetValueStr(ctx, "$tEventID", event.EventID)
		}

		msg := &Message{
			Time:        data.Timestamp,
			MessageType: "group",
			Platform:    "QQ",
			GroupID:     groupID,
			Sender: SenderBase{
				UserID:   userID,
				Nickname: "用户",
			},
		}
		if len(data.MemberOpenID) >= 4 {
			msg.Sender.Nickname = "用户" + data.MemberOpenID[len(data.MemberOpenID)-4:]
		}

		pa.Session.OnGroupMemberJoined(ctx, msg)
	}

	return nil
}

// GroupMemberRemoveReceive 处理群成员减少事件
func (pa *PlatformAdapterOfficialQQ) GroupMemberRemoveReceive(event *dto.WSPayload, data *dto.WSGroupMemberRemoveData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群成员减少事件：%v, %v", event, data)

	appID := pa.AppID
	groupID := formatDiceIDOfficialQQGroupOpenID(appID, data.GroupOpenID)
	userID := formatDiceIDOfficialQQMemberOpenID(appID, data.GroupOpenID, data.MemberOpenID)

	// 如果是机器人自己被移出群
	if userID == pa.EndPoint.UserID || data.MemberOpenID == "" || data.MemberOpenID == "BOT" {
		groupName := s.Parent.Parent.TryGetGroupName(groupID)

		txt := fmt.Sprintf("official qq: 离开群组: <%s>(%s)", groupName, groupID)
		log.Info(txt)

		group, exists := s.ServiceAtNew.Load(groupID)
		if exists {
			group.DiceIDExistsMap.Delete(pa.EndPoint.UserID)
			group.MarkDirty(s.Parent)
		}
	}

	return nil
}

// GroupAddRobotReceive 处理机器人加入群聊事件
func (pa *PlatformAdapterOfficialQQ) GroupAddRobotReceive(event *dto.WSPayload, data *dto.WSGroupRobotEventData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到机器人加入群聊事件：%v, %v", event, data)

	// 转化为 WSGroupMemberAddData
	memberData := &dto.WSGroupMemberAddData{
		GroupOpenID:    data.GroupOpenID,
		MemberOpenID:   "BOT",
		OpMemberOpenID: data.OpMemberOpenID,
		Timestamp:      data.Timestamp,
	}
	return pa.GroupMemberAddReceive(event, memberData)
}

// GroupDelRobotReceive 处理机器人退出群聊事件
func (pa *PlatformAdapterOfficialQQ) GroupDelRobotReceive(event *dto.WSPayload, data *dto.WSGroupRobotEventData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到机器人退出群聊事件：%v, %v", event, data)

	// 转化为 WSGroupMemberRemoveData
	memberData := &dto.WSGroupMemberRemoveData{
		GroupOpenID:    data.GroupOpenID,
		MemberOpenID:   "BOT",
		OpMemberOpenID: data.OpMemberOpenID,
		Timestamp:      data.Timestamp,
	}
	return pa.GroupMemberRemoveReceive(event, memberData)
}

// C2CFriendReceive 处理好友关系变动事件
func (pa *PlatformAdapterOfficialQQ) C2CFriendReceive(event *dto.WSPayload, data *dto.WSC2CFriendData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到好友事件: %s, %v, %v", event.Type, event, data)

	switch event.Type {
	case dto.EventC2CFriendAdd:
		appID := pa.AppID
		userID := formatDiceIDOfficialQQUserOpenID(appID, data.OpenID)

		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		if event != nil && event.EventID != "" {
			VarSetValueStr(ctx, "$tEventID", event.EventID)
		}

		msg := &Message{
			Time:        int64(data.Timestamp),
			MessageType: "private",
			Platform:    "QQ",
			Message:     "",
			Sender: SenderBase{
				UserID:   userID,
				Nickname: "用户",
			},
		}
		if len(data.OpenID) >= 4 {
			msg.Sender.Nickname = "用户" + data.OpenID[len(data.OpenID)-4:]
		}

		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		welcomeStr := DiceFormatTmpl(ctx, "核心:骰子成为好友")
		log.Infof("official qq: 与 %s 成为好友，发送好友致辞: %s", userID, welcomeStr)

		go func() {
			time.Sleep(2 * time.Second)
			for _, i := range ctx.SplitText(welcomeStr) {
				pa.SendToPerson(ctx, userID, strings.TrimSpace(i), "")
			}
			if groupInfo, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID); ok {
				groupInfo.TriggerExtHook(ctx.Dice, func(ext *ExtInfo) func() {
					if ext.OnBecomeFriend == nil {
						return nil
					}
					return func() { ext.OnBecomeFriend(ctx, msg) }
				})
			}
		}()
	case dto.EventC2CFriendDel:
		appID := pa.AppID
		userID := formatDiceIDOfficialQQUserOpenID(appID, data.OpenID)
		log.Infof("official qq: 与 %s 解除好友关系", userID)
	}

	return nil
}
func (pa *PlatformAdapterOfficialQQ) initPaginationCache() {
	pa.paginationMu.Lock()
	defer pa.paginationMu.Unlock()
	if pa.paginationCache == nil {
		pa.paginationCache = make(map[string]*PaginationItem)
	}
}

func (pa *PlatformAdapterOfficialQQ) addToPaginationCache(id string, pages []string) {
	pa.initPaginationCache()
	pa.paginationMu.Lock()
	defer pa.paginationMu.Unlock()

	now := time.Now()
	if len(pa.paginationCache) > 1000 {
		for k, v := range pa.paginationCache {
			if now.Sub(v.CreatedAt) > 1*time.Hour || len(pa.paginationCache) > 1000 {
				delete(pa.paginationCache, k)
			}
		}
	}

	pa.paginationCache[id] = &PaginationItem{
		Pages:     pages,
		CreatedAt: now,
	}
}

func generateCacheID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (pa *PlatformAdapterOfficialQQ) buildPaginationKeyboard(cacheID string, pageIndex int, totalPages int) *keyboard.MessageKeyboard {
	if totalPages <= 1 {
		return nil
	}

	var buttons []*keyboard.Button

	// 上一页
	if pageIndex > 0 {
		buttons = append(buttons, &keyboard.Button{
			ID: fmt.Sprintf("prev_%s_%d", cacheID, pageIndex-1),
			RenderData: &keyboard.RenderData{
				Label:        fmt.Sprintf("上一页 (%d/%d)", pageIndex+1, totalPages),
				VisitedLabel: "跳转中",
				Style:        0, // 灰色线框
			},
			Action: &keyboard.Action{
				Type: keyboard.ActionTypeCallback, // Callback
				Data: fmt.Sprintf("pg:%s:%d", cacheID, pageIndex-1),
				Permission: &keyboard.Permission{
					Type: keyboard.PermissionTypAll, // 所有人可操作
				},
			},
		})
	}

	// 下一页
	if pageIndex < totalPages - 1 {
		buttons = append(buttons, &keyboard.Button{
			ID: fmt.Sprintf("next_%s_%d", cacheID, pageIndex+1),
			RenderData: &keyboard.RenderData{
				Label:        fmt.Sprintf("下一页 (%d/%d)", pageIndex+1, totalPages),
				VisitedLabel: "跳转中",
				Style:        1, // 蓝色线框
			},
			Action: &keyboard.Action{
				Type: keyboard.ActionTypeCallback, // Callback
				Data: fmt.Sprintf("pg:%s:%d", cacheID, pageIndex+1),
				Permission: &keyboard.Permission{
					Type: keyboard.PermissionTypAll, // 所有人可操作
				},
			},
		})
	}

	if len(buttons) == 0 {
		return nil
	}

	return &keyboard.MessageKeyboard{
		Content: &keyboard.CustomKeyboard{
			Rows: []*keyboard.Row{
				{
					Buttons: buttons,
				},
			},
		},
	}
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

	maxLen := 900
	totalLen := len(text)
	if totalLen > 900*5 {
		maxLen = totalLen/5 + 50
		if maxLen > 2800 {
			maxLen = 2800
		}
	}
	textList := utils.SplitLongText(text, maxLen, utils.DefaultSplitPaginationHint)
	if !pa.Session.Parent.Config.OfficialQQUseMarkdown {
		if len(textList) > 5 {
			textList = textList[:5]
		}
	}

	if pa.Session.Parent.Config.OfficialQQUseMarkdown && len(textList) > 1 {
		cacheID := generateCacheID()
		pa.addToPaginationCache(cacheID, textList)

		keyboardObj := pa.buildPaginationKeyboard(cacheID, 0, len(textList))

		if idType == OpenQQUserOpenid {
			rowID, ok := VarGetValueStr(ctx, "$tMsgID")
			if !ok {
				rowID, ok = VarGetValueStr(ctx, "$tEventID")
			}
			if !ok {
				pa.Session.Parent.Logger.Error("official qq 发送单聊消息失败：无法获取消息ID")
				return
			}
			msg, err := pa.sendC2CMsgRaw(ctx, rowID, userID, textList[0], keyboardObj)
			if err == nil && msg != nil {
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "private",
					Message:     textList[0],
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
			return
		}

		if idType != OpenQQCHUser {
			pa.Session.Parent.Logger.Error("official qq 发送私聊消息失败：不支持该功能")
			return
		}
		channelID, guildID, _ := pa.mustExtractTwoID(ctx.Group.ChannelID)
		rowID, ok := VarGetValueStr(ctx, "$tMsgID")
		if !ok {
			rowID, ok = VarGetValueStr(ctx, "$tEventID")
		}
		if !ok || ctx.MessageType == "group" {
			g, c, err := pa.createQQGuildDirectChannel(ctx, guildID, userID)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：", err.Error())
				return
			}
			guildID = g
			channelID = c
		}
		msg, err := pa.sendQQGuildDirectMsgRaw(ctx, rowID, guildID, channelID, textList[0], keyboardObj)
		if err == nil && msg != nil {
			pa.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "private",
				Message:     textList[0],
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, flag)
		}
		return
	}

	for _, t := range textList {
		if idType == OpenQQUserOpenid {
			// 单聊消息
			rowID, ok := VarGetValueStr(ctx, "$tMsgID")
			if !ok {
				rowID, ok = VarGetValueStr(ctx, "$tEventID")
			}
			if !ok {
				pa.Session.Parent.Logger.Error("official qq 发送单聊消息失败：无法获取消息ID")
				return
			}
			msg, err := pa.sendC2CMsgRaw(ctx, rowID, userID, t, nil)
			if err == nil && msg != nil {
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "private",
					Message:     t,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
			continue
		}

		if idType != OpenQQCHUser {
			// 说明不是频道信息
			pa.Session.Parent.Logger.Error("official qq 发送私聊消息失败：不支持该功能")
			return
		}
		channelID, guildID, _ := pa.mustExtractTwoID(ctx.Group.ChannelID)
		rowID, ok := VarGetValueStr(ctx, "$tMsgID")
		if !ok {
			rowID, ok = VarGetValueStr(ctx, "$tEventID")
		}
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
		msg, err := pa.sendQQGuildDirectMsgRaw(ctx, rowID, guildID, channelID, t, nil)
		if err == nil && msg != nil {
			pa.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "private",
				Message:     t,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, flag)
		}
	}
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

func (pa *PlatformAdapterOfficialQQ) initMessageToCreate(ctx *MsgContext, rowMsgID string) *dto.MessageToCreate {
	toCreate := &dto.MessageToCreate{
		MsgSeq: rand.Uint32()%10000000 + 1,
	}
	if ctx != nil {
		if eventID, ok := VarGetValueStr(ctx, "$tEventID"); ok && eventID != "" {
			toCreate.EventID = eventID
		} else {
			toCreate.MsgID = rowMsgID
		}
	} else {
		toCreate.MsgID = rowMsgID
	}
	return toCreate
}

func (pa *PlatformAdapterOfficialQQ) finalizeMessageToCreate(toCreate *dto.MessageToCreate, content string, keyboardObj *keyboard.MessageKeyboard, isFinal bool) {
	if toCreate.Media == nil && content == "" && toCreate.MessageReference == nil {
		return
	}

	useMarkdown := pa.Session.Parent.Config.OfficialQQUseMarkdown
	if useMarkdown && toCreate.MsgType != 7 {
		toCreate.MsgType = 2
		toCreate.Markdown = &dto.Markdown{
			Content: content,
		}
		if isFinal && keyboardObj != nil {
			toCreate.Keyboard = keyboardObj
		}
	} else {
		toCreate.Content = content
		if toCreate.MsgType != 7 {
			toCreate.MsgType = 0
		}
		if isFinal && keyboardObj != nil {
			toCreate.Keyboard = keyboardObj
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) prepareMediaMessage(file *message.FileElement) (string, []byte, error) {
	url := file.URL
	if pa.Session.Parent.Config.OfficialQQFileSendBase64 || isLocalOrNonPublic(url) {
		data, err := getElementBytes(file)
		if err != nil {
			return "", nil, err
		}
		return url, data, nil
	}
	return url, nil, nil
}

func (pa *PlatformAdapterOfficialQQ) sendQQGuildDirectMsgRaw( /* ctx */ _ *MsgContext, rowMsgID string, guildID, channelID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content  string
		msgRef   *dto.MessageReference
	)

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			content += e.Content
		case *message.ImageElement:
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             e.ReplySeq,
				IgnoreGetMessageError: true,
			}
		}
	}

	dMsg := &dto.DirectMessage{
		GuildID:   guildID,
		ChannelID: channelID,
	}
	toCreate := pa.initMessageToCreate(nil, rowMsgID)
	toCreate.MessageReference = msgRef
	pa.finalizeMessageToCreate(toCreate, content, keyboardObj, true)

	res, err := pa.Api.PostDirectMessage(qctx, dMsg, toCreate)
	if err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：", err.Error())
	}
	return res, err
}

// sendC2CMsgRaw 发送单聊消息（使用msg_id被动回复）
func (pa *PlatformAdapterOfficialQQ) sendC2CMsgRaw(ctx *MsgContext, rowMsgID, userOpenID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content string
		msgRef   *dto.MessageReference
	)

	toCreate := pa.initMessageToCreate(ctx, rowMsgID)

	var lastRes *dto.Message
	var lastErr error

	sendCurrent := func(isFinal bool) {
		pa.finalizeMessageToCreate(toCreate, content, keyboardObj, isFinal)
		if toCreate.Media == nil && toCreate.Content == "" && toCreate.Markdown == nil && toCreate.MessageReference == nil {
			return
		}
		res, err := pa.Api.PostC2CMessage(qctx, userOpenID, toCreate)
		if err != nil {
			pa.Session.Parent.Logger.Error("official qq 发送单聊消息失败：" + err.Error())
			lastErr = err
		} else {
			lastRes = res
		}
	}

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			// QQ官方API中不能发送链接，所以全部进行转写绕过
			content += textLinkStrip(e.Content)
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             e.ReplySeq,
				IgnoreGetMessageError: true,
			}
			toCreate.MessageReference = msgRef
		case *message.ImageElement:
			url, data, err := pa.prepareMediaMessage(e.File)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送单聊消息时，获取本地图片数据失败：" + err.Error())
				continue
			}
			fMsg := &C2CRichMediaMessage{
				FileType:   1,
				URL:        url,
				FileData:   data,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostC2CMessage(qctx, userOpenID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送单聊消息时，准备图片信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			toCreate.Media = &dto.MediaInfo{
				FileInfo: media.FileInfo,
			}
		case *message.RecordElement:
			url, data, err := pa.prepareMediaMessage(e.File)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送单聊消息时，获取本地语音数据失败：" + err.Error())
				continue
			}
			fMsg := &C2CRichMediaMessage{
				FileType:   3,
				URL:        url,
				FileData:   data,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostC2CMessage(qctx, userOpenID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送单聊消息时，准备语音信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			toCreate.Media = &dto.MediaInfo{
				FileInfo: media.FileInfo,
			}
		}
	}

	sendCurrent(true)
	return lastRes, lastErr
}

func (pa *PlatformAdapterOfficialQQ) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	rowID, ok := VarGetValueStr(ctx, "$tMsgID")
	if !ok {
		rowID, ok = VarGetValueStr(ctx, "$tEventID")
	}
	if !ok {
		// TODO：允许主动消息发送，并校验频率
		pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：无法直接发送消息")
		return
	}
	groupId, idType := pa.mustExtractID(uid)

	maxLen := 900
	totalLen := len(text)
	if totalLen > 900*5 {
		maxLen = totalLen/5 + 50
		if maxLen > 2800 {
			maxLen = 2800
		}
	}
	textList := utils.SplitLongText(text, maxLen, utils.DefaultSplitPaginationHint)
	if !pa.Session.Parent.Config.OfficialQQUseMarkdown {
		if len(textList) > 5 {
			textList = textList[:5]
		}
	}

	if pa.Session.Parent.Config.OfficialQQUseMarkdown && len(textList) > 1 {
		cacheID := generateCacheID()
		pa.addToPaginationCache(cacheID, textList)

		keyboardObj := pa.buildPaginationKeyboard(cacheID, 0, len(textList))

		switch idType {
		case OpenQQGroupOpenid:
			msg, err := pa.sendQQGroupMsgRaw(ctx, rowID, groupId, textList[0], keyboardObj)
			if err == nil && msg != nil {
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     textList[0],
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		case OpenQQCHChannel:
			msg, err := pa.sendQQChannelMsgRaw(ctx, rowID, groupId, textList[0], keyboardObj)
			if err == nil && msg != nil {
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     textList[0],
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		default:
			pa.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", uid, idType)
		}
		return
	}

	for _, t := range textList {
		switch idType {
		case OpenQQGroupOpenid:
			msg, err := pa.sendQQGroupMsgRaw(ctx, rowID, groupId, t, nil)
			if err == nil && msg != nil {
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     t,
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		case OpenQQCHChannel:
			msg, err := pa.sendQQChannelMsgRaw(ctx, rowID, groupId, t, nil)
			if err == nil && msg != nil {
				pa.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     t,
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		default:
			pa.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", uid, idType)
			return
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) sendQQGroupMsgRaw(ctx *MsgContext, rowMsgID, groupID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content  string
		msgRef   *dto.MessageReference
	)

	toCreate := pa.initMessageToCreate(ctx, rowMsgID)

	var lastRes *dto.Message
	var lastErr error

	sendCurrent := func(isFinal bool) {
		pa.finalizeMessageToCreate(toCreate, content, keyboardObj, isFinal)
		if toCreate.Media == nil && toCreate.Content == "" && toCreate.Markdown == nil && toCreate.MessageReference == nil {
			return
		}
		res, err := pa.Api.PostGroupMessage(qctx, groupID, toCreate)
		if err != nil {
			pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：" + err.Error())
			lastErr = err
		} else {
			lastRes = res
		}
	}

	for _, element := range elems {
		switch elem := element.(type) {
		case *message.TextElement:
			// QQ官方API中不能发送链接，所以全部进行转写绕过
			content += textLinkStrip(elem.Content)
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             elem.ReplySeq,
				IgnoreGetMessageError: true,
			}
			toCreate.MessageReference = msgRef
		case *message.AtElement:
			pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持 AT 他人，跳过该部分")
		case *message.ImageElement:
			url, data, err := pa.prepareMediaMessage(elem.File)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，获取本地图片数据失败：" + err.Error())
				continue
			}
			fMsg := &dto.MessageMediaToCreate{
				FileType:   1,
				URL:        url,
				FileData:   data,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备图片信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			decodedFileInfo, decodeErr := base64.StdEncoding.DecodeString(media.FileInfo)
			if decodeErr != nil {
				decodedFileInfo = []byte(media.FileInfo)
			}
			toCreate.Media = &dto.MediaInfo{
				FileInfo: decodedFileInfo,
			}
		case *message.RecordElement:
			url, data, err := pa.prepareMediaMessage(elem.File)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，获取本地语音数据失败：" + err.Error())
				continue
			}
			fMsg := &dto.MessageMediaToCreate{
				FileType:   3,
				URL:        url,
				FileData:   data,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备语音信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			decodedFileInfo, decodeErr := base64.StdEncoding.DecodeString(media.FileInfo)
			if decodeErr != nil {
				decodedFileInfo = []byte(media.FileInfo)
			}
			toCreate.Media = &dto.MediaInfo{
				FileInfo: decodedFileInfo,
			}
		}
	}

	sendCurrent(true)
	return lastRes, lastErr
}

func (pa *PlatformAdapterOfficialQQ) sendQQChannelMsgRaw( /* ctx */ _ *MsgContext, rowMsgID, channelID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content  string
		msgRef   *dto.MessageReference
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
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             e.ReplySeq,
				IgnoreGetMessageError: true,
			}
		}
	}

	toCreate := pa.initMessageToCreate(nil, rowMsgID)
	toCreate.MessageReference = msgRef
	pa.finalizeMessageToCreate(toCreate, content, keyboardObj, true)

	res, err := pa.Api.PostMessage(qctx, channelID, toCreate)
	if err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送频道消息失败：" + err.Error())
	}
	return res, err
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
	return fmt.Sprintf("OpenQQ-Group-T:%s", groupOpenID)
}

func formatDiceIDOfficialQQMemberOpenID(botID, groupOpenID, memberOpenID string) string {
	// 在没有qq_unionid时的临时方案
	return fmt.Sprintf("OpenQQ-User-T:%s", memberOpenID)
}

func formatDiceIDOfficialQQUserOpenID(botID, userOpenID string) string {
	// 单聊用户OpenID格式
	return fmt.Sprintf("OpenQQ-User-T:%s", userOpenID)
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
		if len(lst) >= 2 {
			return lst[1], "", OpenQQGroupOpenid
		}
		return lst[0], "", OpenQQGroupOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-Member-T:") {
		temp := text[len("OpenQQ-Member-T:"):]
		lst := strings.Split(temp, "-")
		if len(lst) >= 3 {
			return lst[2], lst[1], OpenQQUserOpenid
		}
		if len(lst) == 2 {
			return lst[1], lst[0], OpenQQUserOpenid
		}
		return lst[0], "", OpenQQUserOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-User-T:") {
		temp := text[len("OpenQQ-User-T:"):]
		lst := strings.Split(temp, "-")
		if len(lst) >= 2 {
			return lst[1], "", OpenQQUserOpenid
		}
		return lst[0], "", OpenQQUserOpenid
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

	// 签名验证
	if !pa.verifyWebhookSignature(body, r.Header) {
		log.Error("official qq webhook: 签名验证失败")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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
	payload.Session = &dto.Session{AppID: pa.AppID}

	// 响应确认
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"op": 12, // HTTP回调确认
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("official qq webhook: 响应确认失败: ", err)
	}

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

// verifyWebhookSignature 验证Webhook签名
func (pa *PlatformAdapterOfficialQQ) verifyWebhookSignature(body []byte, header http.Header) bool {
	if pa.AppSecret == "" {
		return true
	}
	pass, err := signature.Verify(pa.AppSecret, header, body)
	if err != nil {
		return false
	}
	return pass
}

type C2CRichMediaMessage struct {
	FileType   int    `json:"file_type"`
	URL        string `json:"url,omitempty"`
	SrvSendMsg bool   `json:"srv_send_msg"`
	FileData   []byte `json:"file_data,omitempty"`
}

func (msg *C2CRichMediaMessage) GetEventID() string {
	return ""
}

func (msg *C2CRichMediaMessage) GetSendType() dto.SendType {
	return dto.RichMedia
}

func isLocalOrNonPublic(urlStr string) bool {
	if urlStr == "" {
		return true
	}
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return true
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return true
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
			return true
		}
	} else if host == "localhost" {
		return true
	}
	return false
}

func getElementBytes(elem *message.FileElement) ([]byte, error) {
	if elem == nil {
		return nil, errors.New("nil element")
	}
	if elem.Stream != nil {
		if seeker, ok := elem.Stream.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
		return io.ReadAll(elem.Stream)
	}
	pathOrUrl := elem.URL
	if pathOrUrl == "" {
		pathOrUrl = elem.File
	}
	if pathOrUrl == "" {
		return nil, errors.New("no file path or url")
	}
	fileElem, err := message.FilepathToFileElement(pathOrUrl)
	if err != nil {
		return nil, err
	}
	if fileElem.Stream == nil {
		return nil, errors.New("failed to get stream")
	}
	return io.ReadAll(fileElem.Stream)
}
