package upgrade

// ExecResp 在线升级的执行结果。
// Success 为 true 表示新版本已下载完成，升级流程已启动，服务即将离线；
// 为 false 时 Err 给出未能启动升级的原因。
type ExecResp struct {
	Success bool   `json:"success" doc:"是否已启动升级"`
	Err     string `json:"err,omitempty" doc:"未能启动升级的原因"`
}
