package base

type LoginRequest struct {
	Password string `json:"password"`
}

type HealthData struct {
	Status      string `json:"status" example:"ok" doc:"系统状态"`
	TestMode    bool   `json:"testMode" example:"false" doc:"是否为测试模式"`
	Initialized bool   `json:"initialized" example:"true" doc:"骰子实例是否已初始化"`
}

type NetworkHealthTarget struct {
	Target     string `json:"target" doc:"检测目标"`
	OK         bool   `json:"ok" doc:"是否可连接"`
	DurationMs int64  `json:"durationMs" doc:"平均延迟毫秒"`
}

type NetworkHealthData struct {
	Total     int                   `json:"total" doc:"检测目标总数"`
	OK        []string              `json:"ok" doc:"可连接目标列表"`
	Targets   []NetworkHealthTarget `json:"targets" doc:"逐目标检测结果"`
	Timestamp int64                 `json:"timestamp" doc:"检测完成时间"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type LoginSaltResponse struct {
	Salt string `json:"salt"`
}

type VersionDetail struct {
	Major         uint64 `json:"major"`
	Minor         uint64 `json:"minor"`
	Patch         uint64 `json:"patch"`
	Prerelease    string `json:"prerelease"`
	BuildMetaData string `json:"buildMetaData"`
}

type VersionInfo struct {
	Value      string        `json:"value"`
	Simple     string        `json:"simple"`
	Code       int64         `json:"code"`
	Detail     VersionDetail `json:"detail"`
	Latest     string        `json:"latest"`
	LatestNote string        `json:"latestNote"`
	LatestCode int64         `json:"latestCode"`
}

type RuntimeInfo struct {
	Uptime        int64  `json:"uptime"`
	OS            string `json:"OS"`
	Arch          string `json:"arch"`
	JustForTest   bool   `json:"justForTest"`
	ContainerMode bool   `json:"containerMode"`
}

type MemoryInfo struct {
	Alloc   uint64 `json:"alloc"`
	Sys     uint64 `json:"sys"`
	UsedSys uint64 `json:"usedSys"`
}

type OverviewData struct {
	AppName    string      `json:"appName"`
	AppChannel string      `json:"appChannel"`
	ExtraTitle string      `json:"extraTitle"`
	Version    VersionInfo `json:"version"`
	Runtime    RuntimeInfo `json:"runtime"`
	Memory     MemoryInfo  `json:"memory"`
}

type LoginReq struct {
	Body LoginRequest
}
