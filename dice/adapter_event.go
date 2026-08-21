package dice

import (
	"time"
)

// 适配器事件名常量。命名规约：领域.动作，全小写，点分。
// 新增事件名时同步更新 docs/superpowers/specs/2026-08-21-adapter-event-bus-design.md 的映射表。
const (
	EventNamePoke              = "poke"                // 戳一戳
	EventNameGroupJoined       = "group.joined"        // 骰子自身加入群
	EventNameGroupMemberJoined = "group.member_joined" // 其他群成员加入群
	EventNameGroupLeave        = "group.leave"         // 群成员离开/被踢
	EventNameGroupMuted        = "group.muted"         // 群禁言（预留，暂无适配器发射）
	EventNameFriendJoined      = "friend.joined"       // 成为好友
	EventNameGuildJoined       = "guild.joined"        // 加入频道（KOOK 等）
	EventNameFriendRequest     = "friend.request"      // 好友申请（仅通知）
	EventNameGroupRequest      = "group.request"       // 加群申请/邀请（仅通知）
)

// AdapterEvent 适配器事件的统一封装。
// 命名刻意区别于未来可能的"插件事件机制"，也区别于库类型 goeventbus.Event。
// 请求类事件（friend.request / group.request）仅通知，不携带回复通道。
type AdapterEvent struct {
	ID         string         `jsbind:"id"          json:"id"`
	Name       string         `jsbind:"name"        json:"name"`     // 事件名，见 EventNameXxx 常量
	Platform   string         `jsbind:"platform"    json:"platform"` // 如 "QQ"、"DISCORD"
	EndPointID string         `jsbind:"endPointId"  json:"endpoint_id"`
	GroupID    string         `jsbind:"groupId"     json:"group_id"`  // UNI-ID 格式，如 QQ:123456
	UserID     string         `jsbind:"userId"      json:"user_id"`   // 事件主体（如被禁言者）
	SenderID   string         `jsbind:"senderId"    json:"sender_id"` // 操作发起者
	Raw        map[string]any `jsbind:"raw"         json:"raw"`       // 适配器原样透传的附加数据
	Time       time.Time      `jsbind:"time"        json:"time"`

	// Ctx 携带构造事件时的消息上下文，供 JS/Go 订阅者回复等操作使用。不序列化。
	Ctx *MsgContext `json:"-" yaml:"-"`
	// Detail 携带旧类型化载荷（*events.PokeEvent 等），仅供兼容订阅器还原旧逻辑。不序列化，不供 JS 使用。
	Detail any `json:"-" yaml:"-"`
}
