package dice //nolint:testpackage

import (
	"testing"

	"sealdice-core/dice/events"
)

func TestIMSession_OnPoke_DoesNotPanicWhenGroupNilButGroupIDKnown(t *testing.T) {
	s := &IMSession{ServiceAtNew: new(SyncMap[string, *GroupInfo])}
	s.ServiceAtNew.Store("QQ-Group:1", &GroupInfo{GroupID: "QQ-Group:1"})

	ctx := &MsgContext{
		MessageType: "group",
		Session:     s,
		EndPoint:    &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:bot"}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic, got: %v", r)
		}
	}()

	s.OnPoke(ctx, &events.PokeEvent{GroupID: "QQ-Group:1"})
}

func TestIMSession_OnPoke_DoesNotPanicOnPrivatePokeWithoutGroup(t *testing.T) {
	s := &IMSession{ServiceAtNew: new(SyncMap[string, *GroupInfo])}
	ctx := &MsgContext{
		MessageType: "private",
		Session:     s,
		EndPoint:    &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:bot"}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic, got: %v", r)
		}
	}()

	s.OnPoke(ctx, &events.PokeEvent{IsPrivate: true})
}
