package dice

type PlatformAdapter interface {
	Serve() int
	DoRelogin() bool
	SetEnable(enable bool)
	QuitGroup(ctx *MsgContext, id string)

	SendToPerson(ctx *MsgContext, uid string, text string, flag string)
	SendToGroup(ctx *MsgContext, uid string, text string, flag string)
	SetGroupCardName(groupId string, userId string, name string)

	MemberBan(groupId string, userId string, duration int64)
	MemberKick(groupId string, userId string)

	GetGroupInfoAsync(groupId string)
}
