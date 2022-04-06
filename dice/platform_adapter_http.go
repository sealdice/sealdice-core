package dice

type HttpSimpleMessage struct {
	Uid     string `json:"uid"`
	Message string `json:"message"`
}

type PlatformAdapterHttp struct {
	RecentMessage []HttpSimpleMessage
}

func (pa *PlatformAdapterHttp) GetGroupInfoAsync(groupId string) {}

func (pa *PlatformAdapterHttp) Serve() int {
	return 0
}

func (pa *PlatformAdapterHttp) DoRelogin() bool {
	return false
}

func (pa *PlatformAdapterHttp) SetEnable(enable bool) {}

func (pa *PlatformAdapterHttp) SendTo(ctx *MsgContext, uid string, text string) {
	pa.RecentMessage = append(pa.RecentMessage, HttpSimpleMessage{uid, text})

}
func (pa *PlatformAdapterHttp) ReplyToSender(ctx *MsgContext, msg *Message, text string) {
	if msg.MessageType == "group" {
		pa.SendTo(ctx, msg.GroupId, text)
	} else {
		pa.SendTo(ctx, msg.Sender.UserId, text)
	}
}

func (pa *PlatformAdapterHttp) ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	if msg.MessageType == "group" {
		pa.SendTo(ctx, msg.GroupId, text)
	} else {
		pa.SendTo(ctx, msg.Sender.UserId, text)
	}
}

func (pa *PlatformAdapterHttp) ReplyGroup(ctx *MsgContext, msg *Message, text string) {
	pa.SendTo(ctx, msg.GroupId, text)
}

func (pa *PlatformAdapterHttp) ReplyPerson(ctx *MsgContext, msg *Message, text string) {
	pa.SendTo(ctx, msg.Sender.UserId, text)
}

func (pa *PlatformAdapterHttp) QuitGroup(ctx *MsgContext, id string) {}
