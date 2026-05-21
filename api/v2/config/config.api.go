package config

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/robfig/cron/v3"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/public_dice"
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

func (s *Service) Dice() *dice.Dice {
	return s.dice
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/reply", s.GetReplyConfig, func(o *huma.Operation) {
		o.Description = "获取自定义回复总开关配置"
	})
	huma.Get(grp, "/advanced", s.GetAdvancedConfig, func(o *huma.Operation) {
		o.Description = "获取高级设置配置"
	})
	huma.Get(grp, "/public-dice", s.GetPublicDiceConfig, func(o *huma.Operation) {
		o.Description = "获取公骰设置配置和可上报终端列表"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Put(grp, "/reply", s.SetReplyConfig, func(o *huma.Operation) {
		o.Description = "保存自定义回复总开关配置"
	})
	huma.Put(grp, "/advanced", s.SetAdvancedConfig, func(o *huma.Operation) {
		o.Description = "保存高级设置配置"
	})
	huma.Put(grp, "/public-dice", s.SetPublicDiceConfig, func(o *huma.Operation) {
		o.Description = "保存公骰设置配置和上报终端选择"
	})
}

func (s *Service) GetReplyConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[ReplyModuleConfig], error) {
	return response.NewItemResponse(ReplyModuleConfig{
		CustomReplyConfigEnable: s.dice.Config.CustomReplyConfigEnable,
	}), nil
}

func (s *Service) SetReplyConfig(_ context.Context, req *ReplyConfigReq) (*response.ItemResponse[ReplyModuleConfig], error) {
	s.dice.Config.CustomReplyConfigEnable = req.Body.CustomReplyConfigEnable
	s.dice.MarkModified()
	s.dice.Save(false)
	return response.NewItemResponse(ReplyModuleConfig{
		CustomReplyConfigEnable: s.dice.Config.CustomReplyConfigEnable,
	}), nil
}

func (s *Service) GetAdvancedConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[dice.AdvancedConfig], error) {
	return response.NewItemResponse(s.dice.AdvancedConfig), nil
}

func (s *Service) SetAdvancedConfig(_ context.Context, req *AdvancedConfigReq) (*response.ItemResponse[dice.AdvancedConfig], error) {
	s.dice.AdvancedConfig = req.Body
	s.dice.MarkModified()
	s.dm.Save()
	return response.NewItemResponse(s.dice.AdvancedConfig), nil
}

func (s *Service) GetPublicDiceConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[PublicDiceInfoResp], error) {
	return response.NewItemResponse(s.buildPublicDiceInfoResp()), nil
}

func (s *Service) SetPublicDiceConfig(_ context.Context, req *PublicDiceUpdateReq) (*response.ItemResponse[PublicDiceInfoResp], error) {
	s.dice.Config.PublicDiceConfig = toDicePublicDiceConfig(req.Body.Config)

	selected := make(map[string]bool, len(req.Body.SelectedEndpointIDs))
	for _, id := range req.Body.SelectedEndpointIDs {
		selected[id] = true
	}
	for _, ep := range s.dice.ImSession.EndPoints {
		ep.IsPublic = selected[ep.ID]
	}

	s.ensurePublicDiceRuntime()
	s.dice.PublicDiceInfoRegister()
	s.dice.PublicDiceEndpointRefresh()
	s.dice.PublicDiceSetupTick()
	s.dice.MarkModified()
	s.dice.Save(false)

	return response.NewItemResponse(s.buildPublicDiceInfoResp()), nil
}

func (s *Service) buildPublicDiceInfoResp() PublicDiceInfoResp {
	endpoints := make([]PublicDiceEndpointItem, 0, len(s.dice.ImSession.EndPoints))
	for _, ep := range s.dice.ImSession.EndPoints {
		endpoints = append(endpoints, PublicDiceEndpointItem{
			ID:           ep.ID,
			UserID:       ep.UserID,
			Platform:     ep.Platform,
			ProtocolType: ep.ProtocolType,
			State:        ep.State,
			IsPublic:     ep.IsPublic,
		})
	}

	return PublicDiceInfoResp{
		Config:    fromDicePublicDiceConfig(s.dice.Config.PublicDiceConfig),
		Endpoints: endpoints,
	}
}

func (s *Service) ensurePublicDiceRuntime() {
	if s.dice.PublicDice == nil {
		s.dice.PublicDice = public_dice.NewClient("https://api.weizaima.com", "")
	}
	if s.dice.Cron == nil {
		s.dice.Cron = cron.New()
	}
}

func fromDicePublicDiceConfig(cfg dice.PublicDiceConfig) PublicDiceConfig {
	return PublicDiceConfig{
		PublicDiceEnable: cfg.Enable,
		PublicDiceID:     cfg.ID,
		PublicDiceName:   cfg.Name,
		PublicDiceBrief:  cfg.Brief,
		PublicDiceNote:   cfg.Note,
		PublicDiceAvatar: cfg.Avatar,
	}
}

func toDicePublicDiceConfig(cfg PublicDiceConfig) dice.PublicDiceConfig {
	return dice.PublicDiceConfig{
		Enable: cfg.PublicDiceEnable,
		ID:     cfg.PublicDiceID,
		Name:   cfg.PublicDiceName,
		Brief:  cfg.PublicDiceBrief,
		Note:   cfg.PublicDiceNote,
		Avatar: cfg.PublicDiceAvatar,
	}
}
