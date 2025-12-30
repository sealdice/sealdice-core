package req

import "sealdice-core/model/common/request"

type GroupPageRequest struct {
	request.PageInfo
	Filter GroupFilter `json:"filter"`
}

type GroupFilter struct {
	// 筛选平台
	Platform []string `json:"platforms" form:"platforms"`
	// 按最后时间降序
	OrderByLastTime bool `json:"orderBy" form:"orderBy"`
	// 查询 N天内 未使用
	UnusedDays int `json:"queryUnusedDays" form:"queryUnusedDays"`
	// 是否正在记录日志（有延迟）
	IsLogging bool `json:"isLogging" form:"isLogging"`
}
