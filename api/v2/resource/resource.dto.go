package resource

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/model/common/response"
)

type ResourceType string

const (
	ResourceTypeImage   ResourceType = "image"
	ResourceTypeUnknown ResourceType = "unknown"
)

type ListQuery struct {
	Page      int    `query:"page"`
	PageSize  int    `query:"pageSize"`
	Type      string `query:"type"`
	Keyword   string `query:"keyword"`
	SortBy    string `query:"sortBy"`
	SortOrder string `query:"sortOrder"`
}

type ResourceItem struct {
	Type ResourceType `json:"type"`
	Name string       `json:"name"`
	Ext  string       `json:"ext"`
	Path string       `json:"path"`
	Size int64        `json:"size"`
}

type ResourceListResp = response.HPageResult[*ResourceItem]

type ResourcePathQuery struct {
	Path string `query:"path"`
}

type ResourceDataQuery struct {
	Path      string `query:"path"`
	Thumbnail bool   `query:"thumbnail"`
}

type ResourcePathReqBody struct {
	Path string `json:"path"`
}

type DeleteReq struct {
	Body ResourcePathReqBody `json:"body"`
}

type UploadForm struct {
	Files []huma.FormFile `form:"files" required:"true"`
}

type UploadReq struct {
	RawBody huma.MultipartFormFiles[UploadForm]
}

type ResourceUploadResp struct {
	Success  bool     `json:"success"`
	TestMode bool     `json:"testMode,omitempty"`
	Failed   []string `json:"failed,omitempty"`
}
