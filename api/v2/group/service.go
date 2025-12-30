package group

import (
	"context"

	"sealdice-core/dice"
	"sealdice-core/model/api/req"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

// 增删改查 - 不能增，不能改
type GroupService struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

//type PageInfo struct {
//	Page     int    `json:"page" form:"page"`         // 页码
//	PageSize int    `json:"pageSize" form:"pageSize"` // 每页大小
//	Keyword  string `json:"keyword" form:"keyword"`   // 关键字
//}

// GetGroupPage 分页获取群列表
func (b *GroupService) GetGroupPage(_ context.Context, ol *request.RequestWrapper[req.GroupPageRequest]) (*response.
	ItemResponse[response.HPageResult[dice.GroupInfo]], error) {
	// TODO：把这个GroupInfo拆出前端需要的部分，不全量返回
	return nil, nil
}
