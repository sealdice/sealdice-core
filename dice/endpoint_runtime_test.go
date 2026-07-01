//nolint:testpackage
package dice

import (
	"testing"
)

func TestCreateTempCtxResolvesLiveEndpointByID(t *testing.T) {
	d, oldEp, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	oldEp.ProtocolType = "pureonebot"
	oldEp.State = StateDisconnected
	oldEp.Enable = false

	newEp := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			ID:           oldEp.ID,
			UserID:       oldEp.UserID,
			Nickname:     "NewBot",
			Platform:     oldEp.Platform,
			ProtocolType: oldEp.ProtocolType,
			Enable:       true,
			State:        StateConnected,
		},
		Adapter: NewOnebotConnItem(AddOnebotEcho{
			Token:      "test-token",
			ConnectURL: "ws://127.0.0.1:8080",
			Mode:       "client",
		}).Adapter,
	}
	newEp.BindRuntime(d.ImSession)
	d.ImSession.EndPoints = []*EndPointInfo{newEp}

	msg := newGroupMsg("QQ-Group:2233", "QQ:4455", ".r 1")
	ctx := CreateTempCtx(oldEp, msg)

	if ctx.EndPoint != newEp {
		t.Fatalf("expected CreateTempCtx to resolve live endpoint %p, got %p", newEp, ctx.EndPoint)
	}
	if ctx.Session != d.ImSession {
		t.Fatalf("expected ctx session to be current imSession, got %#v", ctx.Session)
	}
	if ctx.Dice != d {
		t.Fatalf("expected ctx dice to be current dice, got %#v", ctx.Dice)
	}
}

func TestCreateTempCtxResolvesLiveEndpointByUserIDFallback(t *testing.T) {
	d, oldEp, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	oldEp.ID = "old-id"
	oldEp.ProtocolType = "pureonebot"
	oldEp.State = StateDisconnected
	oldEp.Enable = false

	newEp := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			ID:           "new-id",
			UserID:       oldEp.UserID,
			Nickname:     "NewBot",
			Platform:     oldEp.Platform,
			ProtocolType: oldEp.ProtocolType,
			Enable:       true,
			State:        StateConnected,
		},
		Adapter: NewOnebotConnItem(AddOnebotEcho{
			Token:      "test-token",
			ConnectURL: "ws://127.0.0.1:8080",
			Mode:       "client",
		}).Adapter,
	}
	newEp.BindRuntime(d.ImSession)
	d.ImSession.EndPoints = []*EndPointInfo{newEp}

	msg := newGroupMsg("QQ-Group:2233", "QQ:4455", ".r 1")
	ctx := CreateTempCtx(oldEp, msg)

	if ctx.EndPoint != newEp {
		t.Fatalf("expected CreateTempCtx fallback to resolve live endpoint %p, got %p", newEp, ctx.EndPoint)
	}
}

func TestBindRuntimeSetsSessionOnPureOnebotEndpoint(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep := NewOnebotConnItem(AddOnebotEcho{
		Token:      "test-token",
		ConnectURL: "ws://127.0.0.1:8080",
		Mode:       "client",
	})

	if ep.Session != nil {
		t.Fatalf("expected new endpoint session to be nil before binding, got %#v", ep.Session)
	}

	ep.BindRuntime(d.ImSession)

	if ep.Session != d.ImSession {
		t.Fatalf("expected endpoint session to be bound to imSession, got %#v", ep.Session)
	}

	pa, ok := ep.Adapter.(*PlatformAdapterOnebot)
	if !ok {
		t.Fatalf("expected pureonebot adapter, got %T", ep.Adapter)
	}
	if pa.EndPoint != ep {
		t.Fatalf("expected adapter endpoint back-reference to be rebound")
	}
}
