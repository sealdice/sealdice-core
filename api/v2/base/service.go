package base

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	resp2 "sealdice-core/api/v2/model"
	"sealdice-core/dice"
)

// BaseService 基础服务，封装dice依赖
type BaseService struct {
	dice *dice.Dice
}

// NewBaseService 创建新的基础服务实例
func NewBaseService(d *dice.Dice) *BaseService {
	return &BaseService{
		dice: d,
	}
}

// RegisterRoutes 注册基础模块的路由
func (s *BaseService) RegisterRoutes(api huma.API) {
	// 注册健康检查接口
	huma.Register(api, huma.Operation{
		OperationID: "getHealth",
		Method:      http.MethodGet,
		Path:        "/base/health",
		Summary:     "健康检查",
		Description: "检查服务健康状态",
		Tags:        []string{"base"},
	}, s.health)
}

// health 健康检查处理函数
func (s *BaseService) health(_ context.Context, _ *struct{}) (*resp2.ItemResponse[HealthData], error) {
	data := HealthData{
		Status:   "ok",
		TestMode: s.dice.Parent.JustForTest,
	}
	resp := &resp2.ItemResponse[HealthData]{}
	resp.Body.Item = data
	return resp, nil
}
