package req

import "sealdice-core/model/common/request"

type GroupPageRequest struct {
	request.PageInfo
	Filter GroupFilter `json:"filter" required:"false"`
}

type GroupFilter struct {
	// 筛选平台
	Platform []string `json:"platforms" form:"platforms"  required:"false"`
	// 按最后时间降序
	OrderByLastTime bool `json:"orderByLastTime" form:"orderByLastTime"  required:"false"`
	// 查询 N天内 未使用
	UnusedDays int `json:"queryUnusedDays" form:"queryUnusedDays"  required:"false"`
	// 是否正在记录日志
	IsLogging bool `json:"isLogging" form:"isLogging"  required:"false"`
}
