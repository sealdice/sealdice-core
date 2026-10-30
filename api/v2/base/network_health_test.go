package base //nolint:testpackage // 测试注入实例私有探测函数，不增加生产测试接口。

import (
	"testing"
	"time"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/utils/networkhealth"
)

func TestNetworkHealthReturnsTargetMetadataFromProbe(t *testing.T) {
	t.Parallel()
	dm := &dice.DiceManager{}
	dm.Dice = &dice.Dice{Parent: dm}
	svc := NewBaseService(dm)
	svc.networkHealth = func() networkhealth.Result {
		return networkhealth.Result{
			Total: 2,
			OK:    []string{"qq"},
			Targets: []networkhealth.TargetResult{
				{Target: "qq", Name: "QQ", Label: "QQ 机器人 API", OK: true, Duration: 42 * time.Millisecond},
				{Target: "slack", Name: "Slack", Label: "Slack 机器人 API", OK: false},
			},
			Timestamp: 1700000000,
		}
	}

	resp, err := svc.NetworkHealth(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	item := resp.Body.Item
	if item.Total != 2 || len(item.Targets) != 2 || len(item.OK) != 1 || item.Timestamp != 1700000000 {
		t.Fatalf("unexpected health summary: %+v", item)
	}
	if item.Targets[0].Target != "qq" || item.Targets[0].Name != "QQ" ||
		item.Targets[0].Label != "QQ 机器人 API" || !item.Targets[0].OK ||
		item.Targets[0].DurationMs != 42 {
		t.Fatalf("unexpected qq target: %+v", item.Targets[0])
	}
	if item.Targets[1].Target != "slack" || item.Targets[1].Name != "Slack" ||
		item.Targets[1].Label != "Slack 机器人 API" || item.Targets[1].OK ||
		item.Targets[1].DurationMs != 0 {
		t.Fatalf("unexpected slack target: %+v", item.Targets[1])
	}
}
