package base_test

import (
	"testing"
	"time"

	base "sealdice-core/api/v2/base"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
)

func TestNetworkHealthReturnsFiveTargetsAndOkList(t *testing.T) {
	t.Cleanup(base.ResetNetworkHealthTestHooks)
	base.SetNetworkHealthTestHooks(
		func(target string, _ []string) (bool, time.Duration) {
			switch target {
			case "seal":
				return true, 42 * time.Millisecond
			case "sign":
				return false, 0
			default:
				return true, 10 * time.Millisecond
			}
		},
		func(_ *dice.Dice) []string {
			return []string{"https://sign.example/ping"}
		},
	)

	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}

	svc := base.NewBaseService(dm)
	resp, err := svc.NetworkHealth(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("NetworkHealth returned error: %v", err)
	}

	item := resp.Body.Item
	if item.Total != 5 {
		t.Fatalf("Total = %d, want 5", item.Total)
	}
	if len(item.Targets) != 5 {
		t.Fatalf("len(Targets) = %d, want 5", len(item.Targets))
	}
	if len(item.OK) != 4 {
		t.Fatalf("len(OK) = %d, want 4", len(item.OK))
	}
	if item.Timestamp <= 0 {
		t.Fatalf("Timestamp = %d, want positive unix timestamp", item.Timestamp)
	}

	var sealTarget *base.NetworkHealthTarget
	var signTarget *base.NetworkHealthTarget
	for i := range item.Targets {
		target := &item.Targets[i]
		switch target.Target {
		case "seal":
			sealTarget = target
		case "sign":
			signTarget = target
		}
	}
	if sealTarget == nil || !sealTarget.OK || sealTarget.DurationMs != 42 {
		t.Fatalf("seal target = %#v, want ok with 42ms", sealTarget)
	}
	if signTarget == nil || signTarget.OK || signTarget.DurationMs != 0 {
		t.Fatalf("sign target = %#v, want failed with 0ms", signTarget)
	}
}

func TestNetworkHealthIncludesFailedSignTargetWhenNoSignServers(t *testing.T) {
	t.Cleanup(base.ResetNetworkHealthTestHooks)
	base.SetNetworkHealthTestHooks(
		func(target string, _ []string) (bool, time.Duration) {
			if target == "sign" {
				t.Fatal("sign connectivity should not be checked without sign urls")
			}
			return true, time.Millisecond
		},
		func(_ *dice.Dice) []string {
			return nil
		},
	)

	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}

	svc := base.NewBaseService(dm)
	resp, err := svc.NetworkHealth(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("NetworkHealth returned error: %v", err)
	}

	for _, target := range resp.Body.Item.Targets {
		if target.Target == "sign" {
			if target.OK {
				t.Fatal("sign target OK = true, want false when there are no sign servers")
			}
			return
		}
	}
	t.Fatal("expected sign target in network health response")
}
