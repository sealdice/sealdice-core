package api

import "sealdice-core/utils/networkhealth"

// runNetworkHealthCheck 保持 V1 接口原有的机器人平台 API 检测语义。
// 具体探测逻辑与 V2 共用 utils/networkhealth，避免两条接口再次分叉。
func runNetworkHealthCheck() networkhealth.Result {
	return networkhealth.Run()
}
