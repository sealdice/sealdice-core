package model

// VersionDetail 版本号详细信息
type VersionDetail struct {
	Major         uint64 `json:"major"`
	Minor         uint64 `json:"minor"`
	Patch         uint64 `json:"patch"`
	Prerelease    string `json:"prerelease"`
	BuildMetaData string `json:"buildMetaData"`
}

type BaseInfoResponse struct {
	AppName        string        `json:"appName"`
	AppChannel     string        `json:"appChannel"`
	Version        string        `json:"version"`
	VersionSimple  string        `json:"versionSimple"`
	VersionDetail  VersionDetail `json:"versionDetail"`
	VersionNew     string        `json:"versionNew"`
	VersionNewNote string        `json:"versionNewNote"`
	VersionCode    int64         `json:"versionCode"`
	VersionNewCode int64         `json:"versionNewCode"`
	MemoryAlloc    uint64        `json:"memoryAlloc"`
	Uptime         int64         `json:"uptime"`
	MemoryUsedSys  uint64        `json:"memoryUsedSys"`
	ExtraTitle     string        `json:"extraTitle"`
	OS             string        `json:"OS"`
	Arch           string        `json:"arch"`
	JustForTest    bool          `json:"justForTest"`
	ContainerMode  bool          `json:"containerMode"`
}
