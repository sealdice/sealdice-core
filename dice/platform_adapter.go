package dice

type PlatformAdapter interface {
	Serve() int
	DoRelogin() bool
	SetEnable(enable bool)
	QuitGroup(ctx *MsgContext, ID string)

	SendToPerson(ctx *MsgContext, userID string, text string, flag string)
	SendToGroup(ctx *MsgContext, groupID string, text string, flag string)
	SetGroupCardName(ctx *MsgContext, name string)

	SendFileToPerson(ctx *MsgContext, userID string, path string, flag string)
	SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string)

	MemberBan(groupID string, userID string, duration int64)
	MemberKick(groupID string, userID string)

	GetGroupInfoAsync(groupID string)
}
