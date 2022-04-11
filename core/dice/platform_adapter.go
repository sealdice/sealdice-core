package dice

type PlatformAdapter interface {
	GetGroupInfoAsync(groupId string)
	Serve() int
	DoRelogin() bool
	SetEnable(enable bool)
	QuitGroup(ctx *MsgContext, id string)

	SendToPerson(ctx *MsgContext, uid string, text string, flag string)
	SendToGroup(ctx *MsgContext, uid string, text string, flag string)
}
