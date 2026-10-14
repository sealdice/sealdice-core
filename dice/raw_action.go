package dice

import (
	"context"
	"errors"
	"fmt"
)

// RawActionAdapter 可选接口：支持 sendRaw 动作透传的适配器实现之。
// 未实现的适配器在 SendRaw 分发时被类型断言自然跳过。
type RawActionAdapter interface {
	RawAction(ctx context.Context, action string, params map[string]any) (any, error)
}

// rawActionEndpointKey 是 params 里的保留键，用于指定由哪个账号端点执行动作。
// SendRaw 会把它从 params 中剥离，不会透传给适配器。
const rawActionEndpointKey = "endpoint_id"

// SendRaw 出站动作透传：按平台定位在线端点，校验能力清单后调用适配器。
// 只要该协议能力清单声明了此动作即可调用，不另设权限。
// 同平台存在多个在线端点时默认取第一个；可通过 params[endpoint_id] 指定账号（端点 UserID）。
func (d *Dice) SendRaw(platform, action string, params map[string]any) (any, error) {
	if d.ImSession == nil {
		return nil, errors.New("会话未初始化")
	}
	wantEndpoint := ""
	if params != nil {
		if v, ok := params[rawActionEndpointKey]; ok {
			s, isStr := v.(string)
			if !isStr {
				return nil, fmt.Errorf("参数 %s 必须是字符串（端点 UserID）", rawActionEndpointKey)
			}
			wantEndpoint = s
			filtered := make(map[string]any, len(params)-1)
			for k, val := range params {
				if k != rawActionEndpointKey {
					filtered[k] = val
				}
			}
			params = filtered
		}
	}

	var fallback *EndPointInfo
	for _, ep := range d.ImSession.EndPoints {
		if ep == nil || ep.Adapter == nil || ep.Platform != platform || !ep.Enable {
			continue
		}
		caps, ok := adapterCapabilitiesFor(ep)
		if !ok {
			continue
		}
		if _, declared := caps.RawActions[action]; !declared {
			continue // 该协议未声明此动作，尝试下一个同平台端点
		}
		ra, isRaw := ep.Adapter.(RawActionAdapter)
		if !isRaw {
			continue
		}
		if wantEndpoint != "" {
			if ep.UserID != wantEndpoint {
				continue
			}
			if params == nil {
				params = map[string]any{}
			}
			return ra.RawAction(context.Background(), action, params)
		}
		if fallback == nil {
			fallback = ep
		}
	}
	if wantEndpoint != "" {
		return nil, fmt.Errorf("端点 %s 不存在、不可用或不支持动作 %s", wantEndpoint, action)
	}
	if fallback != nil {
		ra, _ := fallback.Adapter.(RawActionAdapter)
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
