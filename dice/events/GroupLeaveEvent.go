package events

// GroupLeaveEvent all ID must be UNI-ID format e.g., QQ:1234567890
type GroupLeaveEvent struct {
	GroupID    string `jsbind:"groupId"    json:"group_id"`    // The ID of the group from which the member was kicked.
	UserID     string `jsbind:"userId"     json:"user_id"`     // The ID of the user who was kicked from the group.
	OperatorID string `jsbind:"operatorId" json:"operator_id"` // The ID of the user who performed the kick operation.
}
