package imconnection

import "sealdice-core/model/common/request"

type CreateBody struct {
	Platform string                 `json:"platform"`
	Config   map[string]interface{} `json:"config"`
}

type UpdateReq struct {
	ID   string                                         `path:"id"`
	Body request.RequestWrapper[map[string]interface{}] `json:"body"`
}

type EnableBody struct {
	Enable bool `json:"enable"`
}

type EnableReq struct {
	ID   string                             `path:"id"`
	Body request.RequestWrapper[EnableBody] `json:"body"`
}
