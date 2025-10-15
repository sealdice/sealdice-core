package event

type Notify struct {
	SubType  string `json:"sub_type"`
	TargetId int64  `json:"target_id"`
	UserId   int64  `json:"user_id"`
	GroupId  int64  `json:"group_id"`
}

func (en Notify) Type() string {
	return "notice:notify"
}

type GroupRecall struct {
	GroupId    int64 `json:"group_id"`
	UserId     int64 `json:"user_id"`
	OperatorId int64 `json:"operator_id"`
	MessageId  int64 `json:"message_id"`
}

func (en GroupRecall) Type() string {
	return "notice:group_recall"
}

type PrivateRecall struct {
	UserId    int64 `json:"user_id"`
	MessageId int64 `json:"message_id"`
}

func (en PrivateRecall) Type() string {
	return "notice:friend_recall"
}

type GroupDecrease struct {
	SubType    string `json:"sub_type"` // leave/kick/kick_me
	GroupId    int64  `json:"group_id"`
	UserId     int64  `json:"user_id"`
	OperatorId int64  `json:"operator_id"`
}

func (en GroupDecrease) Type() string {
	return "notice:group_decrease"
}

type GroupIncrease struct {
	SubType    string `json:"sub_type"` // approve/invite
	GroupId    int64  `json:"group_id"`
	UserId     int64  `json:"user_id"`
	OperatorId int64  `json:"operator_id"`
}

func (en GroupIncrease) Type() string {
	return "notice:group_increase"
}

type Admin struct {
	SubType string `json:"sub_type"` // set/unset
	GroupId int64  `json:"group_id"`
	UserId  int64  `json:"user_id"`
}

func (en Admin) Type() string {
	return "notice:group_admin"
}

type GroupFile struct {
	GroupId int64 `json:"group_id"`
	UserId  int64 `json:"user_id"`
	File    File  `json:"file"`
}

type File struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Busid int64  `json:"busid"`
}

func (en GroupFile) Type() string {
	return "notice:group_upload"
}

type GroupBan struct {
	SubType    string `json:"sub_type"` // ban/lift_ban
	GroupId    int64  `json:"group_id"`
	UserId     int64  `json:"user_id"`
	Duration   int64  `json:"duration"` // s
	OperatorId int64  `json:"operator_id"`
}

func (en GroupBan) Type() string {
	return "notice:group_ban"
}

type PrivateAdd struct {
	UserId int64 `json:"user_id"`
}

func (en PrivateAdd) Type() string {
	return "notice:friend_add"
}
