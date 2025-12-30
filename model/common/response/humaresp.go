package response

// MessageResponse 通用消息响应结构
type MessageResponse struct {
	Status int `json:"-"`
	Body   struct {
		Message string `json:"message"`
	} `json:"body"`
}

type ItemResponse[T any] struct {
	Status int     `json:"-"`
	Body   Body[T] `json:"body"`
}

type Body[T any] struct {
	Item T `json:"item" doc:"响应数据项"`
}

func NewItemResponse[T any](item T) *ItemResponse[T] {
	res := Body[T]{}
	res.Item = item
	return &ItemResponse[T]{
		Body: res,
	}
}
