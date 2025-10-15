package event

type Replyer interface {
	Reply(data any) error
}
