package helpdoc

import (
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
)

type StatusResp struct {
	Loading  bool `json:"loading"`
	TestMode bool `json:"testMode,omitempty"`
}

type TreeResp struct {
	Data []*dice.HelpDoc `json:"data"`
}

type HelpTextItemQuery struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	ID       string `query:"id"`
	Group    string `query:"group"`
	From     string `query:"from"`
	Title    string `query:"title"`
}

type HelpTextItemPageResp struct {
	Total    int              `json:"total"`
	Data     dice.HelpTextVos `json:"data"`
	PageNum  int              `json:"pageNum"`
	PageSize int              `json:"pageSize"`
}

type HelpConfigBody struct {
	Aliases map[string][]string `json:"aliases"`
}

type ConfigReq struct {
	Body request.RequestWrapper[HelpConfigBody] `json:"body"`
}

type ConfigResp = HelpConfigBody

type DeleteReqBody struct {
	Keys []string `json:"keys"`
}

type DeleteReq struct {
	Body request.RequestWrapper[DeleteReqBody] `json:"body"`
}

type SimpleResp struct {
	Success  bool   `json:"success"`
	TestMode bool   `json:"testMode,omitempty"`
	Err      string `json:"err,omitempty"`
}

type HelpDocUploadInitReqBody struct {
	Group     string `json:"group"`
	Filename  string `json:"filename"`
	FileSize  int64  `json:"fileSize"`
	FileHash  string `json:"fileHash"`
	ChunkSize int64  `json:"chunkSize"`
}

type UploadInitReq struct {
	Body request.RequestWrapper[HelpDocUploadInitReqBody] `json:"body"`
}

type HelpDocUploadSessionResp struct {
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

type HelpDocUploadChunkResp struct {
	Success       bool  `json:"success"`
	UploadedBytes int64 `json:"uploadedBytes"`
	UploadedChunk int   `json:"uploadedChunk"`
}

type HelpDocUploadCompleteReqBody struct {
	SessionID string `json:"sessionId"`
}

type UploadCompleteReq struct {
	Body request.RequestWrapper[HelpDocUploadCompleteReqBody] `json:"body"`
}

type HelpDocUploadCompleteResp struct {
	Success  bool   `json:"success"`
	TestMode bool   `json:"testMode,omitempty"`
	Filename string `json:"filename"`
	Group    string `json:"group"`
}
