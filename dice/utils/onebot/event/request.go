package event

type FriendRequest struct {
	UserId  int64  `json:"user_id"`
	Comment string `json:"comment"`
	Flag    string `json:"flag"`
}

func (fr FriendRequest) Type() string {
	return "request:friend"
}

func (fr *FriendRequest) Reply(replyer Replyer, approve bool, remark string) error {
	if replyer == nil {
		return ErrNoAvailable
	}
	data := struct {
		Approve bool   `json:"approve"`
		Remark  string `json:"remark"`
	}{Approve: approve, Remark: remark}
	return replyer.Reply(data)
}

type GroupRequest struct {
	SubType string `json:"sub_type"` // add invite
	GroupId int64  `json:"group_id"`
	UserId  int64  `json:"user_id"`
	Comment string `json:"comment"`
	Flag    string `json:"flag"`
}

func (gr GroupRequest) Type() string {
	return "request:group"
}

func (gr *GroupRequest) Reply(replyer Replyer, approve bool, reason string) error {
	if replyer == nil {
		return ErrNoAvailable
	}
	data := struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}{Approve: approve, Reason: reason}
	return replyer.Reply(data)
}
