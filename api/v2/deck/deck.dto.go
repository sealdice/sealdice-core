package deck

import (
	"github.com/danielgtaylor/huma/v2"

	deckd "sealdice-core/dice"
	"sealdice-core/model/common/response"
)

type ListQuery struct {
	Page      int    `query:"page"`
	PageSize  int    `query:"pageSize"`
	Keyword   string `query:"keyword"`
	SortBy    string `query:"sortBy"`
	SortOrder string `query:"sortOrder"`
}

type DeckItem struct {
	Enable        bool            `json:"enable"`
	ErrText       string          `json:"errText"`
	Filename      string          `json:"filename"`
	Format        string          `json:"format"`
	FormatVersion int64           `json:"formatVersion"`
	FileFormat    string          `json:"fileFormat"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Author        string          `json:"author"`
	License       string          `json:"license"`
	Command       map[string]bool `json:"command"`
	Date          string          `json:"date"`
	UpdateDate    string          `json:"updateDate"`
	Desc          string          `json:"desc"`
	UpdateUrls    []string        `json:"updateUrls"`
	Etag          string          `json:"etag"`
	Cloud         bool            `json:"cloud"`
	StoreID       string          `json:"storeID"`
}

type DeckListResp = response.HPageResult[*DeckItem]

type ReloadResp struct {
	Success  bool `json:"success"`
	TestMode bool `json:"testMode"`
}

type FilenameReqBody struct {
	Filename string `json:"filename"`
}

type FilenameReq struct {
	Body FilenameReqBody `json:"body"`
}

type UpdateReqBody struct {
	Filename     string `json:"filename"`
	TempFileName string `json:"tempFileName"`
}

type UpdateReq struct {
	Body UpdateReqBody `json:"body"`
}

type UpdateCheckResult struct {
	Success      bool   `json:"success"`
	Err          string `json:"err,omitempty"`
	Old          string `json:"old,omitempty"`
	New          string `json:"new,omitempty"`
	Format       string `json:"format,omitempty"`
	Filename     string `json:"filename,omitempty"`
	TempFileName string `json:"tempFileName,omitempty"`
}

type UploadForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type UploadReq struct {
	RawBody huma.MultipartFormFiles[UploadForm]
}

type UploadResp struct {
	Success  bool `json:"success"`
	TestMode bool `json:"testMode"`
}

type UploadInitReqBody struct {
	Filename  string `json:"filename"`
	FileSize  int64  `json:"fileSize"`
	FileHash  string `json:"fileHash"`
	ChunkSize int64  `json:"chunkSize"`
}

type UploadInitReq struct {
	Body UploadInitReqBody `json:"body"`
}

type UploadSessionResp struct {
	Success         bool   `json:"success"`
	SessionID       string `json:"sessionId"`
	ChunkSize       int64  `json:"chunkSize"`
	UploadedChunks  []int  `json:"uploadedChunks"`
	UploadedBytes   int64  `json:"uploadedBytes"`
	ExpectedChunks  int    `json:"expectedChunks"`
	ResumeSupported bool   `json:"resumeSupported"`
}

type UploadChunkPath struct {
	SessionID string `path:"sessionId"`
	Index     int    `path:"index"`
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

type UploadChunkResp struct {
	Success       bool  `json:"success"`
	UploadedBytes int64 `json:"uploadedBytes"`
	UploadedChunk int   `json:"uploadedChunk"`
}

type UploadCompleteReqBody struct {
	SessionID string `json:"sessionId"`
}

type UploadCompleteReq struct {
	Body UploadCompleteReqBody `json:"body"`
}

type UploadCompleteResp struct {
	Success  bool   `json:"success"`
	TestMode bool   `json:"testMode"`
	Filename string `json:"filename"`
}

func FromDeckInfo(item *deckd.DeckInfo) *DeckItem {
	if item == nil {
		return nil
	}
	return &DeckItem{
		Enable:        item.Enable,
		ErrText:       item.ErrText,
		Filename:      item.Filename,
		Format:        item.Format,
		FormatVersion: item.FormatVersion,
		FileFormat:    item.FileFormat,
		Name:          item.Name,
		Version:       item.Version,
		Author:        item.Author,
		License:       item.License,
		Command:       item.Command,
		Date:          item.Date,
		UpdateDate:    item.UpdateDate,
		Desc:          item.Desc,
		UpdateUrls:    item.UpdateUrls,
		Etag:          item.Etag,
		Cloud:         item.Cloud,
		StoreID:       item.StoreID,
	}
}

type DeleteResp = response.SimpleOK
