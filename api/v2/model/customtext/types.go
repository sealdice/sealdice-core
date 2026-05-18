package customtext

import (
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
)

type TextResp struct {
	Texts       dice.TextTemplateWithWeightDict                   `json:"texts"`
	HelpInfo    dice.TextTemplateWithHelpDict                     `json:"helpInfo"`
	PreviewInfo map[string]map[string]dice.TextItemCompatibleInfo `json:"previewInfo"`
}

type CategoryPath struct {
	Category string `path:"category"`
}

type SaveCategoryBody struct {
	Data dice.TextTemplateWithWeight `json:"data"`
}

type SaveCategoryReq struct {
	Category string                                   `path:"category"`
	Body     request.RequestWrapper[SaveCategoryBody] `json:"body"`
}

type PreviewRefreshReq struct {
	Category string `path:"category"`
}
