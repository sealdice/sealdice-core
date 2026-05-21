package customtext

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct {
	dice     *dice.Dice
	dm       *dice.DiceManager
	autoSave bool
}

func NewService(dm *dice.DiceManager) *Service {
	return newService(dm, true)
}

func NewServiceWithAutoSave(dm *dice.DiceManager, autoSave bool) *Service {
	return newService(dm, autoSave)
}

func newService(dm *dice.DiceManager, autoSave bool) *Service {
	return &Service{
		dice:     dm.GetDice(),
		dm:       dm,
		autoSave: autoSave,
	}
}

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
	category := strings.TrimSpace(req.Category)
	if category == "" {
		return nil, huma.Error400BadRequest("missing category")
	}
	data := normalizeTextTemplate(req.Body.Data)
	if s.dice.TextMapRaw == nil {
		s.dice.TextMapRaw = dice.TextTemplateWithWeightDict{}
	}
	s.dice.TextMapRaw[category] = data
	dice.SetupTextHelpInfo(s.dice, s.dice.TextMapHelpInfo, s.dice.TextMapRaw, "configs/text-template.yaml")
	s.dice.GenerateTextMap()
	s.saveText()
	for key, item := range s.dice.TextMapRaw[category] {
		dice.TextMapCompatibleCheck(s.dice, category, key, item)
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

func normalizeTextTemplate(data dice.TextTemplateWithWeight) dice.TextTemplateWithWeight {
	for _, items := range data {
		for _, item := range items {
			if len(item) > 0 {
				if text, ok := item[0].(string); ok {
					item[0] = strings.TrimSpace(text)
				}
			}
			if len(item) > 1 {
				switch weight := item[1].(type) {
				case int:
					item[1] = weight
				case int64:
					item[1] = int(weight)
				case float64:
					item[1] = int(weight)
				case float32:
					item[1] = int(weight)
				}
			}
		}
	}
	return data
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

func (s *Service) saveText() {
	if s.autoSave {
		s.dice.SaveText()
	}
}
