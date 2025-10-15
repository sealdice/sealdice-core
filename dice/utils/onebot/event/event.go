package event

const (
	EVENT_MESSAGE = "message"
	EVENT_NOTICE  = "notice"
	EVENT_REQUEST = "request"
	EVENT_META    = "meta_event"
	Echo          = "echo"
)

type Eventer interface {
	Type() string
}

type Event struct {
	Types   []string
	Time    int64
	SelfId  int64
	RawData []byte
	Replyer Replyer
}
