package backup

type FileItem struct {
	Name      string `json:"name"`
	FileSize  int64  `json:"fileSize"`
	Selection int64  `json:"selection"`
}

type FileListResp struct {
	Items []*FileItem `json:"items"`
}

type DownloadReq struct {
	Name string `query:"name"`
}

type DownloadResp struct {
	TestMode bool   `json:"testMode,omitempty"`
	Name     string `json:"name,omitempty"`
	Data     string `json:"data,omitempty"`
}

type BkupDeleteReq struct {
	Name string `json:"name"`
}

type ExecReq struct {
	Selection uint64 `json:"selection"`
}

type Config struct {
	AutoBackupEnable     bool   `json:"autoBackupEnable"`
	AutoBackupTime       string `json:"autoBackupTime"`
	AutoBackupSelection  uint64 `json:"autoBackupSelection"`
	BackupCleanStrategy  int    `json:"backupCleanStrategy"`
	BackupCleanKeepCount int    `json:"backupCleanKeepCount"`
	BackupCleanKeepDur   string `json:"backupCleanKeepDur"`
	BackupCleanTrigger   int    `json:"backupCleanTrigger"`
	BackupCleanCron      string `json:"backupCleanCron"`
}
