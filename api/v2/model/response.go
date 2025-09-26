package model

// ErrorResponse 基础错误响应结构
type ErrorResponse struct {
	ErrorMsg string `json:"errorMsg" example:"错误描述信息"`
}

// ItemResponse 单项数据响应结构
type ItemResponse[T any] struct {
	Body struct {
		Item T `json:"item"`
	} `json:"body"`
}

// PagedResponse 分页列表响应结构
type PagedResponse[T any] struct {
	Body struct {
		Items      []T `json:"items" doc:"数据列表"`
		Pagination struct {
			CurrentPage int `json:"currentPage" example:"1" doc:"当前页码"`
			PerPage     int `json:"perPage" example:"20" doc:"每页数量"`
			Total       int `json:"total" example:"100" doc:"总记录数"`
		} `json:"pagination"`
	} `json:"body"`
}

// EmptyResponse 空响应结构（用于DELETE等操作）
type EmptyResponse struct {
	Message string `json:"message" example:"操作成功" doc:"可选的成功提示"`
}
