package request

type RequestWrapper[T any] struct {
	Body T `json:"body"`
}

func NewRequestWrapper[T any](body T) *RequestWrapper[T] {
	return &RequestWrapper[T]{
		Body: body,
	}
}
