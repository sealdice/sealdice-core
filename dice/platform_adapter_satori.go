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
	"sealdice-core/utils/satori"
	"strings"
	"time"

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
	// dm := d.Parent

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
	defer resp.Body.Close()
	if err != nil {
		log.Error("dial:", err)
		pa.EndPoint.State = 3
		return 1
	}
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
	if logins == nil || len(logins) < 1 {
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
					if pa.CancelFunc != nil {
						pa.CancelFunc()
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
		pa.sendMsgRaw(ctx, "private:"+id, text, flag, "私聊")
	} else {
		log.Errorf("satori %s 平台暂不支持私聊消息发送", pa.Platform)
	}
}

func (pa *PlatformAdapterSatori) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	pa.sendMsgRaw(ctx, UserIDExtract(groupID), text, flag, "群")
}

func (pa *PlatformAdapterSatori) sendMsgRaw(ctx *MsgContext, channelID string, text string, flag string, msgType string) {
	log := pa.Session.Parent.Logger
	req, err := json.Marshal(map[string]interface{}{
		"channel_id": channelID,
		"content":    pa.encodeMessage(text),
	})
	if err != nil {
		log.Errorf("satori 发送%s(%s)消息失败: %s", msgType, channelID, err)
		return
	}
	data, err := pa.post("message.create", bytes.NewBuffer(req))
	if err != nil {
		log.Errorf("satori 发送%s(%s)消息失败: %s", msgType, channelID, err)
		return
	}
	var messages []SatoriMessage
	err = json.Unmarshal(data, &messages)
	if err != nil {
		log.Errorf("satori 发送%s(%s)消息失败: %s", msgType, channelID, err)
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
	log.Errorf("satori %s 平台暂不支持群聊消息发送", pa.Platform)
}

func (pa *PlatformAdapterSatori) MemberBan(groupID string, userID string, duration int64) {
	pa.Session.Parent.Logger.Errorf("satori 禁言群(%s)内成员(%s)失败：尚不支持", groupID, userID)
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

func (pa *PlatformAdapterSatori) decodeMessage(messageEvent *SatoriEvent) *Message {
	log := pa.Session.Parent.Logger
	if messageEvent.Message == nil {
		log.Errorf("satori 消息解析失败：消息为空")
		return nil
	}
	msg := new(Message)
	msg.RawID = messageEvent.Message.ID

	msgRoot := satori.ElementParse(messageEvent.Message.Content)
	content := strings.Builder{}
	msgRoot.Traverse(func(el *satori.Element) {
		switch el.Type {
		case "at":
			if el.Attrs["role"] != "all" {
				content.WriteString(fmt.Sprintf("[CQ:at,qq=%s]", el.Attrs["id"]))
			}
		case "img":
			content.WriteString(fmt.Sprintf("[CQ:image,file=%s]", el.Attrs["src"]))
		case "root":
			// pass
		default:
			content.WriteString(el.ToString())
		}
	})

	msg.Message = content.String()
	msg.Platform = pa.Platform

	sender := SenderBase{}
	sender.UserID = formatDiceIDSatori(pa.Platform, messageEvent.User.ID)
	sender.Nickname = messageEvent.User.Name
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

func (pa *PlatformAdapterSatori) encodeMessage(content string) string {
	dice := pa.Session.Parent
	elems := dice.ConvertStringMessage(content)
	var msg strings.Builder
	for _, elem := range elems {
		switch e := elem.(type) {
		case *TextElement:
			msg.WriteString(satori.ContentEscape(e.Content))
		case *AtElement:
			if e.Target == "all" {
				msg.WriteString("<at type=\"all\"/>")
			} else {
				msg.WriteString(fmt.Sprintf("<at id=\"%s\"/>", e.Target))
			}
		}
	}
	return msg.String()
}

func (pa *PlatformAdapterSatori) handleEvent(event SatoriPayload[SatoriEvent]) {
	s := pa.Session
	switch event.Body.Type {
	case "message-created": // 消息创建
		msg := pa.decodeMessage(event.Body)
		if msg != nil {
			s.Execute(pa.EndPoint, msg, false)
		}
	case "message-updated": // 消息编辑
	case "message-deleted": // 消息撤回
	case "guild-added": // 加入群组
	case "guild-updated": // 群组被修改
	case "guild-removed": // 退出群组
	case "guild-request": // 收到入群邀请
	case "friend-request": // 收到好友申请
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
