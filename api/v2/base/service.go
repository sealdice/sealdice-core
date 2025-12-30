package base

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	nanoid "github.com/matoous/go-nanoid/v2"

	"sealdice-core/dice"
	"sealdice-core/model/api/req"
	"sealdice-core/model/api/resp"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

// BaseService 基础服务，封装dice依赖
type BaseService struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

// NewBaseService 创建新的基础服务实例
// 特殊增加dm降低封装层 - 或许应该传入dm获取dice才是正道。
func NewBaseService(dm *dice.DiceManager) *BaseService {
	return &BaseService{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *BaseService) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/health", s.health, func(o *huma.Operation) {
		o.Description = "检查服务是否正常"
		o.Summary = "检查服务是否正常"
	})
	huma.Post(grp, "/login", s.Login, func(o *huma.Operation) {
		o.Description = "登录获取Token"
		o.Summary = "登录获取Token"
	})
	huma.Get(grp, "/security-check", s.SecurityCheck, func(o *huma.Operation) {
		o.Description = "检查安全状态"
		o.Summary = "检查安全状态"
	})
}

// health 健康检查处理函数
func (s *BaseService) health(_ context.Context, _ *struct{}) (*response.ItemResponse[resp.HealthData], error) {
	if s.dice == nil {
		return nil, huma.Error500InternalServerError("Dice instance is nil,contact administrator")
	}
	data := resp.HealthData{
		Status:   "ok",
		TestMode: s.dice.Parent.JustForTest,
	}
	return response.NewItemResponse[resp.HealthData](data), nil
}

// Login 用户登录
func (s *BaseService) Login(_ context.Context, req *request.RequestWrapper[req.LoginRequest]) (*response.ItemResponse[resp.LoginResponse], error) {
	if s.dm.UIPasswordHash == "" || s.dm.UIPasswordHash == req.Body.Password {
		// 改用一个其他的生成策略简化冗余代码
		head := fmt.Sprintf("%x", time.Now().Unix())
		token, err := nanoid.New(64)
		if err != nil {
			return nil, err
		}
		token += ":" + head
		s.dice.Parent.AccessTokens.Store(token, true)
		s.dice.LastUpdatedTime = time.Now().Unix()
		s.dice.Parent.Save()
		return response.NewItemResponse[resp.LoginResponse](resp.LoginResponse{
			Token: token,
		}), nil
	}
	return nil, huma.Error401Unauthorized("Invalid password")
}

// SecurityCheck 安全配置检查
func (s *BaseService) SecurityCheck(_ context.Context, _ *struct{}) (*response.ItemResponse[bool], error) {
	isPublicService := strings.HasPrefix(s.dm.ServeAddress, "0.0.0.0") || s.dm.ServeAddress == ":3211"
	isEmptyPassword := s.dm.UIPasswordHash == ""
	return response.NewItemResponse[bool](!isEmptyPassword || !isPublicService), nil
}

// 私以为这应该是个WebSocket接口
//func (s *BaseService) GetDiceLogItems(_ context.Context, req *request.RequestWrapper[]) (*response.ItemResponse[bool], error) {
//}
