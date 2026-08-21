package dice

import (
	"context"
	"testing"
)

func TestAdapterCapabilityDeclarations(t *testing.T) {
	ob, ok := GetAdapterCapabilities("onebot")
	if !ok {
		t.Fatal("onebot 能力未注册")
	}
	for _, name := range []string{"send_group_msg", "send_private_msg", "delete_msg", "set_group_ban", "set_group_kick", "get_group_member_info"} {
		if _, ok := ob.RawActions[name]; !ok {
			t.Fatalf("onebot RawActions 缺少 %s", name)
		}
	}
	for _, name := range []string{EventNamePoke, EventNameGroupJoined, EventNameGroupMemberJoined, EventNameGroupLeave, EventNameFriendJoined, EventNameFriendRequest, EventNameGroupRequest} {
		if _, ok := ob.EmitEvents[name]; !ok {
			t.Fatalf("onebot EmitEvents 缺少 %s", name)
		}
	}

	mk, ok := GetAdapterCapabilities("milky")
	if !ok {
		t.Fatal("milky 能力未注册")
	}
	for _, name := range []string{"get_group_member_info", "send_group_nudge"} {
		if _, ok := mk.RawActions[name]; !ok {
			t.Fatalf("milky RawActions 缺少 %s", name)
		}
	}
}

func TestMilkyRawActionParamValidation(t *testing.T) {
	pa := &PlatformAdapterMilky{}
	// 未连接（IntentSession 为 nil）时应返回错误而非 panic
	_, err := pa.RawAction(context.Background(), "get_group_member_info", map[string]any{"group_id": 1, "user_id": 2})
	if err == nil {
		t.Fatal("未连接时应报错")
	}
	// 参数类型错误应报错而非 panic
	_, err = pa.RawAction(context.Background(), "get_group_member_info", map[string]any{"group_id": "abc", "user_id": 2})
	if err == nil {
		t.Fatal("非法参数应报错")
	}
}
