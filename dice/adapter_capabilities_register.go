package dice

// 集中声明各平台（不含 milky/pureonebot/gocq —— 见各自文件）的事件能力。
// 键优先匹配 ProtocolType，为空时 SendRaw 回退到平台小写名。
func init() {
	// telegram：ProtocolType 为空，回退键 "tg"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "tg",
		Platform:     "TG",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameGroupJoined:  {Name: EventNameGroupJoined, Description: "骰子加入群"},
			EventNameFriendJoined: {Name: EventNameFriendJoined, Description: "成为好友"},
		},
	})
	// dingtalk：ProtocolType 为空，回退键 "dingtalk"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "dingtalk",
		Platform:     "DINGTALK",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameGroupJoined: {Name: EventNameGroupJoined, Description: "骰子加入群"},
		},
	})
	// discord：ProtocolType 为空，回退键 "discord"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "discord",
		Platform:     "DISCORD",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameMessageDeleted: {Name: EventNameMessageDeleted, Description: "消息删除"},
		},
	})
	// kook：ProtocolType 为空，回退键 "kook"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "kook",
		Platform:     "KOOK",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameGuildJoined:    {Name: EventNameGuildJoined, Description: "加入频道"},
			EventNameMessageDeleted: {Name: EventNameMessageDeleted, Description: "消息删除"},
		},
	})
	// walle-q：ProtocolType "walle-q"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "walle-q",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameGroupJoined: {Name: EventNameGroupJoined, Description: "骰子加入群"},
		},
	})
	// official_qq：ProtocolType "official"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "official",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameGroupMemberJoined: {Name: EventNameGroupMemberJoined, Description: "群成员加入"},
			EventNameFriendJoined:      {Name: EventNameFriendJoined, Description: "成为好友"},
		},
	})
	// satori：ProtocolType "satori"
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "satori",
		Platform:     "satori",
		EmitEvents: map[string]AdapterEventSpec{
			EventNameFriendRequest: {Name: EventNameFriendRequest, Description: "好友申请（仅通知）", RequestOnly: true},
		},
	})
}
