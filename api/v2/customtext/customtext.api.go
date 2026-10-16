package customtext

import (
	"context"
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct{ dice *dice.Dice }

func NewService(dm *dice.DiceManager) *Service { return &Service{dice: dm.GetDice()} }

func (s *Service) Dice() *dice.Dice {
	return s.dice
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/", s.GetText, func(o *huma.Operation) {
		o.Description = "获取自定义文案"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Put(grp, "/{category}", s.SaveCategory, func(o *huma.Operation) {
		o.Description = "保存指定分类的自定义文案"
	})
	huma.Post(grp, "/{category}/preview-refresh", s.PreviewRefresh, func(o *huma.Operation) {
		o.Description = "刷新指定分类的文案兼容性预览"
	})
}

func (s *Service) GetText(_ context.Context, _ *request.Empty) (*response.ItemResponse[TextResp], error) {
	return response.NewItemResponse[TextResp](TextResp{
		Texts:       s.dice.TextMapRaw,
		HelpInfo:    s.dice.TextMapHelpInfo,
		PreviewInfo: previewInfoToMap(&s.dice.TextMapCompatible),
	}), nil
}

func (s *Service) SaveCategory(_ context.Context, req *SaveCategoryReq) (*response.ItemResponse[response.SimpleOK], error) {
	if err := s.dice.SaveTextCategory(req.Category, req.Body.Data); err != nil {
		if errors.Is(err, dice.ErrInvalidConfig) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		s.dice.Logger.Errorf("保存文案失败: %v", err)
		return nil, huma.Error500InternalServerError("保存文案失败")
	}
	return response.NewItemResponse[response.SimpleOK](response.SimpleOK{Success: true}), nil
}

func (s *Service) PreviewRefresh(_ context.Context, req *PreviewRefreshReq) (*response.ItemResponse[response.SimpleOK], error) {
	category := strings.TrimSpace(req.Category)
	if category == "" {
		return nil, huma.Error400BadRequest("missing category")
	}
	for key, item := range s.dice.TextMapRaw[category] {
		dice.TextMapCompatibleCheck(s.dice, category, key, item)
	}
	return response.NewItemResponse[response.SimpleOK](response.SimpleOK{Success: true}), nil
}

func previewInfoToMap(src *dice.TextTemplateCompatibleDict) map[string]map[string]dice.TextItemCompatibleInfo {
	out := map[string]map[string]dice.TextItemCompatibleInfo{}
	if src == nil {
		return out
	}
	src.Range(func(key string, value *dice.SyncMap[string, dice.TextItemCompatibleInfo]) bool {
		items := map[string]dice.TextItemCompatibleInfo{}
		if value != nil {
			value.Range(func(text string, info dice.TextItemCompatibleInfo) bool {
				items[text] = info
				return true
			})
		}
		out[key] = items
		return true
	})
	return out
}
