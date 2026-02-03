package resp

// HealthData 健康检查响应数据
type HealthData struct {
	Status   string `json:"status" example:"ok" doc:"系统状态"`
	TestMode bool   `json:"testMode" example:"false" doc:"是否为测试模式"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
