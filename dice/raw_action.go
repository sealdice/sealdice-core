package dice

import (
	"context"
	"fmt"
)

// RawActionAdapter 可选接口：支持 sendRaw 动作透传的适配器实现之。
// 未实现的适配器在 SendRaw 分发时被类型断言自然跳过。
type RawActionAdapter interface {
	RawAction(ctx context.Context, action string, params map[string]any) (any, error)
}

// SendRaw 出站动作透传：按平台定位在线端点，校验能力清单后调用适配器。
// 只要该协议能力清单声明了此动作即可调用，不另设权限。
func (d *Dice) SendRaw(platform, action string, params map[string]any) (any, error) {
	if d.ImSession == nil {
		return nil, fmt.Errorf("会话未初始化")
	}
	for _, ep := range d.ImSession.EndPoints {
		if ep == nil || ep.Adapter == nil {
			continue
		}
		if ep.Platform != platform || !ep.Enable {
			continue
		}
		caps, ok := GetAdapterCapabilities(ep.ProtocolType)
		if !ok {
			continue
		}
		if _, declared := caps.RawActions[action]; !declared {
			continue // 该协议未声明此动作，尝试下一个同平台端点
		}
		ra, ok := ep.Adapter.(RawActionAdapter)
		if !ok {
			continue
		}
		if params == nil {
			params = map[string]any{}
		}
		return ra.RawAction(context.Background(), action, params)
	}
	// 无任何端点声明该动作：给出可定位的错误
	protos := ""
	for _, set := range GetAdapterCapabilitiesByPlatform(platform) {
		if _, declared := set.RawActions[action]; declared {
			protos += set.ProtocolType + ","
		}
	}
	if protos != "" {
		return nil, fmt.Errorf("平台 %s 声明了动作 %s（协议 %s），但没有可用端点", platform, action, protos)
	}
	return nil, fmt.Errorf("平台 %s 不支持动作 %s", platform, action)
}
