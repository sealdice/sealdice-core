package events

// GroupLeaveEvent all ID must be UNI-ID format e.g., QQ:1234567890
type GroupLeaveEvent struct {
	GroupID    string `json:"group_id" jsbind:"groupId"`       // The ID of the group from which the member was kicked.
	UserID     string `json:"user_id" jsbind:"userId"`         // The ID of the user who was kicked from the group.
	OperatorID string `json:"operator_id" jsbind:"operatorId"` // The ID of the user who performed the kick operation.
}
