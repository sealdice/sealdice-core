package dice

type PlatformAdapter interface {
	Serve() int
	DoRelogin() bool
	SetEnable(enable bool)
	QuitGroup(ctx *MsgContext, id string)

	SendToPerson(ctx *MsgContext, uid string, text string, flag string)
	SendToGroup(ctx *MsgContext, uid string, text string, flag string)
	SetGroupCardName(groupID string, userID string, name string)

	SendFileToPerson(ctx *MsgContext, uid string, path string, flag string)
	SendFileToGroup(ctx *MsgContext, uid string, path string, flag string)

	MemberBan(groupID string, userID string, duration int64)
	MemberKick(groupID string, userID string)

	GetGroupInfoAsync(groupID string)
}
