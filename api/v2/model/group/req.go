package group

type GroupModifyRequest struct {
	Active  bool   `json:"active"`
	GroupID string `json:"groupId"`
}

type QuitGroupRequest struct {
	GroupID   string `json:"groupId"`
	Silence   bool   `json:"silence"`
	ExtraText string `json:"extraText"`
}

type BatchQuitGroupRequest struct {
	GroupIDs  []string `json:"groupIds"`
	Silence   bool     `json:"silence"`
	ExtraText string   `json:"extraText"`
}

type BatchNotifyGroupRequest struct {
	GroupIDs []string `json:"groupIds"`
	Text     string   `json:"text"`
}

