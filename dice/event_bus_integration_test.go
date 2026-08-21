package dice

import (
	"context"
	"testing"
)

func TestDiceEventBusWiring(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)
	t.Cleanup(func() { _ = d.EventBus.Close(context.Background()) })

	var hits int
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		hits++
		return nil
	})

	d.ImSession.EmitEvent(&AdapterEvent{
		Name:     EventNamePoke,
		Platform: "QQ",
		Ctx:      &MsgContext{Dice: d, Session: d.ImSession, EndPoint: &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ"}}},
	})
	if hits != 1 {
		t.Fatalf("EmitEvent 应触发订阅: %d", hits)
	}
}

func TestIMSessionEmitEventNilSafe(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d}
	d.EventBus = NewEventBus(d)
	t.Cleanup(func() { _ = d.EventBus.Close(context.Background()) })
	// 无 Ctx、无平台字段也不应 panic，且补齐 Platform/EndPointID
	var gotPlatform string
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		gotPlatform = ev.Platform
		return nil
	})
	d.ImSession.EmitEvent(&AdapterEvent{Name: EventNamePoke, Ctx: &MsgContext{Dice: d, Session: d.ImSession, EndPoint: &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "TG"}}}})
	if gotPlatform != "TG" {
		t.Fatalf("应从 Ctx.EndPoint 补齐 Platform: %q", gotPlatform)
	}
	d.ImSession.EmitEvent(nil) // 不应 panic
}
