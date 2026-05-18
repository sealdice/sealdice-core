package config

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	configm "sealdice-core/api/v2/model/config"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/reply", s.GetReplyConfig, func(o *huma.Operation) {
		o.Description = "获取自定义回复总开关配置"
	})
	huma.Get(grp, "/advanced", s.GetAdvancedConfig, func(o *huma.Operation) {
		o.Description = "获取高级设置配置"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Put(grp, "/reply", s.SetReplyConfig, func(o *huma.Operation) {
		o.Description = "保存自定义回复总开关配置"
	})
	huma.Put(grp, "/advanced", s.SetAdvancedConfig, func(o *huma.Operation) {
		o.Description = "保存高级设置配置"
	})
}

func (s *Service) GetReplyConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[configm.ReplyModuleConfig], error) {
	return response.NewItemResponse(configm.ReplyModuleConfig{
		CustomReplyConfigEnable: s.dice.Config.CustomReplyConfigEnable,
	}), nil
}

func (s *Service) SetReplyConfig(_ context.Context, req *configm.ReplyConfigReq) (*response.ItemResponse[configm.ReplyModuleConfig], error) {
	s.dice.Config.CustomReplyConfigEnable = req.Body.Body.CustomReplyConfigEnable
	s.dice.MarkModified()
	s.dice.Save(false)
	return response.NewItemResponse(configm.ReplyModuleConfig{
		CustomReplyConfigEnable: s.dice.Config.CustomReplyConfigEnable,
	}), nil
}

func (s *Service) GetAdvancedConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[dice.AdvancedConfig], error) {
	return response.NewItemResponse(s.dice.AdvancedConfig), nil
}

func (s *Service) SetAdvancedConfig(_ context.Context, req *configm.AdvancedConfigReq) (*response.ItemResponse[dice.AdvancedConfig], error) {
	s.dice.AdvancedConfig = req.Body.Body
	s.dice.MarkModified()
	s.dm.Save()
	return response.NewItemResponse(s.dice.AdvancedConfig), nil
}
