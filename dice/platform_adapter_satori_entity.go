package dice

import "encoding/json"

type SatoriChannelType int

const (
	SatoriTextChannel     SatoriChannelType = iota // 文本频道
	SatoriDirectChannel                            // 私聊频道
	SatoriCategoryChannel                          // 分类频道
	SatoriVoiceChannel                             // 语音频道
)

type SatoriChannel struct {
	ID       string            `json:"id"`
	Type     SatoriChannelType `json:"type"`
	Name     string            `json:"name"`
	ParentID string            `json:"parent_id"`
}

type SatoriGuild struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type SatoriGuildMember struct {
	User     *SatoriUser `json:"user"`
	Nick     string      `json:"nick"`
	Avatar   string      `json:"avatar"`
	JoinedAt int64       `json:"joined_at"`
}

type SatoriGuildRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SatoriUser struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Nick   string `json:"nick"`
	Avatar string `json:"avatar"`
	IsBot  bool   `json:"is_bot"`
}

type SatoriStatus int

const (
	SatoriOffline SatoriStatus = iota
	SatoriOnline
	SatoriConnect
	SatoriDisconnect
	SatoriReconnect
)

type SatoriLogin struct {
	User     *SatoriUser  `json:"user"`
	SelfID   string       `json:"self_id"`
	Platform string       `json:"platform"`
	Status   SatoriStatus `json:"status"`
}

type SatoriMessage struct {
	ID        string             `json:"id"`
	Content   string             `json:"content"`
	Channel   *SatoriChannel     `json:"channel"`
	Guild     *SatoriGuild       `json:"guild"`
	Member    *SatoriGuildMember `json:"member"`
	User      *SatoriUser        `json:"user"`
	CreatedAt int64              `json:"created_at"`
	UpdatedAt int64              `json:"updated_at"`
}

type SatoriArgv struct {
	Name      string        `json:"name"`
	Arguments []interface{} `json:"arguments"`
	Options   interface{}   `json:"options"`
}

type SatoriButton struct {
	ID string `json:"id"`
}

type SatoriOpCode int

const (
	SatoriOpEvent SatoriOpCode = iota
	SatoriOpPing
	SatoriOpPong
	SatoriOpIdentify
	SatoriOpReady
)

type SatoriPayload[T SatoriIdentify | SatoriReady | SatoriEvent | any] struct {
	Op   SatoriOpCode `json:"op"`
	Body *T           `json:"body"`
}

type SatoriIdentify struct {
	Token    string `json:"token"`
	Sequence int64  `json:"sequence"`
}

type SatoriReady struct {
	Logins []*SatoriLogin `json:"logins"`
}

type SatoriEvent struct {
	ID        json.Number        `json:"id"`
	Type      string             `json:"type"`
	Platform  string             `json:"platform"`
	SelfID    string             `json:"self_id"`
	Timestamp int64              `json:"timestamp"`
	Argv      *SatoriArgv        `json:"argv"`
	Button    *SatoriButton      `json:"button"`
	Channel   *SatoriChannel     `json:"channel"`
	Guild     *SatoriGuild       `json:"guild"`
	Login     *SatoriLogin       `json:"login"`
	Member    *SatoriGuildMember `json:"member"`
	Message   *SatoriMessage     `json:"message"`
	Operator  *SatoriUser        `json:"operator"`
	Role      *SatoriGuildRole   `json:"role"`
	User      *SatoriUser        `json:"user"`
}
