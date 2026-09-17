package dice

import (
	"context"

	"sealdice-core/dice/events"
)

// registerAdapterEventCompat 将旧 IMSession.OnXxx 通知逻辑注册为总线订阅器。
// 在 Dice.Init 中最先注册（早于任何 JS 插件订阅），因此旧逻辑先于插件回调执行，
// 与旧实现的顺序语义一致。
func (d *Dice) registerAdapterEventCompat() {
	if d.EventBus == nil {
		return
	}
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		poke, _ := ev.Detail.(*events.PokeEvent)
		if poke == nil {
			return nil
		}
		d.ImSession.legacyOnPoke(ev.Ctx, poke)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameGroupLeave, func(ctx context.Context, ev *AdapterEvent) error {
		leave, _ := ev.Detail.(*events.GroupLeaveEvent)
		if leave == nil {
			return nil
		}
		d.ImSession.legacyOnGroupLeave(ev.Ctx, leave)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameGroupJoined, func(ctx context.Context, ev *AdapterEvent) error {
		msg, _ := ev.Detail.(*Message)
		if msg == nil {
			return nil
		}
		d.ImSession.legacyOnGroupJoined(ev.Ctx, msg)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameGroupMemberJoined, func(ctx context.Context, ev *AdapterEvent) error {
		msg, _ := ev.Detail.(*Message)
		if msg == nil {
			return nil
		}
		d.ImSession.legacyOnGroupMemberJoined(ev.Ctx, msg)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameMessageDeleted, func(ctx context.Context, ev *AdapterEvent) error {
		msg, _ := ev.Detail.(*Message)
		if msg == nil {
			return nil
		}
		d.ImSession.legacyOnMessageDeleted(ev.Ctx, msg)
		return nil
	})
}
