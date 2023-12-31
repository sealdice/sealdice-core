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

	// DeleteFriend 删除好友，目前只有 QQ 平台下的 gocq 和 walleq 实现有这个方法
	// DeleteFriend(ctx *MsgContext, id string)

	// EditMessage replace the content of the message with msgID with message.
	// Context is retrieved from ctx.
	EditMessage(ctx *MsgContext, msgID, message string)
	// RecallMessage recalls the message with msgID. Context is retrieved from ctx.
	RecallMessage(ctx *MsgContext, msgID string)
}

// 实现检查
var (
	_ PlatformAdapter = (*PlatformAdapterGocq)(nil)
	_ PlatformAdapter = (*PlatformAdapterDingTalk)(nil)
	_ PlatformAdapter = (*PlatformAdapterDodo)(nil)
	_ PlatformAdapter = (*PlatformAdapterHTTP)(nil)
	_ PlatformAdapter = (*PlatformAdapterKook)(nil)
	_ PlatformAdapter = (*PlatformAdapterMinecraft)(nil)
	_ PlatformAdapter = (*PlatformAdapterOfficialQQ)(nil)
	_ PlatformAdapter = (*PlatformAdapterRed)(nil)
	_ PlatformAdapter = (*PlatformAdapterSealChat)(nil)
	_ PlatformAdapter = (*PlatformAdapterSlack)(nil)
	_ PlatformAdapter = (*PlatformAdapterTelegram)(nil)
	_ PlatformAdapter = (*PlatformAdapterWalleQ)(nil)
)
