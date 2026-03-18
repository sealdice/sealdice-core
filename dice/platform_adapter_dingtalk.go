package dice

import (
	"fmt"
	"strings"
	"sync"
	"time"

	dingtalk "github.com/Szzrain/DingTalk-go"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"

	"sealdice-core/message"
)

type PlatformAdapterDingTalk struct {
	Session       *IMSession        `json:"-"           yaml:"-"`
	ClientID      string            `json:"clientID"    yaml:"clientID"`
	Token         string            `json:"token"       yaml:"token"`
	RobotCode     string            `json:"robotCode"   yaml:"robotCode"`
	CoolAppCode   string            `json:"coolAppCode" yaml:"coolAppCode"`
	EndPoint      *EndPointInfo     `json:"-"           yaml:"-"`
	IntentSession *dingtalk.Session `json:"-"           yaml:"-"`
	sessionMu     sync.RWMutex
	sessionOpened bool
}

func (pa *PlatformAdapterDingTalk) DoRelogin() bool {
	if err := pa.closeSessionLocked(); err != nil {
		pa.Session.Parent.Logger.Error("Dingtalk 断开连接失败: ", err)
		return false
	}
	pa.Session.Parent.Logger.Infof("正在启用DingTalk服务……")
	if err := pa.openSessionLocked(); err != nil {
		pa.Session.Parent.Logger.Errorf("与DingTalk服务进行连接时出错:%s", err.Error())
		pa.EndPoint.State = 3
		pa.EndPoint.Enable = false
		return false
	}
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	return true
}

func (pa *PlatformAdapterDingTalk) SetEnable(enable bool) {
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用DingTalk服务……")
		if err := pa.openSessionLocked(); err != nil {
			pa.Session.Parent.Logger.Errorf("与DingTalk服务进行连接时出错:%s", err.Error())
			pa.EndPoint.State = 3
			pa.EndPoint.Enable = false
			return
		}
		pa.EndPoint.State = 1
		pa.EndPoint.Enable = true
	} else {
		if err := pa.closeSessionLocked(); err != nil {
			pa.Session.Parent.Logger.Error("Dingtalk 断开连接失败: ", err)
			return
		}
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
	}
}

func (pa *PlatformAdapterDingTalk) QuitGroup(ctx *MsgContext, id string) {
	pa.SendToGroup(ctx, id, "不支持此功能, 请手动移除机器人", "")
}

func (pa *PlatformAdapterDingTalk) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterDingTalk) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterDingTalk) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	msg := dingtalk.MessageSampleText{Content: text}
	rawUserID := ExtractDingTalkUserID(uid)

	pa.sessionMu.RLock()
	if pa.IntentSession == nil || !pa.sessionOpened {
		pa.sessionMu.RUnlock()
		pa.Session.Parent.Logger.Warn("Dingtalk session 未开启，忽略私聊发送")
		return
	}
	session := pa.IntentSession
	pa.sessionMu.RUnlock()

	messageID, err := session.MessagePrivateSend(rawUserID, pa.RobotCode, &msg)
	if err != nil {
		pa.Session.Parent.Logger.Error("Dingtalk SendToPerson Error: ", err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "DINGTALK",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: messageID,
	}, flag)
}

func (pa *PlatformAdapterDingTalk) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	msg := dingtalk.MessageSampleText{Content: text}
	rawGroupID := ExtractDingTalkGroupID(uid)

	pa.sessionMu.RLock()
	if pa.IntentSession == nil || !pa.sessionOpened {
		pa.sessionMu.RUnlock()
		pa.Session.Parent.Logger.Warn("Dingtalk session 未开启，忽略群聊发送")
		return
	}
	session := pa.IntentSession
	pa.sessionMu.RUnlock()

	messageID, err := session.MessageGroupSend(rawGroupID, pa.RobotCode, pa.CoolAppCode, &msg)
	if err != nil {
		pa.Session.Parent.Logger.Error("Dingtalk SendToGroup Error: ", err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "DINGTALK",
		MessageType: "group",
		Message:     text,
		GroupID:     uid,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: messageID,
	}, flag)
}

func (pa *PlatformAdapterDingTalk) SetGroupCardName(ctx *MsgContext, name string) {
}

func (pa *PlatformAdapterDingTalk) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	fileElement, err := message.FilepathToFileElement(path)
	if err == nil {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterDingTalk) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	fileElement, err := message.FilepathToFileElement(path)
	if err == nil {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterDingTalk) MemberBan(groupID string, userID string, duration int64) {
}

func (pa *PlatformAdapterDingTalk) MemberKick(groupID string, userID string) {
}

func (pa *PlatformAdapterDingTalk) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterDingTalk) RecallMessage(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterDingTalk) GetGroupInfoAsync(groupID string) {

}

func (pa *PlatformAdapterDingTalk) OnChatReceive(_ *dingtalk.Session, data *chatbot.BotCallbackDataModel) {
	groupInfo, ok := pa.Session.ServiceAtNew.Load(FormatDiceIDDingTalkGroup(data.ConversationId))
	if ok {
		groupInfo.GroupName = data.ConversationTitle
	}
	msg := pa.toStdMessage(data)
	pa.Session.Execute(pa.EndPoint, msg, false)
}

func (pa *PlatformAdapterDingTalk) OnGroupJoined(_ *dingtalk.Session, data *dingtalk.GroupJoinedEvent) {
	palogger := pa.Session.Parent.Logger
	msg := &Message{
		Platform: "DINGTALK",
		RawID:    data.EventId,
		GroupID:  FormatDiceIDDingTalkGroup(data.OpenConversationId),
		Sender: SenderBase{
			UserID: FormatDiceIDDingTalk(data.Operator),
		},
	}
	palogger.Info("Dingtalk OnGroupJoined: ", data)
	pa.CoolAppCode = data.CoolAppCode
	pa.RobotCode = data.RobotCode
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	logger := pa.Session.Parent.Logger
	ep := pa.EndPoint
	ctx := &MsgContext{MessageType: "group", EndPoint: ep, Session: pa.Session, Dice: pa.Session.Parent}
	gi := SetBotOnAtGroup(ctx, msg.GroupID)
	gi.InviteUserID = msg.Sender.UserID
	gi.EnteredTime = time.Now().Unix()
	ctx.Player = &GroupPlayerInfo{}
	logger.Infof("发送入群致辞，群: <%s>(%d)", "%未知群名%", data.OpenConversationId)
	text := DiceFormatTmpl(ctx, "核心:骰子进群")
	for _, i := range ctx.SplitText(text) {
		pa.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), "")
	}
	// 触发扩展钩子
	if groupInfo, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID); ok {
		groupInfo.TriggerExtHook(ctx.Dice, func(ext *ExtInfo) func() {
			if ext.OnGroupJoined == nil {
				return nil
			}
			return func() { ext.OnGroupJoined(ctx, msg) }
		})
	}
}

func (pa *PlatformAdapterDingTalk) toStdMessage(data *chatbot.BotCallbackDataModel) *Message {
	msg := &Message{
		Platform: "DINGTALK",
		RawID:    data.MsgId,
		Message:  data.Text.Content,
		Sender: SenderBase{
			Nickname: data.SenderNick,
			UserID:   FormatDiceIDDingTalk(data.SenderStaffId),
		},
		Time: time.Now().Unix(),
	}
	if data.IsAdmin {
		msg.Sender.GroupRole = "admin"
	}
	if data.ConversationType == "2" {
		msg.GroupID = FormatDiceIDDingTalkGroup(data.ConversationId)
		msg.MessageType = "group"
	} else {
		msg.MessageType = "private"
	}
	return msg
}

func (pa *PlatformAdapterDingTalk) Serve() int {
	pa.EndPoint.UserID = FormatDiceIDDingTalk(pa.RobotCode)
	if pa.EndPoint.Nickname == "" {
		pa.EndPoint.Nickname = "DingTalkBot"
	}
	logger := pa.Session.Parent.Logger
	logger.Info("Dingtalk Serve")

	pa.sessionMu.Lock()
	defer pa.sessionMu.Unlock()

	if pa.IntentSession == nil {
		pa.IntentSession = dingtalk.New(pa.ClientID, pa.Token)
		pa.IntentSession.AddEventHandler(pa.OnChatReceive)
		pa.IntentSession.AddEventHandler(pa.OnGroupJoined)
	}

	if err := pa.IntentSession.Open(); err != nil {
		pa.sessionOpened = false
		return 1
	}
	pa.sessionOpened = true

	logger.Info("Dingtalk 连接成功")
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return 0
}

func FormatDiceIDDingTalk(diceDingTalk string) string {
	return fmt.Sprintf("DINGTALK:%s", diceDingTalk)
}

func FormatDiceIDDingTalkGroup(diceDingTalk string) string {
	return fmt.Sprintf("DINGTALK-Group:%s", diceDingTalk)
}

func ExtractDingTalkUserID(id string) string {
	if strings.HasPrefix(id, "DINGTALK:") {
		return id[len("DINGTALK:"):]
	}
	return id
}

func ExtractDingTalkGroupID(id string) string {
	if strings.HasPrefix(id, "DINGTALK-Group:") {
		return id[len("DINGTALK-Group:"):]
	}
	return id
}

func (pa *PlatformAdapterDingTalk) closeSessionLocked() error {
	pa.sessionMu.Lock()
	defer pa.sessionMu.Unlock()

	if pa.IntentSession == nil || !pa.sessionOpened {
		pa.sessionOpened = false
		return nil
	}
	err := pa.IntentSession.Close()
	pa.sessionOpened = false
	pa.IntentSession = nil
	return err
}

func (pa *PlatformAdapterDingTalk) openSessionLocked() error {
	pa.sessionMu.Lock()
	defer pa.sessionMu.Unlock()

	if pa.IntentSession == nil {
		pa.IntentSession = dingtalk.New(pa.ClientID, pa.Token)
		pa.IntentSession.AddEventHandler(pa.OnChatReceive)
		pa.IntentSession.AddEventHandler(pa.OnGroupJoined)
	}
	if pa.sessionOpened {
		return nil
	}
	if err := pa.IntentSession.Open(); err != nil {
		pa.sessionOpened = false
		return err
	}
	pa.sessionOpened = true
	return nil
}
