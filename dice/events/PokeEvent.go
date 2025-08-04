package events

type PokeEvent struct {
	GroupID   string `jsbind:"groupId"   json:"group_id"`
	SenderID  string `jsbind:"senderId"  json:"sender_id"`
	TargetID  string `jsbind:"targetId"  json:"target_id"`
	IsPrivate bool   `jsbind:"isPrivate" json:"is_private"`
}
