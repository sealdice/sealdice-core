package censor

import (
	"github.com/danielgtaylor/huma/v2"

	censorcore "sealdice-core/dice/censor"
	"sealdice-core/model"
)

type CensorStatusResp struct {
	Enable    bool `json:"enable"`
	IsLoading bool `json:"isLoading"`
	TestMode  bool `json:"testMode,omitempty"`
}

type CensorLevelConfig struct {
	Threshold int      `json:"threshold"`
	Handlers  []string `json:"handlers"`
	Score     int      `json:"score"`
}

type CensorLevelConfigs struct {
	Notice  CensorLevelConfig `json:"notice"`
	Caution CensorLevelConfig `json:"caution"`
	Warning CensorLevelConfig `json:"warning"`
	Danger  CensorLevelConfig `json:"danger"`
}

type CensorConfigBody struct {
	Mode          int                `json:"mode"`
	CaseSensitive bool               `json:"caseSensitive"`
	MatchPinyin   bool               `json:"matchPinyin"`
	FilterRegex   string             `json:"filterRegex"`
	LevelConfig   CensorLevelConfigs `json:"levelConfig"`
}

type CensorConfigReq struct {
	Body CensorConfigBody `json:"body"`
}

type CensorConfigResp = CensorConfigBody

type CensorRelatedWord struct {
	Word   string `json:"word"`
	Reason int    `json:"reason"`
}

type CensorWordItem struct {
	Main    string              `json:"main"`
	Level   int                 `json:"level"`
	Related []CensorRelatedWord `json:"related"`
}

type CensorWordsResp struct {
	Data []*CensorWordItem `json:"data"`
}

type CensorFileInfo struct {
	Key      string                  `json:"key"`
	Count    *censorcore.FileCounter `json:"count"`
	FileType string                  `json:"fileType"`
	Name     string                  `json:"name"`
	Author   string                  `json:"author"`
	Version  string                  `json:"version"`
	Desc     string                  `json:"desc"`
	License  string                  `json:"license"`
}

type CensorFilesResp struct {
	Data []*CensorFileInfo `json:"data"`
}

type CensorUploadForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type CensorUploadReq struct {
	RawBody huma.MultipartFormFiles[CensorUploadForm]
}

type CensorSimpleResp struct {
	Success  bool   `json:"success"`
	TestMode bool   `json:"testMode,omitempty"`
	Err      string `json:"err,omitempty"`
}

type CensorDeleteFilesReqBody struct {
	Keys []string `json:"keys"`
}

type CensorDeleteFilesReq struct {
	Body CensorDeleteFilesReqBody `json:"body"`
}

type CensorLogPageQuery struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	UserID   string `query:"userId"`
	Level    int    `query:"level"`
}

type CensorLogPageResp struct {
	Data     []model.CensorLog `json:"data"`
	Total    int64             `json:"total"`
	PageNum  int               `json:"pageNum"`
	PageSize int               `json:"pageSize"`
}
