package dice

type PlatformAdapter interface {
	GetGroupInfoAsync(groupId string)
	Serve() int
	DoRelogin() bool
	SetEnable(enable bool)
	SendTo(ctx *MsgContext, uid string, text string)
	ReplyToSender(ctx *MsgContext, msg *Message, text string)
	ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string)
	ReplyGroup(ctx *MsgContext, msg *Message, text string)
	ReplyPerson(ctx *MsgContext, msg *Message, text string)
	QuitGroup(ctx *MsgContext, id string)
}
