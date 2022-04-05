package dice

type PlatformAdapter interface {
	GetGroupInfoAsync(groupId string)
	ReplyToSender(ctx *MsgContext, msg *Message, text string)
	ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string)
	Serve() int
	DoRelogin() bool
	SetEnable(enable bool)
	QuitGroup(ctx *MsgContext, id string)
	ReplyGroup(ctx *MsgContext, msg *Message, text string)
	ReplyPerson(ctx *MsgContext, msg *Message, text string)
	SendTo(ctx *MsgContext, uid string, text string)
}
