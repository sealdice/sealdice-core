package base //nolint:testpackage // 测试注入实例私有探测函数，不增加生产测试接口。

import (
	"testing"
	"time"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
)

func TestNetworkHealthUsesInstanceProbes(t *testing.T) {
	t.Parallel()
	for _, withSign := range []bool{true, false} {
		name := "without-sign-server"
		if withSign {
			name = "with-sign-server"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dm := &dice.DiceManager{}
			dm.Dice = []*dice.Dice{{Parent: dm}}
			svc := NewBaseService(dm)
			svc.signURLs = func(_ *dice.Dice) []string {
				if withSign {
					return []string{"https://sign.example/ping"}
				}
				return nil
			}
			svc.checkNetwork = func(target string, _ []string) (bool, time.Duration) {
				if target == "sign" {
					if !withSign {
						t.Error("sign probe called without URLs")
					}
					return false, 0
				}
				if target == "seal" {
					return true, 42 * time.Millisecond
				}
				return true, 10 * time.Millisecond
			}
			resp, err := svc.NetworkHealth(t.Context(), &request.Empty{})
			if err != nil {
				t.Fatal(err)
			}
			item := resp.Body.Item
			if item.Total != 5 || len(item.Targets) != 5 || len(item.OK) != 4 || item.Timestamp <= 0 {
				t.Fatalf("unexpected health summary: %+v", item)
			}
			seen := map[string]bool{}
			for _, target := range item.Targets {
				seen[target.Target] = true
				if target.Target == "seal" && (!target.OK || target.DurationMs != 42) {
					t.Fatalf("unexpected seal target: %+v", target)
				}
				if target.Target == "sign" && (target.OK || target.DurationMs != 0) {
					t.Fatalf("unexpected sign target: %+v", target)
				}
			}
			if !seen["seal"] || !seen["sign"] {
				t.Fatalf("missing targets: %+v", item.Targets)
			}
		})
	}
}
