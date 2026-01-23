package dice

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sacOO7/gowebsocket"
	"golang.org/x/time/rate"

	"sealdice-core/message"
	"sealdice-core/utils/satori"
)

type PlatformAdapterSealChat struct {
	Session  *IMSession    `json:"-" yaml:"-"`
	EndPoint *EndPointInfo `json:"-" yaml:"-"`

	ConnectURL string                    `json:"connectUrl" yaml:"connectUrl"` // 连接地址
	Token      string                    `json:"token"      yaml:"token"`
	Socket     *gowebsocket.Socket       `json:"-"          yaml:"-"`
	EchoMap    SyncMap[string, chan any] `json:"-"          yaml:"-"`
	UserID     string                    `json:"-"          yaml:"-"`
	assetCache SyncMap[string, struct{}] `json:"-"          yaml:"-"`

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

const sealChatAssetSrcPrefix = "sealchat://asset/"
const sealChatAssetUploadTimeout = 5 * time.Second

// encodeMessage 将消息文本转换为 Satori 格式（支持图片/文件 asset_id）
func (pa *PlatformAdapterSealChat) encodeMessage(content string) string {
	elems := message.ConvertStringMessage(content)
	return pa.encodeMessageFromElements(elems)
}

func (pa *PlatformAdapterSealChat) ensureSealChatAsset(data []byte, contentType string, filename string) (string, bool) {
	if len(data) == 0 {
		return "", false
	}
	sum := sha256.Sum256(data)
	assetID := hex.EncodeToString(sum[:])
	if pa.assetCache.Exists(assetID) {
		return assetID, true
	}

	payload := map[string]any{
		"asset_id":     assetID,
		"content_type": contentType,
		"data":         base64.StdEncoding.EncodeToString(data),
	}
	if filename != "" {
		payload["filename"] = filename
	}

	echo, ch := pa.sendAPIWithEcho("asset.upload", payload)
	defer pa.EchoMap.Delete(echo)

	select {
	case resp := <-ch:
		if respMap, ok := resp.(map[string]any); ok {
			if okVal, ok := respMap["ok"].(bool); ok && okVal {
				pa.assetCache.Store(assetID, struct{}{})
				return assetID, true
			}
		}
		return assetID, false
	case <-time.After(sealChatAssetUploadTimeout):
		return assetID, false
	}
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
			// 分离部署：优先使用 asset_id，失败时回退 Base64 data URL
			if e.File.Stream != nil {
				data, err := io.ReadAll(io.LimitReader(e.File.Stream, 10*1024*1024)) // 限制 10MB
				if err != nil {
					continue
				}
				contentType := e.File.ContentType
				if contentType == "" {
					contentType = "image/png"
				}
				if assetID, ok := pa.ensureSealChatAsset(data, contentType, e.File.File); ok {
					node.Attrs["src"] = sealChatAssetSrcPrefix + assetID
				} else {
					b64 := base64.StdEncoding.EncodeToString(data)
					node.Attrs["src"] = fmt.Sprintf("data:%s;base64,%s", contentType, b64)
				}
			} else if e.File.URL != "" {
				// HTTP URL 直接使用（大小写不敏感）
				urlLower := strings.ToLower(e.File.URL)
				if strings.HasPrefix(urlLower, "http://") || strings.HasPrefix(urlLower, "https://") || strings.HasPrefix(urlLower, sealChatAssetSrcPrefix) {
					node.Attrs["src"] = e.File.URL
				} else if strings.HasPrefix(e.File.URL, "base64://") {
					// 已经是 base64 格式
					b64Data := e.File.URL[9:]
					data, err := base64.StdEncoding.DecodeString(b64Data)
					if err == nil {
						if assetID, ok := pa.ensureSealChatAsset(data, "image/png", e.File.File); ok {
							node.Attrs["src"] = sealChatAssetSrcPrefix + assetID
						} else {
							node.Attrs["src"] = fmt.Sprintf("data:image/png;base64,%s", b64Data)
						}
					} else {
						node.Attrs["src"] = fmt.Sprintf("data:image/png;base64,%s", b64Data)
					}
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
				if assetID, ok := pa.ensureSealChatAsset(data, contentType, e.File); ok {
					node.Attrs["src"] = sealChatAssetSrcPrefix + assetID
				} else {
					b64 := base64.StdEncoding.EncodeToString(data)
					node.Attrs["src"] = fmt.Sprintf("data:%s;base64,%s", contentType, b64)
				}
			} else if e.URL != "" {
				urlLower := strings.ToLower(e.URL)
				if strings.HasPrefix(urlLower, "http://") || strings.HasPrefix(urlLower, "https://") || strings.HasPrefix(urlLower, sealChatAssetSrcPrefix) {
					node.Attrs["src"] = e.URL
				} else if strings.HasPrefix(e.URL, "base64://") {
					b64Data := e.URL[9:]
					data, err := base64.StdEncoding.DecodeString(b64Data)
					if err == nil {
						if assetID, ok := pa.ensureSealChatAsset(data, "application/octet-stream", e.File); ok {
							node.Attrs["src"] = sealChatAssetSrcPrefix + assetID
						} else {
							node.Attrs["src"] = fmt.Sprintf("data:application/octet-stream;base64,%s", b64Data)
						}
					} else {
						node.Attrs["src"] = fmt.Sprintf("data:application/octet-stream;base64,%s", b64Data)
					}
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

func (pa *PlatformAdapterSealChat) sendAPIWithEcho(api string, data any) (string, chan any) {
	echo := gonanoid.Must()
	ch := make(chan any, 1)
	pa.EchoMap.Store(echo, ch)
	pa._sendJSON(pa.Socket, &satori.ScApiMsgPayload{
		Api:  api,
		Echo: echo,
		Data: data,
	})
	return echo, ch
}

func (pa *PlatformAdapterSealChat) sendAPI(api string, data any) chan any {
	_, ch := pa.sendAPIWithEcho(api, data)
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
