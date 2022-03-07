package dice

import (
	"encoding/json"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"time"
)

// 2022/02/03 11:47:42 Recieved message {"font":0,"message":"test","message_id":-487913662,"message_type":"private","post_type":"message","raw_message":"test","self_id":1001,"sender":{"age":0,"nickname":"鏈ㄨ惤","sex":"unknown","user_id":1002},"sub_type":"friend","target_id":1001,"time":1643860062,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"qqq","message_id":884917177,"message_seq":1434,"message_type":"group","post_type":"message","raw_message":"qqq","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1643863961,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"[CQ:at,qq=1001]   .r test","message_id":888971055,"message_seq":1669,"message_type":"group","post_type":"message","raw_message":"[CQ:at,qq=1001]   .r test","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1644127751,"user_id":1002}

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

func ReplyPerson(ctx *MsgContext, userId int64, text string) {
	replyPersonRaw(ctx, userId, text, "")
}

func replyPersonRaw(ctx *MsgContext, userId int64, text string, flag string) {
	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.OnMessageSend(ctx, "private", userId, text, flag)
		}
	}
	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	type GroupMessageParams struct {
		MessageType string `json:"message_type"`
		UserId      int64  `json:"user_id"`
		Message     string `json:"message"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		Action: "send_msg",
		Params: GroupMessageParams{
			MessageType: "private",
			UserId:      userId,
			Message:     text,
		},
	})

	socketSendText(ctx.Socket, string(a))
}

func GetGroupInfo(socket *gowebsocket.Socket, groupId int64) {
	type GroupMessageParams struct {
		GroupId int64 `json:"group_id"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
		Echo   int64              `json:"echo"`
	}{
		"get_group_info",
		GroupMessageParams{
			groupId,
		},
		-2,
	})

	socketSendText(socket, string(a))
	//socket.SendText(string(a))
}

func SetGroupAddRequest(socket *gowebsocket.Socket, flag string, subType string, approve bool, reason string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		SubType string `json:"sub_type"`
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_group_add_request",
		DetailParams{
			Flag:    flag,
			SubType: subType,
			Approve: approve,
			Reason:  reason,
		},
	})

	socketSendText(socket, string(a))
	//socket.SendText(string(a))
}

func QuitGroup(ctx *MsgContext, groupId int64) {
	type GroupMessageParams struct {
		GroupId int64 `json:"group_id"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		"set_group_leave",
		GroupMessageParams{
			groupId,
		},
	})

	socketSendText(ctx.Socket, string(a))
	//s.Socket.SendText(string(a))
}

func ReplyGroup(ctx *MsgContext, groupId int64, text string) {
	replyGroupRaw(ctx, groupId, text, "")
}

func replyGroupRaw(ctx *MsgContext, groupId int64, text string, flag string) {
	if ctx.Session.ServiceAt[groupId] != nil {
		for _, i := range ctx.Session.ServiceAt[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.OnMessageSend(ctx, "group", groupId, text, flag)
			}
		}
	}

	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
		Message string `json:"message"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		"send_group_msg",
		GroupMessageParams{
			groupId,
			text, // "golang client test",
		},
	})
	socketSendText(ctx.Socket, string(a))
	//ctx.Session.Socket.SendText(string(a))
}

func ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		replyGroupRaw(ctx, msg.GroupId, text, flag)
	} else {
		replyPersonRaw(ctx, msg.Sender.UserId, text, flag)
	}
}

func ReplyToSender(ctx *MsgContext, msg *Message, text string) {
	ReplyToSenderRaw(ctx, msg, text, "")
}

func (c *ConnectInfoItem) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo   int64  `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   -1,
	})

	//if s.Socket != nil {
	socketSendText(c.Socket, string(a))
	//s.Socket.SendText(string(a))
	//}
}
