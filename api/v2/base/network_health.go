package base

import (
	"context"

	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

// NetworkHealth 返回机器人平台 API 的连通性。
// 目标的展示名称与标签由后端统一提供，前端不再维护目标列表。
func (s *BaseService) NetworkHealth(_ context.Context, _ *request.Empty) (*response.ItemResponse[NetworkHealthData], error) {
	result := s.networkHealth()

	targets := make([]NetworkHealthTarget, len(result.Targets))
	for index, target := range result.Targets {
		targets[index] = NetworkHealthTarget{
			Target:     target.Target,
			Name:       target.Name,
			Label:      target.Label,
			OK:         target.OK,
			DurationMs: target.Duration.Milliseconds(),
		}
	}

	return response.NewItemResponse(NetworkHealthData{
		Total:     result.Total,
		OK:        result.OK,
		Targets:   targets,
		Timestamp: result.Timestamp,
	}), nil
}
