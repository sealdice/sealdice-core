package story

import (
	"sealdice-core/dice"
	"sealdice-core/model"
	"sealdice-core/model/common/response"
)

type StoryInfo struct {
	TotalLogs    int `json:"totalLogs"`
	TotalItems   int `json:"totalItems"`
	CurrentLogs  int `json:"currentLogs"`
	CurrentItems int `json:"currentItems"`
}

type StoryInfoResp = StoryInfo

type StoryLogView struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	GroupID    string `json:"groupId"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
	Size       *int   `json:"size"`
	UploadURL  string `json:"uploadUrl"`
	UploadTime int64  `json:"uploadTime"`
	LinkState  string `json:"linkState"`
}

type LogPageQuery struct {
	PageNum          int    `query:"pageNum"`
	PageSize         int    `query:"pageSize"`
	Name             string `query:"name"`
	GroupID          string `query:"groupId"`
	CreatedTimeBegin int64  `query:"createdTimeBegin"`
	CreatedTimeEnd   int64  `query:"createdTimeEnd"`
}

type LogPageResp struct {
	Result   bool           `json:"result"`
	Err      string         `json:"err,omitempty"`
	Total    int            `json:"total"`
	Data     []StoryLogView `json:"data,omitempty"`
	PageNum  int            `json:"pageNum"`
	PageSize int            `json:"pageSize"`
}

type ItemPageQuery struct {
	LogID    uint64 `query:"logId"`
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	GroupID  string `query:"groupId"`
	LogName  string `query:"logName"`
}

type LogLinePageResp = []*model.LogOneItem

type ExportParquetQuery struct {
	LogID uint64 `query:"logId"`
}

type DeleteLogReqBody struct {
	ID uint64 `json:"id"`
}

type DeleteLogReq struct {
	Body DeleteLogReqBody `json:"body"`
}

type DeleteLogResp struct {
	Success bool `json:"success"`
}

type UploadLogReqBody struct {
	ID    uint64 `json:"id"`
	Force bool   `json:"force"`
}

type UploadLogReq struct {
	Body UploadLogReqBody `json:"body"`
}

type UploadLogResp struct {
	URL        string `json:"url"`
	Unofficial bool   `json:"unofficial"`
	Reused     bool   `json:"reused"`
	Forced     bool   `json:"forced"`
}

type BackupListResp struct {
	Result bool                  `json:"result"`
	Err    string                `json:"err,omitempty"`
	Data   []dice.StoryLogBackup `json:"data,omitempty"`
}

type BackupDownloadQuery struct {
	Name string `query:"name"`
}

type BackupBatchDeleteReqBody struct {
	Names []string `json:"names"`
}

type BackupBatchDeleteReq struct {
	Body BackupBatchDeleteReqBody `json:"body"`
}

type BackupBatchDeleteResp struct {
	Result bool     `json:"result"`
	Fails  []string `json:"fails,omitempty"`
}

type CleanupPreviewQuery struct {
	Months int `query:"months"`
}

type CleanupPreviewResp struct {
	Logs          int   `json:"logs"`
	Items         int   `json:"items"`
	OldestUpdated int64 `json:"oldestUpdated,omitempty"`
	NewestUpdated int64 `json:"newestUpdated,omitempty"`
	CanVacuum     bool  `json:"canVacuum"`
}

type CleanupReqBody struct {
	Months int  `json:"months"`
	Vacuum bool `json:"vacuum"`
}

type CleanupReq struct {
	Body CleanupReqBody `json:"body"`
}

type CleanupResp struct {
	Logs     int  `json:"logs"`
	Items    int  `json:"items"`
	Vacuumed bool `json:"vacuumed"`
}

type LogPageItemResponse = response.ItemResponse[LogPageResp]
