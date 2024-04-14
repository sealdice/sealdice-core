package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"sealdice-core/dice/model"
	"time"

	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

func (pa *PlatformOnebot12) mustExtractID(id string) (int64, QQUidType) {
	if strings.HasPrefix(id, "QQ:") {
		num, _ := strconv.ParseInt(id[len("QQ:"):], 10, 64)
		return num, QQUidPerson
	}
	if strings.HasPrefix(id, "QQ-Group:") {
		num, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
		return num, QQUidGroup
	}
	if strings.HasPrefix(id, "PG-QQ:") {
		num, _ := strconv.ParseInt(id[len("PG-QQ:"):], 10, 64)
		return num, QQUidPerson
	}
	return 0, 0
}

func (pa *PlatformOnebot12) mustExtractChannelID(id string) (string, QQUidType) {
	if strings.HasPrefix(id, "QQ-CH:") {
		return id[len("QQ-CH:"):], QQUidChannelPerson
	}
	if strings.HasPrefix(id, "QQ-CH-Group:") {
		return id[len("QQ-CH-Group:"):], QQUidChannelGroup
	}
	return "", 0
}

// GetGroupInfoAsync 异步获取群聊信息
func (pa *PlatformOnebot12) GetGroupInfoAsync(groupID string) {
	type GroupMessageParams struct {
		GroupID int64 `json:"group_id"`
	}
	realGroupID, idType := pa.mustExtractID(groupID)
	if idType != QQUidGroup {
		return
	}

	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupID,
		},
		-2,
	})

	socketSendText(pa.Socket, string(a))
}

type OnebotGroupInfo2 struct {
	GroupID         int64  `json:"group_id"`          // 群号
	GroupName       string `json:"group_name"`        // 群名称
	GroupMemo       string `json:"group_memo"`        // 群备注
	GroupCreateTime uint32 `json:"group_create_time"` // 群创建时间
	GroupLevel      uint32 `json:"group_level"`       // 群等级
	MemberCount     int32  `json:"member_count"`      // 成员数
	MaxMemberCount  int32  `json:"max_member_count"`  // 最大成员数（群容量）
}

// GetGroupInfo 获取群聊信息
func (pa *PlatformOnebot12) GetGroupInfo(groupID string) *OnebotGroupInfo {
	type GroupMessageParams struct {
		GroupID int64 `json:"group_id"`
	}
	realGroupID, idType := pa.mustExtractID(groupID)
	if idType != QQUidGroup {
		return nil
	}

	echo := pa.getCustomEcho()
	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupID,
		},
		echo,
	})

	data := &OnebotGroupInfo{}
	err := pa.waitEcho2(echo, data, func(emi *echoMapInfo) {
		emi.echoOverwrite = -2 // 强制覆盖为获取群信息，与之前兼容
		socketSendText(pa.Socket, string(a))
	})
	if err == nil {
		return data
	}
	return nil
}

func (pa *PlatformOnebot12) getCustomEcho() string {
	if pa.customEcho > -10 {
		pa.customEcho = -10
	}
	pa.customEcho--
	return strconv.FormatInt(pa.customEcho, 10)
}

func (pa *PlatformOnebot12) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
	rawID, idType := pa.mustExtractID(userID)

	if idType != QQUidPerson {
		return
	}

	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, &Message{
					Message:     text,
					MessageType: "private",
					Platform:    pa.EndPoint.Platform,
					Sender: SenderBase{
						Nickname: pa.EndPoint.Nickname,
						UserID:   pa.EndPoint.UserID,
					},
				}, flag)
			})
		}
	}

	type GroupMessageParams struct {
		MessageType string `json:"message_type"`
		UserID      int64  `json:"user_id"`
		Message     string `json:"message"`
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)
	for _, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_msg",
			Params: GroupMessageParams{
				MessageType: "private",
				UserID:      rawID,
				Message:     subText,
			},
		})
		doSleepQQ(ctx)
		socketSendText(pa.Socket, string(a))
	}
}

func (pa *PlatformOnebot12) SendToChannelGroup(ctx *MsgContext, userID string, text string, flag string) {
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

func (pa *PlatformOnebot12) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	if groupID == "" {
		return
	}
	rawID, idType := pa.mustExtractID(groupID)
	if idType == 0 {
		pa.SendToChannelGroup(ctx, groupID, text, flag)
		return
	}

	if ctx.Session.ServiceAtNew[groupID] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupID].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, &Message{
						Message:     text,
						MessageType: "group",
						Platform:    pa.EndPoint.Platform,
						GroupID:     groupID,
						Sender: SenderBase{
							Nickname: pa.EndPoint.Nickname,
							UserID:   pa.EndPoint.UserID,
						},
					}, flag)
				})
			}
		}
	}

	type GroupMessageParams struct {
		GroupID int64  `json:"group_id"`
		Message string `json:"message"`
	}

	type GroupArrMessageParams struct {
		GroupID int64         `json:"group_id"`
		Message []interface{} `json:"message"` // 消息内容，原则上是OneBotV11MsgItem但是实际很杂说不清
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)

	for index, subText := range texts {
		var a []byte
		if pa.useArrayMessage {
			a, _ = json.Marshal(oneBotCommand{
				Action: "send_group_msg",
				Params: GroupArrMessageParams{
					GroupID: rawID,
					Message: OneBot11CqMessageToArrayMessage(subText),
				},
			})
		} else {
			a, _ = json.Marshal(oneBotCommand{
				Action: "send_group_msg",
				Params: GroupMessageParams{
					rawID,
					subText, // "golang client test",
				},
			})
		}

		if len(texts) > 1 && index != 0 {
			doSleepQQ(ctx)
		}
		socketSendText(pa.Socket, string(a))
	}
}

func (pa *PlatformOnebot12) SendFileToPerson(ctx *MsgContext, userID string, path string, _ string) {
	rawID, idType := pa.mustExtractID(userID)
	if idType != QQUidPerson {
		return
	}

	dice := pa.Session.Parent
	// 路径可以是 http/base64/本地路径，但 gocq 的文件上传只支持本地文件，所以临时下载到本地
	fileName, temp, err := dice.ExtractLocalTempFile(path)
	defer func(name string) {
		_ = os.Remove(name)
	}(temp.Name())

	if err != nil {
		dice.Logger.Errorf("尝试发送文件[path=%s]出错: %s", path, err.Error())
		return
	}

	type uploadPrivateFileParams struct {
		UserID int64  `json:"user_id"`
		File   string `json:"file"`
		Name   string `json:"name"`
	}
	a, _ := json.Marshal(oneBotCommand{
		Action: "upload_private_file",
		Params: uploadPrivateFileParams{
			UserID: rawID,
			File:   temp.Name(),
			Name:   fileName,
		},
	})
	doSleepQQ(ctx)
	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformOnebot12) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	if groupID == "" {
		return
	}
	rawID, idType := pa.mustExtractID(groupID)
	if idType == 0 {
		// qq频道尚不支持文件发送，降级
		pa.SendToChannelGroup(ctx, groupID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
		return
	}

	dice := pa.Session.Parent
	// 路径可以是 http/base64/本地路径，但 gocq 的文件上传只支持本地文件，所以临时下载到本地
	fileName, temp, err := dice.ExtractLocalTempFile(path)
	defer func(name string) {
		_ = os.Remove(name)
	}(temp.Name())

	if err != nil {
		dice.Logger.Errorf("尝试发送文件[path=%s]出错: %s", path, err.Error())
		return
	}

	type uploadGroupFileParams struct {
		GroupID int64  `json:"group_id"`
		File    string `json:"file"`
		Name    string `json:"name"`
	}
	a, _ := json.Marshal(oneBotCommand{
		Action: "upload_group_file",
		Params: uploadGroupFileParams{
			GroupID: rawID,
			File:    temp.Name(),
			Name:    fileName,
		},
	})
	doSleepQQ(ctx)
	socketSendText(pa.Socket, string(a))
}

// SetGroupAddRequest 同意加群
func (pa *PlatformOnebot12) SetGroupAddRequest(flag string, subType string, approve bool, reason string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		SubType string `json:"sub_type"`
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "set_group_add_request",
		Params: DetailParams{
			Flag:    flag,
			SubType: subType,
			Approve: approve,
			Reason:  reason,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformOnebot12) waitEcho(echo any, beforeWait func()) *MessageQQ2 {
	// pa.echoList = append(pa.echoList, )
	ch := make(chan *MessageQQ2, 1)

	if pa.echoMap == nil {
		pa.echoMap = new(SyncMap[any, chan *MessageQQ2])
	}

	// 注: 之所以这样是因为echo是json.RawMessage
	e := lo.Must(json.Marshal(echo))
	pa.echoMap.Store(string(e), ch)

	beforeWait()
	return <-ch
}

func (pa *PlatformOnebot12) waitEcho2(echo any, value interface{}, beforeWait func(emi *echoMapInfo)) error {
	if pa.echoMap2 == nil {
		pa.echoMap2 = new(SyncMap[any, *echoMapInfo])
	}

	emi := &echoMapInfo{ch: make(chan string, 1)}
	beforeWait(emi)

	pa.echoMap2.Store(echo, emi)
	val := <-emi.ch
	if val == "" {
		return errors.New("超时")
	}
	return json.Unmarshal([]byte(val), value)
}

// GetGroupMemberInfo 获取群成员信息
func (pa *PlatformOnebot12) GetGroupMemberInfo(groupID string, userID string) *OnebotUserInfo {
	type DetailParams struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
		NoCache bool   `json:"no_cache"`
	}

	echo := pa.getCustomEcho()

	a, _ := json.Marshal(oneBotCommand{
		Action: "get_group_member_info",
		Params: DetailParams{
			GroupID: groupID,
			UserID:  userID,
			NoCache: false,
		},
		Echo: echo,
	})

	msg := pa.waitEcho(echo, func() {
		socketSendText(pa.Socket, string(a))
	})

	d := msg.Data
	if msg.Data == nil {
		return &OnebotUserInfo{}
	}

	return &OnebotUserInfo{
		Nickname: d.Nickname,
		UserID:   string(d.UserID),
		GroupID:  string(d.GroupID),
		Card:     d.Card,
	}
}

// GetStrangerInfo 获取陌生人信息
func (pa *PlatformOnebot12) GetStrangerInfo(userID string) *OnebotUserInfo {
	type DetailParams struct {
		UserID  string `json:"user_id"`
		NoCache bool   `json:"no_cache"`
	}

	echo := pa.getCustomEcho()

	a, _ := json.Marshal(oneBotCommand{
		Action: "get_stranger_info",
		Params: DetailParams{
			UserID:  userID,
			NoCache: false,
		},
		Echo: echo,
	})

	msg := pa.waitEcho(echo, func() {
		socketSendText(pa.Socket, string(a))
	})

	d := msg.Data
	if msg.Data == nil {
		return &OnebotUserInfo{}
	}

	return &OnebotUserInfo{
		Nickname: d.Nickname,
		UserID:   string(d.UserID),
	}
}

// SetGroupCardNameBase 设置群名片
func (pa *PlatformOnebot12) SetGroupCardNameBase(groupID int64, userID int64, card string) {
	type DetailParams struct {
		GroupID int64  `json:"group_id"`
		UserID  int64  `json:"user_id"`
		Card    string `json:"card"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_group_card",
		DetailParams{
			GroupID: groupID,
			UserID:  userID,
			Card:    card,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformOnebot12) SetFriendAddRequest(flag string, approve bool, remark string, reaseon string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		Remark  string `json:"remark"` // 备注名
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_friend_add_request",
		DetailParams{
			Flag:    flag,
			Approve: approve,
			Remark:  remark,
			Reason:  reaseon,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformOnebot12) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo   int64  `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   -1,
	})

	socketSendText(pa.Socket, string(a))
}
func (pa *PlatformOnebot12) QQChannelTrySolve(message string) {
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

func (pa *PlatformOnebot12) packTempCtx(msgQQ *MessageQQ2, msg *Message) *MsgContext {
	ep := pa.EndPoint
	session := pa.Session

	ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: session, Dice: session.Parent}

	switch msg.MessageType {
	case "private":
		d := pa.GetStrangerInfo(string(msgQQ.UserID)) // 先获取个人信息，避免不存在id
		msg.Sender.UserID = FormatDiceIDQQ(string(msgQQ.UserID))
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		if ctx.Player.Name == "" {
			ctx.Player.Name = d.Nickname
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, ctx.Player.Name)
	case "group":
		d := pa.GetGroupMemberInfo(string(msgQQ.GroupID), string(msgQQ.UserID)) // 先获取个人信息，避免不存在id
		msg.Sender.UserID = FormatDiceIDQQ(string(msgQQ.UserID))
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		if ctx.Player.Name == "" {
			ctx.Player.Name = d.Card
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, ctx.Player.Name)
	}

	return ctx
}
