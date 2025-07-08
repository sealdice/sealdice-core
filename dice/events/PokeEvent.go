package events

type PokeEvent struct {
	GroupID   string `json:"group_id" jsbind:"groupId"`
	SenderID  string `json:"sender_id" jsbind:"senderId"`
	TargetID  string `json:"target_id" jsbind:"targetId"`
	IsPrivate bool   `json:"is_private" jsbind:"isPrivate"`
}
