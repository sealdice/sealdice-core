package dice

// 通知类事件的标准化发射帮助函数。
// 这些事件旧分发路径是适配器内联扩展循环（或不存在），因此只发射、不设兼容订阅器。

// EmitFriendRequest 好友申请（仅通知）。
func EmitFriendRequest(ctx *MsgContext, userID string, comment string) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:   EventNameFriendRequest,
		UserID: userID,
		Raw:    map[string]any{"comment": comment},
		Ctx:    ctx,
	})
}

// EmitGroupRequest 加群申请/邀请（仅通知）。subType 为 "add"/"invite"。
func EmitGroupRequest(ctx *MsgContext, groupID string, userID string, subType string) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:    EventNameGroupRequest,
		GroupID: groupID,
		UserID:  userID,
		Raw:     map[string]any{"subType": subType},
		Ctx:     ctx,
	})
}

// EmitFriendJoined 成为好友。
func EmitFriendJoined(ctx *MsgContext, msg *Message) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:   EventNameFriendJoined,
		UserID: msg.Sender.UserID,
		Raw:    map[string]any{"message": msg.Message},
		Ctx:    ctx,
		Detail: msg,
	})
}

// EmitGuildJoined 加入频道。
func EmitGuildJoined(ctx *MsgContext, msg *Message) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:     EventNameGuildJoined,
		GroupID:  msg.GroupID,
		SenderID: msg.Sender.UserID,
		Raw:      map[string]any{"message": msg.Message},
		Ctx:      ctx,
		Detail:   msg,
	})
}
