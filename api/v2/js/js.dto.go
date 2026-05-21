package js

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/response"
)

// --- core query / path types ---

type ListQuery struct {
	Page      int    `query:"page"`
	PageSize  int    `query:"pageSize"`
	Keyword   string `query:"keyword"`
	SortBy    string `query:"sortBy"`
	SortOrder string `query:"sortOrder"`
}

type NamePath struct {
	Name string `path:"name"`
}

type DataListQuery struct {
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
	Keyword  string `query:"keyword"`
}

type DataGetQuery struct {
	Key string `query:"key"`
}

// --- request bodies ---

type JsFilenameReqBody struct {
	Filename string `json:"filename"`
}

type FilenameReq struct {
	Body JsFilenameReqBody `json:"body"`
}

type JsNameReqBody struct {
	Name string `json:"name"`
}

type NameReq struct {
	Body JsNameReqBody `json:"body"`
}

type JsExecuteReqBody struct {
	Value string `json:"value"`
}

type ExecuteReq struct {
	Body JsExecuteReqBody `json:"body"`
}

type JsCheckUpdateReqBody struct {
	Filename string `json:"filename"`
}

type CheckUpdateReq struct {
	Body JsCheckUpdateReqBody `json:"body"`
}

type JsUpdateReqBody struct {
	Filename     string `json:"filename"`
	TempFileName string `json:"tempFileName"`
}

type UpdateReq struct {
	Body JsUpdateReqBody `json:"body"`
}

type JsDataSetReqBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DataSetReq struct {
	Body JsDataSetReqBody `json:"body"`
}

type JsDataDeleteReqBody struct {
	Keys []string `json:"keys"`
}

type DataDeleteReq struct {
	Body JsDataDeleteReqBody `json:"body"`
}

type JsDeleteDeadConfigsReqBody struct {
	Names []string `json:"names"`
}

type DeleteDeadConfigsReq struct {
	Body JsDeleteDeadConfigsReqBody `json:"body"`
}

// --- config request types ---

type JsSetConfigsReqBody struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

type SetConfigsReq struct {
	Body JsSetConfigsReqBody `json:"body"`
}

type JsResetConfigReqBody struct {
	Name string   `json:"name"`
	Keys []string `json:"keys"`
}

type ResetConfigReq struct {
	Body JsResetConfigReqBody `json:"body"`
}

// --- config request types ---

// --- response types ---

type JsListResp struct {
	Success  bool     `json:"success"`
	Data     []JsInfo `json:"data"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

type JsInfo struct {
	dice.JsScriptInfo
	HasConfig      bool `json:"hasConfig"`
	BuiltinUpdated bool `json:"builtinUpdated"`
}

type JsStatusResp struct {
	Status bool `json:"status"`
}

type JsExecuteResp struct {
	Ret     interface{} `json:"ret,omitempty"`
	Outputs []string    `json:"outputs"`
	Err     string      `json:"err,omitempty"`
}

type JsCheckUpdateResp struct {
	Success      bool   `json:"success"`
	Old          string `json:"old,omitempty"`
	New          string `json:"new,omitempty"`
	Format       string `json:"format,omitempty"`
	Filename     string `json:"filename,omitempty"`
	TempFileName string `json:"tempFileName,omitempty"`
	Err          string `json:"err,omitempty"`
}

type JsUpdateResp struct {
	Success bool `json:"success"`
}

type JsSimpleResult struct {
	Success  bool `json:"success"`
	TestMode bool `json:"testMode,omitempty"`
}

// --- KV data types ---

type DataKV struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	IsJSON bool   `json:"isJson"`
}

type DataListResp struct {
	Keys     []DataKV `json:"keys"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

type DataInfoResp struct {
	KeyCount  int   `json:"keyCount"`
	FileSize  int64 `json:"fileSize"`
	CanShrink bool  `json:"canShrink"`
}

// --- dead config types ---

type DeadConfig struct {
	Name     string `json:"name"`
	FileSize int64  `json:"fileSize"`
}

type DeadConfigsResp struct {
	Configs []DeadConfig `json:"configs"`
}

// --- upload form type ---

type UploadForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type UploadReq struct {
	RawBody huma.MultipartFormFiles[UploadForm]
}

type JsUploadResp struct {
	Success  bool `json:"success"`
	TestMode bool `json:"testMode,omitempty"`
}

type JsUploadInitReqBody struct {
	Filename  string `json:"filename"`
	FileSize  int64  `json:"fileSize"`
	FileHash  string `json:"fileHash"`
	ChunkSize int64  `json:"chunkSize"`
}

type UploadInitReq struct {
	Body JsUploadInitReqBody `json:"body"`
}

type JsUploadSessionResp struct {
	Success         bool   `json:"success"`
	SessionID       string `json:"sessionId"`
	ChunkSize       int64  `json:"chunkSize"`
	UploadedChunks  []int  `json:"uploadedChunks"`
	UploadedBytes   int64  `json:"uploadedBytes"`
	ExpectedChunks  int    `json:"expectedChunks"`
	ResumeSupported bool   `json:"resumeSupported"`
}

type UploadChunkQuery struct {
	SessionID string `path:"sessionId"`
	Index     int    `path:"index"`
}

type UploadChunkReq struct {
	SessionID string `path:"sessionId"`
	Index     int    `path:"index"`
	RawBody   []byte
}

type JsUploadChunkResp struct {
	Success       bool  `json:"success"`
	UploadedBytes int64 `json:"uploadedBytes"`
	UploadedChunk int   `json:"uploadedChunk"`
}

type JsUploadCompleteReqBody struct {
	SessionID string `json:"sessionId"`
}

type UploadCompleteReq struct {
	Body JsUploadCompleteReqBody `json:"body"`
}

type JsUploadCompleteResp struct {
	Success  bool   `json:"success"`
	TestMode bool   `json:"testMode,omitempty"`
	Filename string `json:"filename"`
}

// --- config map types ---

type PluginConfigMap map[string]*APIPluginConfig

type APIPluginConfig struct {
	PluginName        string             `json:"pluginName"`
	Configs           []*dice.ConfigItem `json:"configs"`
	OrderedConfigKeys []string           `json:"orderedConfigKeys"`
}

// ---- Huma Typedefs ----

type ListItemResponse = response.ItemResponse[JsListResp]
type StatusItemResponse = response.ItemResponse[JsStatusResp]
type ExecuteItemResponse = response.ItemResponse[JsExecuteResp]
type CheckUpdateItemResponse = response.ItemResponse[JsCheckUpdateResp]
type UpdateItemResponse = response.ItemResponse[JsUpdateResp]
type SimpleItemResponse = response.ItemResponse[JsSimpleResult]
type DataListItemResponse = response.ItemResponse[DataListResp]
type DataValueItemResponse = response.ItemResponse[DataKV]
type DataInfoItemResponse = response.ItemResponse[DataInfoResp]
type DeadConfigsItemResponse = response.ItemResponse[DeadConfigsResp]
type ConfigMapItemResponse = response.ItemResponse[PluginConfigMap]
