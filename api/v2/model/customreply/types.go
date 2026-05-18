package customreply

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type FileInfo struct {
	Enable          bool   `json:"enable"`
	Filename        string `json:"filename"`
	CreateTimestamp int64  `json:"createTimestamp"`
	UpdateTimestamp int64  `json:"updateTimestamp"`
	ItemCount       int    `json:"itemCount"`
}

type ReplyFileListResp = response.HPageResult[*FileInfo]

type FilenamePath struct {
	Filename string `path:"filename"`
}

type FileListQuery struct {
	Page      int    `query:"page"`
	PageSize  int    `query:"pageSize"`
	Keyword   string `query:"keyword"`
	SortBy    string `query:"sortBy"`
	SortOrder string `query:"sortOrder"`
}

type RulePageQuery struct {
	Filename string `path:"filename"`
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

type ConditionPageQuery struct {
	Filename string `path:"filename"`
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

type FileBody struct {
	Filename string `json:"filename"`
}

type FileReq struct {
	Body request.RequestWrapper[FileBody] `json:"body"`
}

type SaveReq struct {
	Filename string                                   `path:"filename"`
	Body     request.RequestWrapper[dice.ReplyConfig] `json:"body"`
}

type DebugModeResp struct {
	Value bool `json:"value"`
}

type DebugModeReq struct {
	Body request.RequestWrapper[DebugModeResp] `json:"body"`
}

type UploadForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type UploadReq struct {
	RawBody huma.MultipartFormFiles[UploadForm]
}

type RuleInfo struct {
	Index int             `json:"index"`
	Item  *dice.ReplyItem `json:"item"`
}

type RulePageResp = response.HPageResult[*RuleInfo]

type ConditionInfo struct {
	Index int                     `json:"index"`
	Item  dice.ReplyConditionBase `json:"item"`
}

type ConditionPageResp = response.HPageResult[*ConditionInfo]

type ReplyFileDetail struct {
	Enable          bool                 `json:"enable"`
	Interval        float64              `json:"interval"`
	Name            string               `json:"name"`
	Author          []string             `json:"author"`
	Version         string               `json:"version"`
	CreateTimestamp int64                `json:"createTimestamp"`
	UpdateTimestamp int64                `json:"updateTimestamp"`
	Desc            string               `json:"desc"`
	StoreID         string               `json:"storeID"`
	Conditions      dice.ReplyConditions `json:"conditions"`
	Filename        string               `json:"filename"`
	ItemCount       int                  `json:"itemCount"`
}
