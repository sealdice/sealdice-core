package satori

type Channel struct {
	ID       string      `json:"id"`
	Type     ChannelType `json:"type"`
	Name     string      `json:"name"`
	ParentID string      `json:"parent_id" gorm:"null"`
}

type ChannelType int

const (
	TextChannelType ChannelType = iota
	DirectChannelType
	VoiceChannelType
	CategoryChannelType
)

type Guild struct {
	ID     string
	Name   string
	Avatar string
}

type GuildRole struct {
	ID          string
	Name        string
	Color       int
	Position    int
	Permissions int64
	Hoist       bool
	Mentionable bool
}

type User struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Nick          string `json:"nick"`
	UserID        string // Deprecated
	Username      string // Deprecated
	Nickname      string // Deprecated
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
	IsBot         bool   `json:"is_bot"`
}

type GuildMember struct {
	User     *User    `json:"user"`
	Name     string   `json:"name"` // 指用户名吗？
	Nick     string   `json:"nick"`
	Avatar   string   `json:"avatar"`
	Title    string   `json:"title"`
	Roles    []string `json:"roles"`
	JoinedAt uint64   `json:"joined_at"`
}

type Login struct {
	User     *User
	Platform string
	SelfID   string
	Status   Status
}

type Status int

const (
	StatusOffline Status = iota
	StatusOnline
	StatusConnect
	StatusDisconnect
	StatusReconnect
)

type Message struct {
	ID        string       `json:"id"`
	MessageID string       // Deprecated
	Channel   *Channel     `json:"channel"`
	Guild     *Guild       `json:"guild"`
	User      *User        `json:"user"`
	Member    *GuildMember `json:"member"`
	Content   string       `json:"content"`
	Elements  []*Element   `json:"elements"`
	Timestamp int64        `json:"timestamp"`
	Quote     *Message     `json:"quote"`
	CreatedAt int64        `json:"createdAt"`
	UpdatedAt int64        `json:"updatedAt"`
}

type Button struct {
	ID string
}

type Command struct {
	Name        string
	Description map[string]string
	Arguments   []CommandDeclaration
	Options     []CommandDeclaration
	Children    []Command
}

type CommandDeclaration struct {
	Name        string
	Description map[string]string
	Type        string
	Required    bool
}

type Argv struct {
	Name      string
	Arguments []interface{}
	Options   map[string]interface{}
}

type EventName string

const (
	EventGenresAdded          EventName = "genres-added"
	EventGenresDeleted        EventName = "genres-deleted"
	EventMessage              EventName = "message"
	EventMessageCreated       EventName = "message-created"
	EventMessageDeleted       EventName = "message-deleted"
	EventMessageUpdated       EventName = "message-updated"
	EventMessagePinned        EventName = "message-pinned"
	EventMessageUnpinned      EventName = "message-unpinned"
	EventInteractionCommand   EventName = "interaction/command"
	EventReactionAdded        EventName = "reaction-added"
	EventReactionDeleted      EventName = "reaction-deleted"
	EventReactionDeletedOne   EventName = "reaction-deleted/one"
	EventReactionDeletedAll   EventName = "reaction-deleted/all"
	EventReactionDeletedEmoji EventName = "reaction-deleted/emoji"
	EventSend                 EventName = "send"
	EventFriendRequest        EventName = "friend-request"
	EventGuildRequest         EventName = "guild-request"
	EventGuildMemberRequest   EventName = "guild-member-request"
)

type Event struct {
	ID        any          `json:"id"`
	Type      EventName    `json:"type"`
	SelfID    string       `json:"selfID"`
	Platform  string       `json:"platform"`
	Timestamp uint64       `json:"timestamp"`
	Argv      *Argv        `json:"argv"`
	Channel   *Channel     `json:"channel"`
	Guild     *Guild       `json:"guild"`
	Login     *Login       `json:"login"`
	Member    *GuildMember `json:"member"`
	Message   *Message     `json:"message"`
	Operator  *User        `json:"operator"`
	Role      *GuildRole   `json:"role"`
	User      *User        `json:"user"`
	Button    *Button      `json:"button"`
}

type GatewayPayloadStructure struct {
	Op   Opcode      `json:"op"`
	Body interface{} `json:"body"`
}

type Opcode int

const (
	OpEvent Opcode = iota
	OpPing
	OpPong
	OpIdentify
	OpReady
)

type GatewayBody struct {
	Event    Event
	Ping     struct{}
	Pong     struct{}
	Identify struct {
		Token    string
		Sequence int
	}
	Ready struct {
		Logins []Login
	}
}

type WebSocket struct {
	Connecting int
	Open       int
	Closing    int
	Closed     int
	ReadyState ReadyState
}

type ReadyState int

const (
	WebSocketConnecting ReadyState = iota
	WebSocketOpen
	WebSocketClosing
	WebSocketClosed
)

type ScApiMsgPayload struct {
	Api  string `json:"api"`
	Echo string `json:"echo"`
	Data any    `json:"data"`
}
