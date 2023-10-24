package dice

import (
	"fmt"
	dingtalk "github.com/Szzrain/DingTalk-go"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"strings"
	"time"
)

type PlatformAdapterDingTalk struct {
	Session       *IMSession        `yaml:"-" json:"-"`
	ClientID      string            `yaml:"clientID" json:"clientID"`
	Token         string            `yaml:"token" json:"token"`
	RobotCode     string            `yaml:"robotCode" json:"robotCode"`
	CoolAppCode   string            `yaml:"coolAppCode" json:"coolAppCode"`
	EndPoint      *EndPointInfo     `yaml:"-" json:"-"`
	IntentSession *dingtalk.Session `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterDingTalk) DoRelogin() bool {
	err := pa.IntentSession.Close()
	if err != nil {
		pa.Session.Parent.Logger.Error("Dingtalk 断开连接失败: ", err)
		return false
	}
	pa.Session.Parent.Logger.Infof("正在启用DingTalk服务……")
	if pa.IntentSession == nil {
		pa.Serve()
		return false
	}
	err = pa.IntentSession.Open()
	if err != nil {
		pa.Session.Parent.Logger.Errorf("与DingTalk服务进行连接时出错:%s", err.Error())
		pa.EndPoint.State = 3
		pa.EndPoint.Enable = false
		return false
	}
	return true
}

func (pa *PlatformAdapterDingTalk) SetEnable(enable bool) {
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用DingTalk服务……")
		if pa.IntentSession == nil {
			pa.Serve()
			return
		}
		err := pa.IntentSession.Open()
		if err != nil {
			pa.Session.Parent.Logger.Errorf("与DingTalk服务进行连接时出错:%s", err.Error())
			pa.EndPoint.State = 3
			pa.EndPoint.Enable = false
			return
		}
	} else {
		err := pa.IntentSession.Close()
		if err != nil {
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

func (pa *PlatformAdapterDingTalk) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	msg := dingtalk.MessageSampleText{Content: text}
	rawUserID := ExtractDingTalkUserID(uid)
	_, err := pa.IntentSession.MessagePrivateSend(rawUserID, pa.RobotCode, &msg)
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
	}, flag)
}

func (pa *PlatformAdapterDingTalk) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	msg := dingtalk.MessageSampleText{Content: text}
	rawGroupID := ExtractDingTalkGroupID(uid)
	_, err := pa.IntentSession.MessageGroupSend(rawGroupID, pa.RobotCode, pa.CoolAppCode, &msg)
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
	}, flag)
}

func (pa *PlatformAdapterDingTalk) SetGroupCardName(groupID string, userID string, name string) {
}

func (pa *PlatformAdapterDingTalk) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterDingTalk) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
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

func (pa *PlatformAdapterDingTalk) GetGroupInfoAsync(groupID string) {

}

func (pa *PlatformAdapterDingTalk) OnChatReceive(_ *dingtalk.Session, data *chatbot.BotCallbackDataModel) {
	//palogger := pa.Session.Parent.Logger
	pa.EndPoint.UserID = FormatDiceIDDingTalk(data.ChatbotUserId)
	if pa.Session.ServiceAtNew[FormatDiceIDDingTalkGroup(data.ConversationId)] != nil {
		pa.Session.ServiceAtNew[FormatDiceIDDingTalkGroup(data.ConversationId)].GroupName = data.ConversationTitle
	}
	//palogger.Info("Dingtalk OnChatReceive: ", data.Text.Content, " Sender: ", data.SenderId, " CorpId: ", data.SenderCorpId, " Conversation Type: ", data.ConversationType)
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
	for _, i := range strings.Split(text, "###SPLIT###") {
		pa.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), "")
	}
	if ctx.Session.ServiceAtNew[msg.GroupID] != nil {
		for _, i := range ctx.Session.ServiceAtNew[msg.GroupID].ActivatedExtList {
			if i.OnGroupJoined != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnGroupJoined(ctx, msg)
				})
			}
		}
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
	logger := pa.Session.Parent.Logger
	logger.Info("Dingtalk Serve")
	pa.IntentSession = dingtalk.New(pa.ClientID, pa.Token)
	pa.IntentSession.AddEventHandler(pa.OnChatReceive)
	pa.IntentSession.AddEventHandler(pa.OnGroupJoined)
	err := pa.IntentSession.Open()
	if err != nil {
		return 1
	}
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
