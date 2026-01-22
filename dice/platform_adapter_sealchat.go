package dice

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	ds "github.com/sealdice/dicescript"
	"golang.org/x/time/rate"

	"sealdice-core/dice/service"
	"sealdice-core/message"
	"sealdice-core/utils/satori"

	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sacOO7/gowebsocket"
)

// 哨兵错误
var errMissingCharIdOrName = errors.New("missing id or name")

type PlatformAdapterSealChat struct {
	Session  *IMSession    `json:"-" yaml:"-"`
	EndPoint *EndPointInfo `json:"-" yaml:"-"`

	ConnectURL string                    `json:"connectUrl" yaml:"connectUrl"` // 连接地址
	Token      string                    `json:"token"      yaml:"token"`
	Socket     *gowebsocket.Socket       `json:"-"          yaml:"-"`
	EchoMap    SyncMap[string, chan any] `json:"-"          yaml:"-"`
	UserID     string                    `json:"-"          yaml:"-"`

	Reconnecting    bool `json:"-" yaml:"-"`
	RetryTimes      int  `json:"-" yaml:"-"`
	RetryTimesLimit int  `json:"-" yaml:"-"`

	// 心跳相关
	heartbeatStop chan struct{}
	lastPong      int64

	// 角色卡写入速率限制器 (60次/分钟)
	characterSetLimiter *rate.Limiter
}

func (pa *PlatformAdapterSealChat) Serve() int {
	if !strings.HasPrefix(pa.ConnectURL, "ws://") && !strings.HasPrefix(pa.ConnectURL, "wss://") {
		pa.ConnectURL = "ws://" + pa.ConnectURL
	}
	socket := gowebsocket.New(pa.ConnectURL)
	pa.Socket = &socket
	pa.EndPoint.Nickname = "SealChat Bot"
	pa.EndPoint.UserID = "SEALCHAT:BOT"
	pa.RetryTimesLimit = 15
	// 初始化角色卡写入速率限制器: 60次/分钟，允许突发5次
	pa.characterSetLimiter = rate.NewLimiter(rate.Every(time.Minute/60), 5)
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	pa.socketSetup()
	socket.Connect()
	return 0
}

func (pa *PlatformAdapterSealChat) _sendJSON(socket *gowebsocket.Socket, data any) bool {
	// TODO: 修改上游代码，使其支持发送 JSON
	marshal, err := json.Marshal(data)
	if err != nil {
		return false
	}
	socket.SendText(string(marshal))
	return true
}

func (pa *PlatformAdapterSealChat) socketSetup() {
	ep := pa.EndPoint
	log := pa.Session.Parent.Logger
	socket := pa.Socket
	socket.OnConnected = func(socket gowebsocket.Socket) {
		ep.State = 2
		ep.Enable = true

		d := pa.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)

		pa._sendJSON(&socket, satori.GatewayPayloadStructure{
			Op: satori.OpIdentify,
			Body: map[string]string{
				"token": pa.Token,
			},
		})

		log.Info("SealChat 建立连接，正在发送身份验证信息")
		pa.Reconnecting = false
	}
	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		gatewayMsg := satori.GatewayPayloadStructure2{}
		err := json.Unmarshal([]byte(message), &gatewayMsg)
		if len(message) == 0 {
			return
		}
		// fmt.Printf("XXXXX: %s\n", message)

		solved := false
		if err == nil && gatewayMsg.Op != nil {
			switch *gatewayMsg.Op {
			case satori.OpReady:
				info := gatewayMsg.Body.(map[string]any)
				if info["errorMsg"] != nil {
					log.Infof("SealChat 连接失败: %s", info["errorMsg"])
					ep.State = 3
				} else {
					data := struct {
						Body struct {
							User satori.User `json:"user"`
						} `json:"body"`
					}{}
					err = json.Unmarshal([]byte(message), &data)
					if err != nil {
						log.Errorf("SealChat 解析用户信息失败: %s", err)
					} else {
						pa.UserID = data.Body.User.ID
						ep.UserID = FormatDiceIDSealChat(data.Body.User.ID)
						ep.Nickname = data.Body.User.Nick
						ep.State = 1
						log.Infof("SealChat 连接成功: %s", ep.Nickname)

						// 握手成功，通过验证
						pa.RetryTimes = 0
						pa.RetryTimesLimit = 15
					}

					go func() {
						// 等一会再发，因为好像有的模块会在这个事件之后注册指令
						time.Sleep(time.Duration(5) * time.Second)
						pa.registerCommands()
					}()
				}
				// 启动心跳
				pa.startHeartbeat()
				solved = true
			case satori.OpPing:
				// 收到服务端 Ping，回复 Pong
				pa._sendJSON(&socket, satori.GatewayPayloadStructure{
					Op:   satori.OpPong,
					Body: map[string]any{},
				})
				solved = true
			case satori.OpPong:
				// 收到 Pong，更新最后心跳时间
				atomic.StoreInt64(&pa.lastPong, time.Now().Unix())
				solved = true
			case satori.OpEvent:
				pa.dispatchMessage(message)
				solved = true
			default:
				log.Infof("SealChat: %s", message)
			}
		}

		if solved {
			return
		}

		apiMsg := satori.ScApiMsgPayload{}
		err = json.Unmarshal([]byte(message), &apiMsg)
		if err == nil {
			// 处理来自 SealChat 的 API 请求（角色卡同步等）
			if apiMsg.Api != "" && apiMsg.Echo != "" {
				pa.handleApiRequest(apiMsg)
				return
			}
			// 处理 API 响应
			if x, ok := pa.EchoMap.Load(apiMsg.Echo); ok {
				x <- apiMsg.Data
			}
		}
	}
	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Errorf("SealChat websocket出现错误: %s", err)
		pa.stopHeartbeat()
		if !socket.IsConnected {
			pa.Reconnecting = false
			time.Sleep(time.Duration(10) * time.Second)
			if !pa.tryReconnect(*pa.Socket) {
				log.Errorf("短时间内连接失败次数过多，不再进行重连")
				ep.State = 3
			}
		}
	}
	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Info("与SealChat服务器断开连接，尝试进行重连")
		pa.stopHeartbeat()
		time.Sleep(time.Duration(2) * time.Second)
		if !pa.tryReconnect(*pa.Socket) {
			ep.State = 3
			log.Errorf("到达连接次数上限，不再进行重连")
		}
	}
	pa.Socket = socket
}

func (pa *PlatformAdapterSealChat) tryReconnect(socket gowebsocket.Socket) bool {
	log := pa.Session.Parent.Logger
	if socket.IsConnected {
		return true
	}
	if pa.Reconnecting {
		return true
	}
	pa.Reconnecting = true

	if !pa.EndPoint.Enable {
		pa.Reconnecting = false
		return true
	}

	if pa.RetryTimes >= pa.RetryTimesLimit {
		pa.Reconnecting = false
		return false
	}

	pa.RetryTimes++
	log.Infof("尝试重新连接SealChat中[%d/%d]", pa.RetryTimes, pa.RetryTimesLimit)
	socket = gowebsocket.New(pa.ConnectURL)
	pa.Socket = &socket
	pa.socketSetup()
	socket.Connect()

	return true
}

// startHeartbeat 启动心跳协程
func (pa *PlatformAdapterSealChat) startHeartbeat() {
	pa.stopHeartbeat()
	pa.heartbeatStop = make(chan struct{})
	atomic.StoreInt64(&pa.lastPong, time.Now().Unix())
	log := pa.Session.Parent.Logger

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if pa.Socket == nil || !pa.Socket.IsConnected {
					return
				}
				// 发送 Ping
				pa._sendJSON(pa.Socket, satori.GatewayPayloadStructure{
					Op:   satori.OpPing,
					Body: map[string]any{},
				})
				// 检查超时（45秒无响应则断开）
				last := atomic.LoadInt64(&pa.lastPong)
				if time.Now().Unix()-last > 45 {
					log.Warn("SealChat 心跳超时，断开连接")
					pa.Socket.Close()
					return
				}
			case <-pa.heartbeatStop:
				return
			}
		}
	}()
}

// stopHeartbeat 停止心跳协程
func (pa *PlatformAdapterSealChat) stopHeartbeat() {
	if pa.heartbeatStop != nil {
		select {
		case <-pa.heartbeatStop:
			// 已关闭
		default:
			close(pa.heartbeatStop)
		}
		pa.heartbeatStop = nil
	}
}

// encodeMessage 将消息文本转换为 Satori 格式（支持图片 Base64）
func (pa *PlatformAdapterSealChat) encodeMessage(content string) string {
	elems := message.ConvertStringMessage(content)
	return pa.encodeMessageFromElements(elems)
}

// encodeMessageFromElements 将消息元素列表转换为 Satori 格式
func (pa *PlatformAdapterSealChat) encodeMessageFromElements(elems []message.IMessageElement) string {
	var msg strings.Builder
	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			msg.WriteString(satori.ContentEscape(e.Content))
		case *message.AtElement:
			if e.Target == "all" {
				msg.WriteString(`<at type="all"/>`)
			} else {
				msg.WriteString(fmt.Sprintf(`<at id="%s"/>`, e.Target))
			}
		case *message.ImageElement:
			if e.File == nil {
				continue
			}
			node := &satori.Element{
				Type:  "img",
				Attrs: make(satori.Dict),
			}
			if e.File.File != "" {
				node.Attrs["title"] = e.File.File
			}
			// 分离部署：优先使用 Base64 data URL
			if e.File.Stream != nil {
				data, err := io.ReadAll(io.LimitReader(e.File.Stream, 10*1024*1024)) // 限制 10MB
				if err != nil {
					continue
				}
				contentType := e.File.ContentType
				if contentType == "" {
					contentType = "image/png"
				}
				b64 := base64.StdEncoding.EncodeToString(data)
				node.Attrs["src"] = fmt.Sprintf("data:%s;base64,%s", contentType, b64)
			} else if e.File.URL != "" {
				// HTTP URL 直接使用（大小写不敏感）
				urlLower := strings.ToLower(e.File.URL)
				if strings.HasPrefix(urlLower, "http://") || strings.HasPrefix(urlLower, "https://") {
					node.Attrs["src"] = e.File.URL
				} else if strings.HasPrefix(e.File.URL, "base64://") {
					// 已经是 base64 格式
					b64Data := e.File.URL[9:]
					node.Attrs["src"] = fmt.Sprintf("data:image/png;base64,%s", b64Data)
				}
			}
			if node.Attrs["src"] != nil {
				msg.WriteString(node.ToString())
			}
		case *message.FileElement:
			// 文件元素：同样转为 Base64
			node := &satori.Element{
				Type:  "file",
				Attrs: make(satori.Dict),
			}
			if e.File != "" {
				node.Attrs["title"] = e.File
			}
			if e.Stream != nil {
				data, err := io.ReadAll(io.LimitReader(e.Stream, 10*1024*1024)) // 限制 10MB
				if err != nil {
					continue
				}
				contentType := e.ContentType
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				b64 := base64.StdEncoding.EncodeToString(data)
				node.Attrs["src"] = fmt.Sprintf("data:%s;base64,%s", contentType, b64)
			} else if e.URL != "" {
				urlLower := strings.ToLower(e.URL)
				if strings.HasPrefix(urlLower, "http://") || strings.HasPrefix(urlLower, "https://") {
					node.Attrs["src"] = e.URL
				}
			}
			if node.Attrs["src"] != nil {
				msg.WriteString(node.ToString())
			}
		case *message.ReplyElement:
			msg.WriteString(fmt.Sprintf(`<quote id="%s"/>`, e.ReplySeq))
		}
	}
	return msg.String()
}

func (pa *PlatformAdapterSealChat) GetGroupInfoAsync(_ string) {}

func FormatDiceIDSealChat(id string) string {
	// 避免双前缀：如果已有前缀则直接返回
	if strings.HasPrefix(id, "SEALCHAT:") {
		return id
	}
	return fmt.Sprintf("SEALCHAT:%s", id)
}

func FormatDiceIDSealChatPrivate(id string) string {
	// 避免双前缀：如果已有前缀则直接返回
	if strings.HasPrefix(id, "PG-SEALCHAT:") {
		return id
	}
	return fmt.Sprintf("PG-SEALCHAT:%s", id)
}

func FormatDiceIDSealChatGroup(id string) string {
	// 避免双前缀：如果已有前缀则直接返回
	if strings.HasPrefix(id, "SEALCHAT-Group:") {
		return id
	}
	return fmt.Sprintf("SEALCHAT-Group:%s", id)
}

func (pa *PlatformAdapterSealChat) DoRelogin() bool {
	log := pa.Session.Parent.Logger
	pa.Reconnecting = true
	if pa.Socket != nil {
		pa.Socket.Close()
	}

	socket := gowebsocket.New(pa.ConnectURL)
	log.Infof("SealChat 重新连接")
	pa.Socket = &socket
	pa.socketSetup()
	socket.Connect()
	pa.Reconnecting = false
	return true
}

func (pa *PlatformAdapterSealChat) SetEnable(enable bool) {
	log := pa.Session.Parent.Logger
	if enable {
		pa.EndPoint.Enable = true
		log.Infof("Sealchat 连接中")
		if pa.Socket != nil && pa.Socket.IsConnected {
			pa.Reconnecting = true
			pa.Socket.Close()
			socket := gowebsocket.New(pa.ConnectURL)
			pa.Socket = &socket
			pa.socketSetup()
			socket.Connect()
			pa.Reconnecting = false
		} else {
			pa.Reconnecting = true
			socket := gowebsocket.New(pa.ConnectURL)
			pa.Socket = &socket
			pa.socketSetup()
			socket.Connect()
			pa.Reconnecting = false
		}
	} else {
		pa.EndPoint.Enable = false
		pa.Reconnecting = true
		if pa.Socket != nil && pa.Socket.IsConnected {
			pa.Socket.Close()
		}
		pa.Reconnecting = false
	}
}

func (pa *PlatformAdapterSealChat) sendAPI(api string, data any) chan any {
	echo := gonanoid.Must()
	ch := make(chan any, 1)
	pa.EchoMap.Store(echo, ch)
	pa._sendJSON(pa.Socket, &satori.ScApiMsgPayload{
		Api:  api,
		Echo: echo,
		Data: data,
	})
	return ch
}

func ExtractSealChatPrivateChatID(id string, userId string) string {
	if strings.HasPrefix(id, "SEALCHAT:") {
		id1 := id[len("SEALCHAT:"):]
		id2 := userId[len("SEALCHAT:"):]
		if id1 > id2 {
			id1, id2 = id2, id1
		}
		return fmt.Sprintf("%s:%s", id1, id2)
	}
	return id
}

func ExtractSealChatUserID(id string) string {
	if strings.HasPrefix(id, "SEALCHAT:") {
		return id[len("SEALCHAT:"):]
	}
	if strings.HasPrefix(id, "SEALCHAT-Group:") {
		return id[len("SEALCHAT-Group:"):]
	}
	return id
}

func (pa *PlatformAdapterSealChat) _sendTo(ctx *MsgContext, chId string, text string, flag string, msgType string) {
	// 使用 encodeMessage 转换消息格式（支持图片等）
	encodedContent := pa.encodeMessage(text)

	pa.sendAPI("message.create", map[string]any{
		"channel_id": chId,
		"content":    encodedContent,
	})

	var groupID string
	if msgType == "private" {
		groupID = FormatDiceIDSealChatPrivate(chId)
	} else {
		groupID = FormatDiceIDSealChatGroup(chId)
	}
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "SEALCHAT",
		MessageType: msgType,
		Message:     text,
		GroupID:     groupID,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSealChat) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	chId := ExtractSealChatUserID(groupID)
	encodedContent := pa.encodeMessageFromElements(msg)

	pa.sendAPI("message.create", map[string]any{
		"channel_id": chId,
		"content":    encodedContent,
	})

	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "SEALCHAT",
		MessageType: "group",
		Message:     encodedContent,
		GroupID:     FormatDiceIDSealChatGroup(chId),
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSealChat) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	<-pa.sendAPI("channel.private.create", map[string]string{
		"user_id": ExtractSealChatUserID(userID),
	})

	chId := ExtractSealChatPrivateChatID(userID, pa.EndPoint.UserID)
	encodedContent := pa.encodeMessageFromElements(msg)

	pa.sendAPI("message.create", map[string]any{
		"channel_id": chId,
		"content":    encodedContent,
	})

	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "SEALCHAT",
		MessageType: "private",
		Message:     encodedContent,
		GroupID:     FormatDiceIDSealChatPrivate(chId),
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSealChat) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	<-pa.sendAPI("channel.private.create", map[string]string{
		"user_id": ExtractSealChatUserID(uid),
	})

	text = satori.ContentEscape(text)
	gid := ExtractSealChatPrivateChatID(uid, pa.EndPoint.UserID)
	pa._sendTo(ctx, gid, text, flag, "private")
}

func (pa *PlatformAdapterSealChat) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	chId := ExtractSealChatUserID(uid)
	text = satori.ContentEscape(text)
	pa._sendTo(ctx, chId, text, flag, "group")
}

func (pa *PlatformAdapterSealChat) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	fileElement, err := message.FilepathToFileElement(path)
	if err != nil {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
		return
	}
	// 使用 SendSegmentToPerson 发送图片/文件
	pa.SendSegmentToPerson(ctx, uid, []message.IMessageElement{
		&message.ImageElement{File: fileElement},
	}, flag)
}

func (pa *PlatformAdapterSealChat) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	fileElement, err := message.FilepathToFileElement(path)
	if err != nil {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
		return
	}
	// 使用 SendSegmentToGroup 发送图片/文件
	pa.SendSegmentToGroup(ctx, uid, []message.IMessageElement{
		&message.ImageElement{File: fileElement},
	}, flag)
}

func (pa *PlatformAdapterSealChat) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterSealChat) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterSealChat) QuitGroup(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterSealChat) SetGroupCardName(mctx *MsgContext, text string) {
	pa.sendAPI("bot.channel_member.set_name", map[string]string{
		"user_id":    ExtractSealChatUserID(mctx.Player.UserID),
		"channel_id": ExtractSealChatUserID(mctx.Group.GroupID),
		"name":       text,
	})
}

func (pa *PlatformAdapterSealChat) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterSealChat) RecallMessage(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterSealChat) dispatchMessage(msg string) {
	ev := satori.Event{}
	err := json.Unmarshal([]byte(msg), &ev)
	if err != nil {
		pa.Session.Parent.Logger.Error("PlatformAdapterSealChat.dispatchMessage", err)
		return
	}

	switch ev.Type {
	case satori.EventMessageCreated:
		if ev.Message.User.ID == pa.UserID {
			// 自己发的消息，不管
			return
		}
		pa.Session.Execute(pa.EndPoint, pa.toStdMessage(ev.Message), false)
		return
	case satori.EventMessageDeleted:
		stdMsg := pa.toStdMessage(ev.Message)
		// 注; 缺少 User、Channel[导致MessageType出不来]
		mctx := CreateTempCtx(pa.EndPoint, stdMsg)
		pa.Session.OnMessageDeleted(mctx, stdMsg)
		return
	default:
		// fmt.Println("msg", ev.Type, "|", ev)
	}
}

func (pa *PlatformAdapterSealChat) toStdMessage(scMsg *satori.Message) *Message {
	msg := new(Message)

	elRoot := satori.ElementParse(scMsg.Content)
	msg.Time = scMsg.Timestamp

	// TODO: 这里会有一个很怪的行为，也就是结构化数据转cq码，以后改掉

	cqMsg := strings.Builder{}
	elRoot.Traverse(func(el *satori.Element) {
		switch el.Type {
		case "at":
			if el.Attrs["role"] != "all" {
				cqMsg.WriteString(fmt.Sprintf("<@%s>", el.Attrs["id"]))
			}
		case "root":
			// 啥都不做
		default:
			cqMsg.WriteString(el.ToString())
		}
	})

	msg.Message = strings.TrimSpace(cqMsg.String())

	msg.Platform = "SEALCHAT"
	if scMsg.Channel.Type == satori.DirectChannelType {
		msg.MessageType = "private"
		msg.GroupID = FormatDiceIDSealChatPrivate(scMsg.Channel.ID)
	} else {
		msg.MessageType = "group"
		msg.GroupID = FormatDiceIDSealChatGroup(scMsg.Channel.ID)
		msg.GroupName = scMsg.Channel.Name
	}

	msg.RawID = scMsg.ID
	send := new(SenderBase)
	if scMsg.User != nil {
		send.UserID = FormatDiceIDSealChat(scMsg.User.ID)
	}

	if scMsg.Member != nil {
		// 注: 部分消息，比如message-deleted没有member
		send.Nickname = scMsg.Member.Nick
		// 权限处理：解析 Member.Roles 映射到 GroupRole
		send.GroupRole = pa.parseGroupRole(scMsg.Member.Roles)
	}
	if send.Nickname == "" && scMsg.User != nil {
		send.Nickname = scMsg.User.Nick
	}

	if send.Nickname == "" {
		send.Nickname = fmt.Sprintf("用户%4s", scMsg.Channel.ID)
	}
	msg.Sender = *send
	return msg
}

// parseGroupRole 解析 Member.Roles 映射到标准 GroupRole
// 返回值: "owner" | "admin" | "member"
func (pa *PlatformAdapterSealChat) parseGroupRole(roles []string) string {
	for _, role := range roles {
		switch strings.ToLower(role) {
		case "owner", "群主", "creator":
			return "owner"
		case "admin", "管理员", "administrator", "moderator":
			return "admin"
		}
	}
	return "member"
}

// handleApiRequest 处理来自 SealChat 的 API 请求
func (pa *PlatformAdapterSealChat) handleApiRequest(msg satori.ScApiMsgPayload) {
	log := pa.Session.Parent.Logger
	d := pa.Session.Parent

	switch msg.Api {
	case "character.get":
		// 获取角色卡数据
		pa.handleCharacterGet(msg, d)
	case "character.set":
		// 写入角色卡数据
		pa.handleCharacterSet(msg, d)
	case "character.list":
		// 获取用户的角色卡列表
		pa.handleCharacterList(msg, d)
	case "character.new":
		// 新建角色卡并绑定
		pa.handleCharacterNew(msg, d)
	case "character.save":
		// 保存独立卡为角色卡
		pa.handleCharacterSave(msg, d)
	case "character.tag":
		// 绑定/解绑角色卡
		pa.handleCharacterTag(msg, d)
	case "character.untagAll":
		// 从所有群解绑
		pa.handleCharacterUntagAll(msg, d)
	case "character.load":
		// 加载角色卡到独立卡
		pa.handleCharacterLoad(msg, d)
	case "character.delete":
		// 删除角色卡
		pa.handleCharacterDelete(msg, d)
	default:
		log.Warnf("SealChat: unknown API request: %s", msg.Api)
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":    false,
			"error": "unknown api",
		})
	}
}

// sendApiResponse 发送 API 响应
func (pa *PlatformAdapterSealChat) sendApiResponse(echo string, data any) {
	pa._sendJSON(pa.Socket, &satori.ScApiMsgPayload{
		Api:  "",
		Echo: echo,
		Data: data,
	})
}

// handleCharacterGet 处理获取角色卡请求
func (pa *PlatformAdapterSealChat) handleCharacterGet(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 格式化 ID
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)
	formattedUserID := FormatDiceIDSealChat(userID)

	attrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 将属性导出为 map
	attrsData := make(map[string]any)
	if attrs != nil {
		attrs.Range(func(key string, value *ds.VMValue) bool {
			// 将 VMValue 转换为可 JSON 序列化的值
			attrsData[key] = vmValueToAny(value)
			return true
		})
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":   true,
		"data": attrsData,
		"name": attrs.Name,
		"type": attrs.SheetType,
	})
}

// vmValueToAny 将 VMValue 转换为可 JSON 序列化的值
func vmValueToAny(v *ds.VMValue) any {
	if v == nil {
		return nil
	}
	switch v.TypeId {
	case ds.VMTypeInt:
		return v.MustReadInt()
	case ds.VMTypeFloat:
		return v.MustReadFloat()
	case ds.VMTypeString:
		s, _ := v.ReadString()
		return s
	default:
		// 对于复杂类型，尝试转换为字符串
		return v.ToString()
	}
}

// anyToVMValue 将 any 类型转换为 VMValue
func anyToVMValue(v any) *ds.VMValue {
	switch val := v.(type) {
	case float64:
		// JSON 解析时数字默认为 float64
		if val == float64(int64(val)) {
			return ds.NewIntVal(ds.IntType(val))
		}
		return ds.NewFloatVal(val)
	case int:
		return ds.NewIntVal(ds.IntType(val))
	case int64:
		return ds.NewIntVal(ds.IntType(val))
	case string:
		return ds.NewStrVal(val)
	case bool:
		if val {
			return ds.NewIntVal(1)
		}
		return ds.NewIntVal(0)
	default:
		return ds.NewStrVal(fmt.Sprintf("%v", v))
	}
}

// handleCharacterSet 处理写入角色卡请求
// 安全限制：仅允许通过此API写入SealChat平台的角色卡
func (pa *PlatformAdapterSealChat) handleCharacterSet(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)
	attrsData, _ := dataMap["attrs"].(map[string]any)
	nameData, hasName := dataMap["name"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 验证至少提供了 attrs 或 name
	if attrsData == nil && (!hasName || nameData == "") {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing attrs or name"})
		return
	}

	// 格式化 ID
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)
	formattedUserID := FormatDiceIDSealChat(userID)

	// 安全限制：仅允许写入 SealChat 平台的角色卡
	if !strings.HasPrefix(formattedGroupID, "SEALCHAT-Group:") {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "write access denied: only SealChat platform cards can be modified"})
		return
	}

	// 速率限制：60次/分钟
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	attrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 写入属性
	if attrsData != nil {
		for k, v := range attrsData {
			attrs.Store(k, anyToVMValue(v))
		}
	}

	// 更新名称（如果提供）
	if hasName && nameData != "" {
		attrs.Name = nameData
		attrs.SetModified()
	}

	pa.sendApiResponse(msg.Echo, map[string]any{"ok": true})
}

// handleCharacterList 处理获取角色卡列表请求
func (pa *PlatformAdapterSealChat) handleCharacterList(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	userID, _ := dataMap["user_id"].(string)
	if userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing user_id"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	list, err := d.AttrsManager.GetCharacterList(formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 构建返回列表
	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		result = append(result, map[string]any{
			"id":         item.Id,
			"name":       item.Name,
			"sheet_type": item.SheetType,
			"updated_at": item.UpdatedAt,
		})
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":   true,
		"list": result,
	})
}

// getCharIdFromRequest 从请求中获取角色卡ID（支持id或name两种方式）
func (pa *PlatformAdapterSealChat) getCharIdFromRequest(d *Dice, dataMap map[string]any, userID string) (string, string, error) {
	formattedUserID := FormatDiceIDSealChat(userID)

	// 优先使用 id
	if id, ok := dataMap["id"].(string); ok && id != "" {
		// 验证 id 属于该用户且为角色卡类型
		item, err := service.AttrsGetById(d.DBOperator, id)
		if err != nil {
			return "", "", err
		}
		if !item.IsDataExists() {
			return "", "", fmt.Errorf("character not found: %s", id)
		}
		// 安全检查：验证归属用户和类型
		if item.OwnerId != formattedUserID || item.AttrsType != service.AttrsTypeCharacter {
			return "", "", fmt.Errorf("character not owned by user or invalid type")
		}
		return id, item.Name, nil
	}

	// 其次使用 name
	if name, ok := dataMap["name"].(string); ok && name != "" {
		charId, err := d.AttrsManager.CharIdGetByName(formattedUserID, name)
		if err != nil {
			return "", "", err
		}
		if charId == "" {
			return "", "", fmt.Errorf("character not found: %s", name)
		}
		return charId, name, nil
	}

	return "", "", errMissingCharIdOrName
}

// handleCharacterNew 处理新建角色卡请求 (对应 .pc new)
func (pa *PlatformAdapterSealChat) handleCharacterNew(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)
	name, _ := dataMap["name"].(string)
	sheetType, _ := dataMap["sheet_type"].(string)

	if groupID == "" || userID == "" || name == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id, user_id or name"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 检查角色名是否已存在
	if d.AttrsManager.CharCheckExists(formattedUserID, name) {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "character already exists"})
		return
	}

	// 默认 sheet_type
	if sheetType == "" {
		sheetType = "coc7"
	}

	// 创建角色卡
	item, err := d.AttrsManager.CharNew(formattedUserID, name, sheetType)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 绑定到当前群
	if err := d.AttrsManager.CharBind(item.Id, formattedGroupID, formattedUserID); err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":             true,
		"id":             item.Id,
		"name":           name,
		"sheet_type":     sheetType,
		"bound_group_id": formattedGroupID,
	})
}

// handleCharacterSave 处理保存独立卡为角色卡请求 (对应 .pc save)
func (pa *PlatformAdapterSealChat) handleCharacterSave(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)
	name, _ := dataMap["name"].(string)
	sheetType, _ := dataMap["sheet_type"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 加载当前群的独立卡数据
	currentAttrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 如果未提供 name，使用当前角色名
	if name == "" {
		name = currentAttrs.Name
	}
	if name == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "name is required"})
		return
	}

	// 默认 sheet_type
	if sheetType == "" {
		sheetType = currentAttrs.SheetType
	}
	if sheetType == "" {
		sheetType = "coc7"
	}

	var action string
	var charId string

	// 检查角色卡是否已存在
	existingId, _ := d.AttrsManager.CharIdGetByName(formattedUserID, name)
	if existingId == "" {
		// 不存在，创建新卡
		newItem, err := d.AttrsManager.CharNew(formattedUserID, name, sheetType)
		if err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		charId = newItem.Id
		action = "created"

		// 复制当前独立卡数据到新卡
		newAttrs, err := d.AttrsManager.LoadById(charId)
		if err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		currentAttrs.Range(func(key string, value *ds.VMValue) bool {
			newAttrs.Store(key, value)
			return true
		})
		newAttrs.Name = name
		newAttrs.SheetType = sheetType
	} else {
		// 已存在，检查是否被绑定
		bindingGroups := d.AttrsManager.CharGetBindingGroupIdList(existingId)
		if len(bindingGroups) > 0 {
			pa.sendApiResponse(msg.Echo, map[string]any{
				"ok":             false,
				"error":          "character is bound, cannot overwrite",
				"binding_groups": bindingGroups,
			})
			return
		}

		// 覆盖现有卡数据
		charId = existingId
		action = "updated"

		existingAttrs, err := d.AttrsManager.LoadById(charId)
		if err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		// 先清空现有数据，避免残留
		existingAttrs.Clear()
		// 复制源数据
		currentAttrs.Range(func(key string, value *ds.VMValue) bool {
			existingAttrs.Store(key, value)
			return true
		})
		// 同步 Name 和 SheetType
		existingAttrs.Name = name
		existingAttrs.SheetType = sheetType
		existingAttrs.SetModified()
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":     true,
		"id":     charId,
		"name":   name,
		"action": action,
	})
}

// handleCharacterTag 处理绑定/解绑角色卡请求 (对应 .pc tag)
func (pa *PlatformAdapterSealChat) handleCharacterTag(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 检查是否提供了 id 或 name（绑定操作）
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)

	if err != nil && !errors.Is(err, errMissingCharIdOrName) {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if charId != "" {
		// 绑定操作
		if err := d.AttrsManager.CharBind(charId, formattedGroupID, formattedUserID); err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":     true,
			"action": "bind",
			"id":     charId,
			"name":   charName,
		})
	} else {
		// 解绑操作
		currentBindingId, _ := d.AttrsManager.CharGetBindingId(formattedGroupID, formattedUserID)
		if currentBindingId == "" {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "no character bound"})
			return
		}

		// 获取当前绑定角色名
		currentAttrs, _ := d.AttrsManager.LoadById(currentBindingId)
		var unboundName string
		if currentAttrs != nil {
			unboundName = currentAttrs.Name
		}

		// 解绑（绑定空字符串）
		if err := d.AttrsManager.CharBind("", formattedGroupID, formattedUserID); err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":     true,
			"action": "unbind",
			"id":     currentBindingId,
			"name":   unboundName,
		})
	}
}

// handleCharacterUntagAll 处理从所有群解绑请求 (对应 .pc untagAll)
func (pa *PlatformAdapterSealChat) handleCharacterUntagAll(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	userID, _ := dataMap["user_id"].(string)
	groupID, _ := dataMap["group_id"].(string) // 可选，用于获取当前绑定卡

	if userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)

	// 获取角色卡 ID
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)
	if errors.Is(err, errMissingCharIdOrName) {
		// 未提供 id/name，使用当前群绑定的卡
		if groupID != "" {
			formattedGroupID := FormatDiceIDSealChatGroup(groupID)
			charId, _ = d.AttrsManager.CharGetBindingId(formattedGroupID, formattedUserID)
			if charId != "" {
				attrs, _ := d.AttrsManager.LoadById(charId)
				if attrs != nil {
					charName = attrs.Name
				}
			}
		}
	} else if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if charId == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "no character specified or bound"})
		return
	}

	// 执行全部解绑
	unboundGroups := d.AttrsManager.CharUnbindAll(charId)

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":             true,
		"id":             charId,
		"name":           charName,
		"unbound_count":  len(unboundGroups),
		"unbound_groups": unboundGroups,
	})
}

// handleCharacterLoad 处理加载角色卡到独立卡请求 (对应 .pc load)
func (pa *PlatformAdapterSealChat) handleCharacterLoad(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 获取目标角色卡
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 加载目标角色卡
	sourceAttrs, err := d.AttrsManager.LoadById(charId)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 加载当前群的独立卡
	targetAttrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 先清空目标卡数据，避免残留
	targetAttrs.Clear()

	// 复制数据
	sourceAttrs.Range(func(key string, value *ds.VMValue) bool {
		targetAttrs.Store(key, value)
		return true
	})
	targetAttrs.Name = sourceAttrs.Name
	targetAttrs.SheetType = sourceAttrs.SheetType
	targetAttrs.SetModified()

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":         true,
		"id":         charId,
		"name":       charName,
		"sheet_type": sourceAttrs.SheetType,
	})
}

// handleCharacterDelete 处理删除角色卡请求 (对应 .pc del)
func (pa *PlatformAdapterSealChat) handleCharacterDelete(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	userID, _ := dataMap["user_id"].(string)

	if userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	// 获取角色卡
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 检查是否被绑定
	bindingGroups := d.AttrsManager.CharGetBindingGroupIdList(charId)
	if len(bindingGroups) > 0 {
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":             false,
			"error":          "character is bound, use untagAll first",
			"binding_groups": bindingGroups,
		})
		return
	}

	// 删除角色卡
	if err := d.AttrsManager.CharDelete(charId); err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":   true,
		"id":   charId,
		"name": charName,
	})
}

func (pa *PlatformAdapterSealChat) registerCommands() {
	cmdMap := pa.EndPoint.Session.Parent.CmdMap
	m := map[string]string{}
	for k, v := range cmdMap {
		// fmt.Println("??", k, v)
		m[k] = v.ShortHelp
	}

	for _, i := range pa.EndPoint.Session.Parent.ExtList {
		for k, v := range i.GetCmdMap() {
			// fmt.Println("??", k, v)
			m[k] = v.ShortHelp
		}
	}
	mctx := CreateTempCtx(pa.EndPoint, &Message{
		MessageType: "group",
		Sender:      SenderBase{UserID: pa.EndPoint.UserID},
	})
	pa.sendAPI("bot.info.set_name", map[string]string{"name": DiceFormatTmpl(mctx, "核心:骰子名字")})
	pa.sendAPI("bot.command.register", m)
}

func ServeSealChat(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "SEALCHAT" {
		conn := ep.Adapter.(*PlatformAdapterSealChat)
		d.Logger.Infof("SealChat 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接SealChat服务失败")
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}

func NewSealChatConnItem(url string, token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "SEALCHAT"
	conn.ProtocolType = ""
	conn.Enable = true
	conn.RelWorkDir = "extra/sealchat-" + conn.ID
	// Pinenutn: 初始化对应的EchoMap
	conn.Adapter = &PlatformAdapterSealChat{
		EndPoint:   conn,
		ConnectURL: url,
		Token:      token,
	}
	return conn
}
