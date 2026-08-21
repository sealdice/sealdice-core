package dice

import (
	"context"
	"testing"
)

func TestNotifyOnlyEventEmission(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)
	t.Cleanup(func() { _ = d.EventBus.Close(context.Background()) })

	var names []string
	_ = d.EventBus.OnEvent("*", func(ctx context.Context, ev *AdapterEvent) error {
		names = append(names, ev.Name)
		return nil
	})

	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "milky"}}
	d.ImSession.EndPoints = append(d.ImSession.EndPoints, ep)
	ctx := &MsgContext{Dice: d, Session: d.ImSession, EndPoint: ep}

	EmitFriendRequest(ctx, "QQ:10086", "你好")
	EmitGroupRequest(ctx, "QQ:10001", "QQ:10086", "invite")
	EmitFriendJoined(ctx, &Message{Platform: "QQ", Sender: SenderBase{UserID: "QQ:10086"}})
	EmitGuildJoined(ctx, &Message{Platform: "KOOK", GroupID: "KOOK:1", Sender: SenderBase{UserID: "KOOK:2"}})

	want := []string{EventNameFriendRequest, EventNameGroupRequest, EventNameFriendJoined, EventNameGuildJoined}
	if len(names) != len(want) {
		t.Fatalf("发射数量不符: %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("第 %d 个事件应为 %s，实为 %s", i, want[i], names[i])
		}
	}
}
