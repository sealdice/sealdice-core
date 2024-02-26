package dice

import (
	"encoding/json"
	"fmt"
	"strings"

	"sealdice-core/dice/model"
)

type SenderChannel struct {
	Nickname string `json:"nickname"`
	UserID   string `json:"tiny_id"`
}

// {"channel_id":"3574366","guild_id":"51541481646552899","message_id":"BAC3HLRYvXdDAAAAAAA2il4AAAAAAAAAEQ==","notice_type":"guild_channel_recall","operator_id":"1441152187
// 31218202","post_type":"notice","self_id":2589922907,"self_tiny_id":"144115218748146488","time":1650386683,"user_id":144115218731218202}

type MessageQQChannel struct {
	MessageType string `json:"message_type"` // guild
	SubType     string `json:"sub_type"`     // 子类型，channel
	GuildID     string `json:"guild_id"`     // 频道ID
	ChannelID   string `json:"channel_id"`   // 子频道ID
	// UserId      int    `json:"user_id"` // 这个不稳定 有时候是int64
	MessageID string `json:"message_id"` // QQ信息此类型为int64，频道中为string
	Message   string `json:"message"`    // 消息内容
	Time      int64  `json:"time"`       // 发送时间 文档上没有实际有
	PostType  string `json:"post_type"`  // 目前只见到message
	// self_id 2589922907 QQ号
	// self_tiny_id 个人频道ID
	SelfTinyID string `json:"self_tiny_id"` // 文档上没有，个人频道ID
	NoticeType string `json:"notice_type"`  // 文档上没有，但实际有。撤回有信息

	Sender *SenderChannel `json:"sender"` // 发送者
	Echo   int            `json:"echo"`
}

func (msgQQ *MessageQQChannel) toStdMessage() *Message {
	msg := new(Message)
	msg.Time = msgQQ.Time
	msg.MessageType = "group"
	msg.Message = msgQQ.Message
	msg.RawID = msgQQ.MessageID
	msg.Platform = "QQ-CH"
	msg.TmpUID = FormatDiceIDQQCh(msgQQ.SelfTinyID)

	msg.GroupID = FormatDiceIDQQChGroup(msgQQ.GuildID, msgQQ.ChannelID)
	if msgQQ.Sender != nil {
		msg.Sender.Nickname = msgQQ.Sender.Nickname
		msg.Sender.UserID = FormatDiceIDQQCh(msgQQ.Sender.UserID)
	}
	return msg
}

func (pa *PlatformAdapterGocq) QQChannelTrySolve(message string) {
	msgQQ := new(MessageQQChannel)
	err := json.Unmarshal([]byte(message), msgQQ)

	if err == nil {
		// fmt.Println("DDD", message)
		ep := pa.EndPoint
		session := pa.Session

		msg := msgQQ.toStdMessage()
		ctx := &MsgContext{ /* MessageType: msg.MessageType, EndPoint: ep, Session: pa.Session, */ Dice: pa.Session.Parent}

		// 消息撤回
		if msgQQ.PostType == "notice" && msgQQ.NoticeType == "guild_channel_recall" {
			group := session.ServiceAtNew[msg.GroupID]
			if group != nil {
				if group.LogOn {
					_ = model.LogMarkDeleteByMsgID(ctx.Dice.DBLogs, group.GroupID, group.LogCurName, msgQQ.MessageID)
				}
			}
			return
		}

		if msgQQ.PostType == "notice" && msgQQ.NoticeType == "message_reactions_updated" {
			// 一大段的表情设置信息，我也搞不懂是什么
			return
		}

		// 处理命令
		if msgQQ.MessageType == "guild" || msgQQ.MessageType == "private" {
			if msg.Sender.UserID == ep.UserID {
				return
			}

			// fmt.Println("Recieved message1 " + message)
			session.Execute(ep, msg, false)
		} else {
			fmt.Println("CH Recieved message " + message)
		}
	}
	// pa.SendToChannelGroup(ctx, msg.GroupId, msg.Message+"asdasd", "")
}

func (pa *PlatformAdapterGocq) SendToChannelGroup(ctx *MsgContext, userID string, text string, flag string) {
	rawID, _ := pa.mustExtractChannelID(userID)
	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, &Message{
					MessageType: "group",
					Message:     text,
					GroupID:     userID,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
				}, flag)
			})
		}
	}

	lst := strings.Split(rawID, "-")
	if len(lst) < 2 {
		return
	}

	type GroupMessageParams struct {
		// MessageType string `json:"message_type"`
		Message   string `json:"message"`
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
	}

	text = strings.ReplaceAll(text, ".net", "_net")
	text = strings.ReplaceAll(text, ".com", "_com")
	text = strings.ReplaceAll(text, "www.", "www_")
	text = strings.ReplaceAll(text, "log.sealdice", "log_sealdice")
	text = strings.ReplaceAll(text, "dice.weizaima", "dice_weizaima")
	text = strings.ReplaceAll(text, "log.weizaima", "log_weizaima")
	text = strings.ReplaceAll(text, "://", "_//")
	text = textAssetsConvert(text)
	texts := textSplit(text)
	for _, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_guild_channel_msg",
			Params: GroupMessageParams{
				//MessageType: "private",
				GuildID:   lst[0],
				ChannelID: lst[1],
				Message:   subText,
			},
		})
		doSleepQQ(ctx)
		socketSendText(pa.Socket, string(a))
	}
}
