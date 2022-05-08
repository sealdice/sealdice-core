package dice

import (
	"encoding/json"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type oneBotCommand struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   int64       `json:"echo"`
}

type QQUidType int

const (
	QQUidPerson        QQUidType = 1
	QQUidGroup         QQUidType = 2
	QQUidChannelPerson QQUidType = 3
	QQUidChannelGroup  QQUidType = 4
)

func (pa *PlatformAdapterQQOnebot) mustExtractId(id string) (int64, QQUidType) {
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

func (pa *PlatformAdapterQQOnebot) mustExtractChannelId(id string) (string, QQUidType) {
	if strings.HasPrefix(id, "QQ-CH:") {
		return id[len("QQ-CH:"):], QQUidChannelPerson
	}
	if strings.HasPrefix(id, "QQ-CH-Group:") {
		return id[len("QQ-CH-Group:"):], QQUidChannelGroup
	}
	return "", 0
}

// GetGroupInfoAsync 异步获取群聊信息
func (pa *PlatformAdapterQQOnebot) GetGroupInfoAsync(groupId string) {
	type GroupMessageParams struct {
		GroupId int64 `json:"group_id"`
	}
	realGroupId, type_ := pa.mustExtractId(groupId)
	if type_ != QQUidGroup {
		return
	}

	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupId,
		},
		-2,
	})

	socketSendText(pa.Socket, string(a))
}

func socketSendText(socket *gowebsocket.Socket, s string) {
	defer func() {
		if r := recover(); r != nil {
			//core.GetLogger().Error(r)
		}
	}()

	if socket != nil {
		socket.SendText(s)
	}
}

// 不知道为什么，使用这个时候发不出话
func socketSendBinary(socket *gowebsocket.Socket, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			//core.GetLogger().Error(r)
		}
	}()

	if socket != nil {
		socket.SendBinary(data)
	}
}

func doSleepQQ(ctx *MsgContext) {
	if ctx.Dice != nil {
		d := ctx.Dice
		offset := d.MessageDelayRangeEnd - d.MessageDelayRangeStart
		time.Sleep(time.Duration((d.MessageDelayRangeStart + rand.Float64()*offset) * float64(time.Second)))
	} else {
		time.Sleep(time.Duration((0.4 + rand.Float64()/2) * float64(time.Second)))
	}
}

func (pa *PlatformAdapterQQOnebot) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	rawId, type_ := pa.mustExtractId(userId)

	if type_ != QQUidPerson {
		return
	}

	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.OnMessageSend(ctx, "private", userId, text, flag)
		}
	}

	type GroupMessageParams struct {
		MessageType string `json:"message_type"`
		UserId      int64  `json:"user_id"`
		Message     string `json:"message"`
	}

	texts := textSplit(text)
	for _, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_msg",
			Params: GroupMessageParams{
				MessageType: "private",
				UserId:      rawId,
				Message:     subText,
			},
		})
		doSleepQQ(ctx)
		socketSendText(pa.Socket, string(a))
	}
}

func (pa *PlatformAdapterQQOnebot) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	if groupId == "" {
		return
	}
	rawId, type_ := pa.mustExtractId(groupId)
	if type_ == 0 {
		pa.SendToChannelGroup(ctx, groupId, text, flag)
		return
	}

	if ctx.Session.ServiceAtNew[groupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.OnMessageSend(ctx, "group", groupId, text, flag)
			}
		}
	}

	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
		Message string `json:"message"`
	}

	texts := textSplit(text)

	for _, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_group_msg",
			Params: GroupMessageParams{
				rawId,
				subText, // "golang client test",
			},
		})

		doSleepQQ(ctx)
		socketSendText(pa.Socket, string(a))
	}
}

// SetGroupAddRequest 同意加群
func (pa *PlatformAdapterQQOnebot) SetGroupAddRequest(flag string, subType string, approve bool, reason string) {
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

// GetGroupMemberInfo 获取群成员信息
func (pa *PlatformAdapterQQOnebot) GetGroupMemberInfo(GroupId int64, UserId int64) {
	type DetailParams struct {
		GroupId int64 `json:"group_id"`
		UserId  int64 `json:"user_id"`
		NoCache bool  `json:"no_cache"`
	}

	a, _ := json.Marshal(oneBotCommand{
		"get_group_member_info",
		DetailParams{
			GroupId: GroupId,
			UserId:  UserId,
		},
		-3,
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterQQOnebot) SetGroupCardName(groupId string, userId string, name string) {
	groupIdRaw, type_ := pa.mustExtractId(groupId)
	if type_ != QQUidGroup {
		return
	}
	userIdRaw, type_ := pa.mustExtractId(userId)
	if type_ != QQUidPerson {
		return
	}

	pa.SetGroupCardNameBase(groupIdRaw, userIdRaw, name)
}

// SetGroupCardNameBase 设置群名片
func (pa *PlatformAdapterQQOnebot) SetGroupCardNameBase(GroupId int64, UserId int64, Card string) {
	type DetailParams struct {
		GroupId int64  `json:"group_id"`
		UserId  int64  `json:"user_id"`
		Card    string `json:"card"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_group_card",
		DetailParams{
			GroupId: GroupId,
			UserId:  UserId,
			Card:    Card,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterQQOnebot) SetFriendAddRequest(flag string, approve bool, remark string, reaseon string) {
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

func (pa *PlatformAdapterQQOnebot) QuitGroup(ctx *MsgContext, id string) {
	groupId, type_ := pa.mustExtractId(id)
	if type_ != QQUidGroup {
		return
	}
	type GroupMessageParams struct {
		GroupId int64 `json:"group_id"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "set_group_leave",
		Params: GroupMessageParams{
			groupId,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterQQOnebot) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo   int64  `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   -1,
	})

	//if s.Socket != nil {
	socketSendText(pa.Socket, string(a))
	//s.Socket.SendText(string(a))
	//}
}

func textSplit(input string) []string {
	maxLen := 5000 // 以utf-8计算，1666个汉字
	var splits []string

	var l, r int
	for l, r = 0, maxLen; r < len(input); l, r = r, r+maxLen {
		for !utf8.RuneStart(input[r]) {
			r--
		}
		splits = append(splits, input[l:r])
	}
	splits = append(splits, input[l:])
	return splits
}
