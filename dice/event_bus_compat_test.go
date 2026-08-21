package dice

import (
	"context"
	"testing"

	"sealdice-core/dice/events"
)

// OnPoke 改造后：事件经总线分发，插件订阅者可收到（含 Raw/Detail 载荷）。
// 兼容订阅器的 legacy 行为由既有回归测试（go test ./dice/...）兜底。
func TestOnPokeEmitsAdapterEvent(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)
	t.Cleanup(func() { _ = d.EventBus.Close(context.Background()) })
	d.registerAdapterEventCompat()

	var gotName, gotGroup string
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		gotName = ev.Name
		gotGroup = ev.GroupID
		return nil
	})

	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ"}}
	d.ImSession.EndPoints = append(d.ImSession.EndPoints, ep)
	ctx := &MsgContext{Dice: d, Session: d.ImSession, EndPoint: ep}

	// 兼容订阅器内 legacyOnPoke 会访问群激活链路，群不存在时直接返回，不应 panic
	d.ImSession.OnPoke(ctx, &events.PokeEvent{GroupID: "", SenderID: "QQ:1", TargetID: "QQ:2"})

	if gotName != EventNamePoke {
		t.Fatalf("插件订阅者未收到 poke 事件: %q", gotName)
	}
	d.ImSession.OnPoke(ctx, &events.PokeEvent{GroupID: "QQ:10001", SenderID: "QQ:1", TargetID: "QQ:2"})
	if gotGroup != "QQ:10001" {
		t.Fatalf("事件载荷 GroupID 不符: %q", gotGroup)
	}
}
