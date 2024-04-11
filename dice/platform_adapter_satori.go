package dice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fy0/lockfree"

	"sealdice-core/message"
	"sealdice-core/utils/satori"

	"github.com/gorilla/websocket"
)

const SatoriProtocolVersion = "v1"

type PlatformAdapterSatori struct {
	Session     *IMSession    `yaml:"-" json:"-"`
	EndPoint    *EndPointInfo `yaml:"-" json:"-"`
	DiceServing bool          `yaml:"-"`

	Version  string `yaml:"version" json:"version"`
	Platform string `yaml:"platform" json:"platform"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Token    string `yaml:"token" json:"token"`

	wsUrl   *url.URL
	httpUrl *url.URL

	conn       *websocket.Conn
	Ctx        context.Context    `yaml:"-" json:"-"`
	CancelFunc context.CancelFunc `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterSatori) Serve() int {
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	d := pa.Session.Parent

	wsUrl := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", pa.Host, pa.Port),
		Path:   fmt.Sprintf("/%s/events", pa.Version),
	}
	httpUrl := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", pa.Host, pa.Port),
		Path:   fmt.Sprintf("/%s", pa.Version),
	}
	pa.Ctx, pa.CancelFunc = context.WithCancel(context.Background())
	defer func() {
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
	}()

	log.Infof("connecting to %s", wsUrl.String())
	conn, resp, err := websocket.DefaultDialer.DialContext(pa.Ctx, wsUrl.String(), nil)
	if err != nil {
		log.Error("dial:", err)
		pa.EndPoint.State = 3
		return 1
	}
	defer resp.Body.Close()
	pa.conn = conn
	pa.EndPoint.State = 2

	// 鉴权
	auth := &SatoriPayload[SatoriIdentify]{
		Op:   SatoriOpIdentify,
		Body: &SatoriIdentify{Token: pa.Token},
	}
	authData, _ := json.Marshal(auth)
	err = conn.WriteMessage(websocket.TextMessage, authData)
	if err != nil {
		log.Error("auth failed:", err)
		pa.EndPoint.State = 3
		return 1
	}

	_, authRespData, err := conn.ReadMessage()
	if err != nil {
		log.Error("auth failed:", err)
		pa.EndPoint.State = 3
		return 1
	}
	var authResp SatoriPayload[SatoriReady]
	err = json.Unmarshal(authRespData, &authResp)
	if err != nil {
		log.Error("auth failed:", err)
		pa.EndPoint.State = 3
		return 1
	}
	log.Debugf("satori auth resp:%+v", authResp)
	logins := authResp.Body.Logins
	var login *SatoriLogin
	if len(logins) < 1 {
		log.Error("invalid satori login info")
		pa.EndPoint.State = 3
		return 1
	} else if len(logins) > 1 {
		for _, loginResp := range logins {
			if loginResp.Platform == pa.Platform {
				login = loginResp
				break
			}
		}
		if login == nil {
			log.Error("invalid satori login info")
			pa.EndPoint.State = 3
			return 1
		}
	} else {
		login = logins[0]
	}

	pa.wsUrl = &wsUrl
	pa.httpUrl = &httpUrl
	pa.EndPoint.State = 1
	ep.UserID = formatDiceIDSatori(pa.Platform, login.SelfID)
	if login.User != nil {
		ep.Nickname = login.User.Name
	}
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	log.Infof("satori 连接成功，账号<%s>(%s)", pa.EndPoint.Nickname, pa.EndPoint.UserID)

	go func() {
		for {
			select {
			case <-pa.Ctx.Done():
				log.Info("satori finished")
				return
			default:
				msgType, msgData, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Error("satori read failed:", err)
					}
					if pa.CancelFunc != nil {
						pa.CancelFunc()
						pa.CancelFunc = nil
					}
					return
				}
				switch msgType {
				case websocket.TextMessage:
					log.Debugf("message=%s", msgData)
					// handle event
					var payload SatoriPayload[any]
					err = json.Unmarshal(msgData, &payload)
					if err != nil {
						log.Error("satori data unmarshal failed:", err)
						continue
					}
					if payload.Op == SatoriOpEvent {
						var event SatoriPayload[SatoriEvent]
						err = json.Unmarshal(msgData, &event)
						if err != nil {
							log.Error("satori event unmarshal failed:", err)
							continue
						}
						pa.handleEvent(event)
					}
				case websocket.CloseMessage:
					log.Debugf("satori closed")
					return
				default:
					log.Debugf("message[type=%d]=%s", msgType, msgData)
				}
			}
		}
	}()

	// 更新好友列表
	go pa.refreshFriends()
	// 更新群列表
	go pa.refreshGroups()

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			// heartbeat
			hb, _ := json.Marshal(SatoriPayload[any]{
				Op:   SatoriOpPing,
				Body: nil,
			})
			err := conn.WriteMessage(websocket.TextMessage, hb)
			if err != nil {
				log.Error("satori heartbeat failed:", err)
				if pa.CancelFunc != nil {
					pa.CancelFunc()
				}
			}
		case <-pa.Ctx.Done():
			ticker.Stop()
			_ = conn.Close()
			pa.conn = nil
			return 0
		}
	}
}

// refreshFriends 更新好友信息
func (pa *PlatformAdapterSatori) refreshFriends() {
	session := pa.Session
	d := session.Parent
	log := d.Logger
	dm := d.Parent

	next := ""
	for {
		req := make(map[string]interface{})
		if next != "" {
			req["next"] = next
		}
		reqJson, _ := json.Marshal(req)
		data, err := pa.post("friend.list", bytes.NewBuffer(reqJson))
		if err != nil {
			log.Error("satori 获取好友列表失败:", err)
			return
		}
		var friendPage struct {
			Data []SatoriUser `json:"data"`
			Next string       `json:"next"`
		}
		err = json.Unmarshal(data, &friendPage)
		if err != nil {
			log.Error("satori 获取好友列表失败:", err)
			return
		}
		next = friendPage.Next
		for _, friend := range friendPage.Data {
			name := "未知用户"
			if friend.Name != "" {
				name = friend.Name
			}
			if friend.Nick != "" {
				name = friend.Nick
			}
			if name == "" {
				continue
			}
			userID := formatDiceIDSatori(pa.Platform, friend.ID)
			dm.UserNameCache.Set(userID, &GroupNameCacheItem{
				Name: name,
				time: time.Now().Unix(),
			})
		}
		if next == "" {
			break
		}
	}
}

// refreshGroups 更新群信息
func (pa *PlatformAdapterSatori) refreshGroups() {
	session := pa.Session
	d := session.Parent
	log := d.Logger
	dm := d.Parent

	next := ""
	for {
		req := make(map[string]interface{})
		if next != "" {
			req["next"] = next
		}
		reqJson, _ := json.Marshal(req)
		data, err := pa.post("guild.list", bytes.NewBuffer(reqJson))
		if err != nil {
			log.Error("satori 获取群列表失败:", err)
			return
		}
		var groupPage struct {
			Data []SatoriGuild `json:"data"`
			Next string        `json:"next"`
		}
		next = groupPage.Next
		err = json.Unmarshal(data, &groupPage)
		if err != nil {
			log.Error("satori 获取群列表失败:", err)
			return
		}
		for _, group := range groupPage.Data {
			if group.Name == "" {
				continue
			}
			groupID := formatDiceIDSatoriGroup(pa.Platform, group.ID)
			dm.GroupNameCache.Set(groupID, &GroupNameCacheItem{
				Name: group.Name,
				time: time.Now().Unix(),
			})

			groupInfo := session.ServiceAtNew[groupID]
			if groupInfo == nil {
				// 新检测到群
				ctx := &MsgContext{
					Session:  session,
					EndPoint: pa.EndPoint,
					Dice:     d,
				}
				SetBotOnAtGroup(ctx, groupID)
			} else if group.Name != "" && groupInfo.GroupName != group.Name {
				// 更新群名
				groupInfo.GroupName = group.Name
				groupInfo.UpdatedAtTime = time.Now().Unix()
			}

			// 触发群成员更新
			go pa.refreshMembers(group)
		}
		if next == "" {
			break
		}
	}
}

func (pa *PlatformAdapterSatori) refreshMembers(group SatoriGuild) {
	session := pa.Session
	d := session.Parent
	log := d.Logger

	groupID := formatDiceIDSatoriGroup(pa.Platform, group.ID)
	groupInfo := session.ServiceAtNew[groupID]
	next := ""
	for {
		req := map[string]interface{}{
			"guild_id": group.ID,
		}
		if next != "" {
			req["next"] = next
		}

		reqJson, _ := json.Marshal(req)
		data, err := pa.post("guild.member.list", bytes.NewBuffer(reqJson))
		if err != nil {
			log.Error("satori 获取群成员列表失败:", err)
			return
		}
		var memberPage struct {
			Data []SatoriGuildMember `json:"data"`
			Next string              `json:"next"`
		}
		next = memberPage.Next
		err = json.Unmarshal(data, &memberPage)
		if err != nil {
			log.Error("satori 获取群成员列表失败:", err)
			return
		}
		for _, member := range memberPage.Data {
			mem := member
			userID := formatDiceIDSatori(pa.Platform, mem.User.ID)
			if groupInfo != nil {
				p := groupInfo.PlayerGet(d.DBData, userID)
				if p == nil {
					name := mem.Nick
					if name == "" {
						if mem.User.Name != "" {
							name = mem.User.Name
						}
						if mem.User.Nick != "" {
							name = mem.User.Nick
						}
					}
					p = &GroupPlayerInfo{
						Name:          name,
						UserID:        userID,
						ValueMapTemp:  lockfree.NewHashMap(),
						UpdatedAtTime: 0,
					}
					groupInfo.Players.Store(userID, p)
				}
			}
		}
		if next == "" {
			break
		}
	}
}

func (pa *PlatformAdapterSatori) DoRelogin() bool {
	pa.Session.Parent.Logger.Infof("正在启用 satori 连接，请稍后...")
	pa.EndPoint.State = 0
	pa.EndPoint.Enable = false
	if pa.CancelFunc != nil {
		pa.CancelFunc()
	}
	return pa.Serve() == 0
}

func (pa *PlatformAdapterSatori) SetEnable(enable bool) {
	d := pa.Session.Parent
	e := pa.EndPoint
	if enable {
		e.Enable = true
		pa.DiceServing = false
		if pa.conn == nil {
			go ServeQQ(d, e)
		}
	} else {
		e.State = 0
		e.Enable = false
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
	}
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

func (pa *PlatformAdapterSatori) QuitGroup(ctx *MsgContext, _ string) {
	log := pa.Session.Parent.Logger
	log.Errorf("satori %s 平台暂不支持退群", pa.Platform)
}

func (pa *PlatformAdapterSatori) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
	log := pa.Session.Parent.Logger
	if pa.Platform == "QQ" {
		id := UserIDExtract(userID)
		pa.sendMsgRaw(ctx, "private:"+id, text, flag, "private")
	} else {
		log.Errorf("satori %s 平台暂不支持私聊消息发送", pa.Platform)
	}
}

func (pa *PlatformAdapterSatori) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	pa.sendMsgRaw(ctx, UserIDExtract(groupID), text, flag, "group")
}

func (pa *PlatformAdapterSatori) sendMsgRaw(ctx *MsgContext, channelID string, text string, flag string, msgType string) {
	log := pa.Session.Parent.Logger
	req, err := json.Marshal(map[string]interface{}{
		"channel_id": channelID,
		"content":    pa.encodeMessage(text),
	})
	var msgTypeStr string
	if msgType == "private" {
		msgTypeStr = "私聊"
	} else {
		msgTypeStr = "群"
	}
	if err != nil {
		log.Errorf("satori 发送%s(%s)消息失败: %s", msgTypeStr, channelID, err)
		return
	}
	data, err := pa.post("message.create", bytes.NewBuffer(req))
	if err != nil {
		log.Errorf("satori 发送%s(%s)消息失败: %s", msgTypeStr, channelID, err)
		return
	}
	var messages []SatoriMessage
	err = json.Unmarshal(data, &messages)
	if err != nil {
		log.Errorf("satori 发送%s(%s)消息失败: %s", msgTypeStr, channelID, err)
		return
	}
}

func (pa *PlatformAdapterSatori) SetGroupCardName(ctx *MsgContext, name string) {
	log := pa.Session.Parent.Logger
	log.Errorf("satori %s 平台暂不支持设置群成员名片", pa.Platform)
}

func (pa *PlatformAdapterSatori) SendFileToPerson(ctx *MsgContext, userID string, path string, flag string) {
	log := pa.Session.Parent.Logger
	log.Errorf("satori %s 平台暂不支持私聊文件发送", pa.Platform)
}

func (pa *PlatformAdapterSatori) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	log := pa.Session.Parent.Logger
	log.Errorf("satori %s 平台暂不支持群聊文件发送", pa.Platform)
}

func (pa *PlatformAdapterSatori) MemberBan(groupID string, userID string, duration int64) {
	pa.Session.Parent.Logger.Errorf("satori %s 平台暂不支持禁言群(%s)内成员%s", pa.Platform, groupID, userID)
}

func (pa *PlatformAdapterSatori) MemberKick(groupID string, userID string) {
	req, err := json.Marshal(map[string]interface{}{
		"guild_id":  UserIDExtract(groupID),
		"user_id":   UserIDExtract(userID),
		"permanent": false,
	})
	if err != nil {
		pa.Session.Parent.Logger.Errorf("satori 踢出群(%s)内成员(%s)失败：%s", groupID, userID, err)
		return
	}
	_, err = pa.post("guild.member.kick", bytes.NewBuffer(req))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("satori 踢出群(%s)内成员(%s)失败：%s", groupID, userID, err)
		return
	}
}

func (pa *PlatformAdapterSatori) GetGroupInfoAsync(groupID string) {
	logger := pa.Session.Parent.Logger
	req, err := json.Marshal(map[string]interface{}{
		"guild_id": UserIDExtract(groupID),
	})
	if err != nil {
		logger.Errorf("satori 获取群(%s)信息失败：%s", groupID, err)
		return
	}
	data, err := pa.post("guild.get", bytes.NewBuffer(req))
	if err != nil {
		logger.Errorf("satori 获取群(%s)信息失败：%s", groupID, err)
		return
	}
	var groupInfo SatoriGuild
	err = json.Unmarshal(data, &groupInfo)
	if err != nil {
		logger.Errorf("satori 获取群(%s)信息失败：%s", groupID, err)
		return
	}

	go pa.refreshGroups()
}

func (pa *PlatformAdapterSatori) EditMessage(ctx *MsgContext, msgID, message string) {
	log := pa.Session.Parent.Logger
	log.Errorf("satori %s 平台暂不支持编辑消息", pa.Platform)
}

func (pa *PlatformAdapterSatori) RecallMessage(ctx *MsgContext, msgID string) {
	log := pa.Session.Parent.Logger
	log.Errorf("satori %s 平台暂不支持撤回消息", pa.Platform)
}

func (pa *PlatformAdapterSatori) post(resource string, body io.Reader) ([]byte, error) {
	apiUrl := pa.httpUrl.String() + "/" + resource
	client := http.Client{}
	request, _ := http.NewRequest(http.MethodPost, apiUrl, body)
	request.Header.Add("Content-Type", "application/json")
	if pa.Token != "" {
		request.Header.Add("Authorization", "Bearer "+pa.Token)
	}
	request.Header.Add("X-Platform", pa.Platform)
	request.Header.Add("X-Self-ID", UserIDExtract(pa.EndPoint.UserID))
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(resp.Body)
	case http.StatusBadRequest:
		return nil, errors.New("请求参数有误(400)")
	case http.StatusUnauthorized:
		return nil, errors.New("缺失鉴权(401)")
	case http.StatusForbidden:
		return nil, errors.New("权限不足(403)")
	case http.StatusNotFound:
		return nil, errors.New("未找到资源(404)")
	case http.StatusMethodNotAllowed:
		return nil, errors.New("请求方法不允许(405)")
	default:
		errMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("(%d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("(%d) - %s", resp.StatusCode, errMsg)
	}
}

func (pa *PlatformAdapterSatori) toStdMessage(messageEvent *SatoriEvent) *Message {
	session := pa.Session
	d := session.Parent
	log := d.Logger
	dm := d.Parent

	if messageEvent.Message == nil {
		log.Errorf("satori 消息解析失败：消息为空")
		return nil
	}
	msg := new(Message)
	msg.RawID = messageEvent.Message.ID
	msg.Message = decodeMessage(messageEvent.Message.Content)
	msg.Platform = pa.Platform

	sender := SenderBase{}
	sender.UserID = formatDiceIDSatori(pa.Platform, messageEvent.User.ID)
	userName, _ := dm.UserNameCache.Get(sender.UserID)
	if userName != nil {
		sender.Nickname = userName.(*GroupNameCacheItem).Name
	}
	if messageEvent.User.Name != "" {
		sender.Nickname = messageEvent.User.Name
	}
	if messageEvent.User.Nick != "" {
		sender.Nickname = messageEvent.User.Nick
	}
	if messageEvent.Guild != nil {
		// 群聊消息
		msg.MessageType = "group"
		msg.GroupID = formatDiceIDSatoriGroup(pa.Platform, messageEvent.Guild.ID)
		if messageEvent.Member.Nick != "" {
			sender.Nickname = messageEvent.Member.Nick
		}
	} else {
		// 私聊消息
		msg.MessageType = "private"
	}
	if sender.Nickname == "" {
		sender.Nickname = "未知用户"
	}
	msg.Sender = sender

	return msg
}

func decodeMessage(text string) string {
	msgRoot := satori.ElementParse(text)
	content := strings.Builder{}
	msgRoot.Traverse(func(el *satori.Element) {
		switch el.Type {
		case "at":
			if el.Attrs["role"] != "all" {
				content.WriteString(fmt.Sprintf("[CQ:at,qq=%s]", el.Attrs["id"]))
			}
		case "img":
			content.WriteString(fmt.Sprintf("[CQ:image,url=%s]", el.Attrs["src"]))
		case "root":
			// pass
		default:
			content.WriteString(el.ToString())
		}
	})
	return content.String()
}

func (pa *PlatformAdapterSatori) encodeMessage(content string) string {
	elems := message.ConvertStringMessage(content)
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
		case *message.FileElement:
			node := &satori.Element{
				// Type:  "file",
				Attrs: make(satori.Dict),
			}
			if e.File != "" {
				node.Attrs["title"] = e.File
			}
			if e.Stream != nil {
				fileName := e.File
				if fileName == "" {
					fileName = "temp"
				}
				temp, _ := os.CreateTemp("", fileName)
				_, err := io.Copy(temp, e.Stream)
				if err != nil {
					continue
				}
				node.Attrs["src"] = "file:///" + temp.Name()
			} else if e.URL != "" {
				node.Attrs["src"] = e.URL
			}
			// cc 0.2.2 发送 file QQ 会直接爆炸
			// msg.WriteString(node.ToString())
		case *message.ImageElement:
			file := e.File
			node := &satori.Element{
				Type:  "img",
				Attrs: make(satori.Dict),
			}
			if file.File != "" {
				node.Attrs["title"] = file.File
			}
			if file.Stream != nil {
				fileName := file.File
				if fileName == "" {
					fileName = "temp"
				}
				temp, _ := os.CreateTemp("", fileName)
				_, err := io.Copy(temp, file.Stream)
				if err != nil {
					continue
				}
				node.Attrs["src"] = "file:///" + temp.Name()
			} else if e.File.URL != "" {
				node.Attrs["src"] = e.File.URL
			}
			msg.WriteString(node.ToString())
		}
	}
	result := msg.String()
	return result
}

func (pa *PlatformAdapterSatori) handleEvent(event SatoriPayload[SatoriEvent]) {
	s := pa.Session
	switch event.Body.Type {
	case "message-created": // 消息创建
		msg := pa.toStdMessage(event.Body)
		if msg != nil {
			s.Execute(pa.EndPoint, msg, false)
		}
	case "message-updated": // 消息编辑
		pa.editMessageHandle(event.Body)
	case "message-deleted": // 消息撤回
		pa.deleteMessageHandle(event.Body)
	case "guild-added": // 加入群组
		// satori 协议有规定该事件，但是 chronocat 目前并未支持，尚未测试
		// pa.guildAddedHandle(event.Body)
	case "guild-updated": // 群组被修改
	case "guild-removed": // 退出群组
		// satori 协议有规定该事件，但是 chronocat 目前并未支持，尚未测试
		// pa.guildRemovedHandle(event.Body)
	case "guild-request": // 收到入群邀请
		// satori 协议有规定该事件，但是 chronocat 目前并未支持，尚未测试
		// pa.guildRequestHandle(event.Body)
	case "friend-request": // 收到好友申请
		// satori 协议有规定该事件，但是 chronocat 目前并未支持，尚未测试
		// pa.friendRequestHandle(event.Body)
	}
}

func (pa *PlatformAdapterSatori) deleteMessageHandle(e *SatoriEvent) {
	msg := new(Message)
	msg.Time = e.Message.CreatedAt
	msg.RawID = e.Message.ID
	msg.Sender.UserID = formatDiceIDSatori(pa.Platform, e.User.ID)
	msg.Sender.Nickname = e.User.Name
	if e.User.Nick != "" {
		msg.Sender.Nickname = e.User.Nick
	}
	if e.Channel.Type == SatoriDirectChannel {
		msg.MessageType = "private"
	} else {
		msg.MessageType = "group"
		msg.GroupID = formatDiceIDSatoriGroup(pa.Platform, e.Channel.ID)
	}
	mctx := &MsgContext{Session: pa.Session, EndPoint: pa.EndPoint, Dice: pa.Session.Parent, MessageType: msg.MessageType}
	pa.Session.OnMessageDeleted(mctx, msg)
}

func (pa *PlatformAdapterSatori) editMessageHandle(e *SatoriEvent) {
	msg := new(Message)
	msg.Time = e.Message.UpdatedAt
	msg.RawID = e.Message.ID
	msg.Message = decodeMessage(e.Message.Content)
	msg.Platform = pa.Platform
	if e.Channel.Type == SatoriDirectChannel {
		msg.MessageType = "private"
	} else {
		msg.MessageType = "group"
		msg.GroupID = formatDiceIDSatoriGroup(pa.Platform, e.Channel.ID)
	}
	mctx := &MsgContext{
		Session:     pa.Session,
		EndPoint:    pa.EndPoint,
		Dice:        pa.Session.Parent,
		MessageType: msg.MessageType,
		Player:      &GroupPlayerInfo{},
	}
	pa.Session.OnMessageEdit(mctx, msg)
}

//nolint:unused
func (pa *PlatformAdapterSatori) guildAddedHandle(e *SatoriEvent) {

}

//nolint:unused
func (pa *PlatformAdapterSatori) guildRemovedHandle(e *SatoriEvent) {

}

//nolint:unused
func (pa *PlatformAdapterSatori) guildRequestHandle(e *SatoriEvent) {
	d := pa.Session.Parent
	dm := d.Parent
	log := d.Logger

	uid := formatDiceIDSatori(pa.Platform, e.User.ID)
	userName := dm.TryGetUserName(uid)
	guildID := formatDiceIDSatoriGroup(pa.Platform, e.Guild.ID)
	guildName := dm.TryGetGroupName(guildID)
	log.Infof("satori: 收到平台 %s 加群邀请: 群组<%s>(%s) 邀请人:<%s>(%s)",
		pa.Platform, guildName, guildID, userName, uid)

	eid := e.ID.String()
	// 邀请人在黑名单上
	banInfo, ok := d.BanList.GetByID(uid)
	if ok {
		if banInfo.Rank == BanRankBanned && d.BanList.BanBehaviorRefuseInvite {
			pa.sendGuildRequestResult(eid, false, "黑名单")
			return
		}
	}
	// 信任模式，如果不是信任，又不是 master 则拒绝拉群邀请
	isMaster := d.IsMaster(uid)
	if d.TrustOnlyMode && ((banInfo != nil && banInfo.Rank != BanRankTrusted) && !isMaster) {
		pa.sendGuildRequestResult(eid, false, "只允许骰主设置信任的人拉群")
		return
	}
	// 群在黑名单上
	banInfo, ok = d.BanList.GetByID(guildID)
	if ok {
		if banInfo.Rank == BanRankBanned {
			pa.sendGuildRequestResult(eid, false, "群黑名单")
			return
		}
	}
	// 拒绝加群
	if d.RefuseGroupInvite {
		pa.sendGuildRequestResult(eid, false, "设置拒绝加群")
		return
	}
	pa.sendGuildRequestResult(eid, true, "")
}

// sendGuildRequestResult 发送入群邀请处理结果
//
//nolint:unused
func (pa *PlatformAdapterSatori) sendGuildRequestResult(id string, approve bool, comment string) {
	d := pa.Session.Parent
	log := d.Logger
	req := map[string]any{
		"message_id": id,
		"approve":    approve,
		"comment":    comment,
	}
	reqJson, _ := json.Marshal(req)
	_, err := pa.post("guild.approve", bytes.NewBuffer(reqJson))
	if err != nil {
		log.Error("satori 发送入群邀请处理结果失败:", err)
		return
	}
}

//nolint:unused
func (pa *PlatformAdapterSatori) friendRequestHandle(e *SatoriEvent) {
	s := pa.Session
	d := s.Parent
	dm := d.Parent
	log := d.Logger

	uid := formatDiceIDSatori(pa.Platform, e.User.ID)
	userName := dm.TryGetUserName(uid)
	log.Infof("satori: 收到平台 %s 好友请求: 申请人:<%s>(%s)", pa.Platform, userName, uid)

	eid := e.ID.String()
	// 申请人在黑名单上
	banInfo, ok := d.BanList.GetByID(uid)
	if ok {
		if banInfo.Rank == BanRankBanned && d.BanList.BanBehaviorRefuseInvite {
			pa.sendGuildRequestResult(eid, false, "为被禁止用户，准备自动拒绝")
			return
		}
	}

	if strings.TrimSpace(d.FriendAddComment) == "" {
		pa.sendFriendRequestResult(eid, true, "")
	} else {
		pa.sendFriendRequestResult(eid, false, "存在好友问题校验，准备自动拒绝，请联系骰主")
	}
}

// sendFriendRequestResult 发送好友申请处理结果
//
//nolint:unused
func (pa *PlatformAdapterSatori) sendFriendRequestResult(id string, approve bool, comment string) {
	d := pa.Session.Parent
	log := d.Logger
	req := map[string]any{
		"message_id": id,
		"approve":    approve,
		"comment":    comment,
	}
	reqJson, _ := json.Marshal(req)
	_, err := pa.post("friend.approve", bytes.NewBuffer(reqJson))
	if err != nil {
		log.Error("satori 发送好友申请处理结果失败:", err)
		return
	}
}

func formatDiceIDSatori(platform string, diceSatori string) string {
	if platform != "" {
		return fmt.Sprintf("%s:%s", platform, diceSatori)
	} else {
		return fmt.Sprintf("Satori(unknown):%s", diceSatori)
	}
}

func formatDiceIDSatoriGroup(platform, diceSatori string) string {
	if platform != "" {
		return fmt.Sprintf("%s-Group:%s", platform, diceSatori)
	}
	return fmt.Sprintf("Satori(unknown)-Group:%s", diceSatori)
}

//nolint:unused
func formatDiceIDSatoriCh(platform, diceSatori string) string {
	if platform != "" {
		return fmt.Sprintf("%s-CH:%s", platform, diceSatori)
	}
	return fmt.Sprintf("Satori(unknown)-CH:%s", diceSatori)
}
