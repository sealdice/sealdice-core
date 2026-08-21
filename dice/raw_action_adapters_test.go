//nolint:testpackage
package dice

import (
	"testing"
)

func TestAdapterCapabilityDeclarations(t *testing.T) {
	ob, ok := GetAdapterCapabilities("onebot")
	if !ok {
		t.Fatal("onebot 能力未注册")
	}
	for _, name := range []string{"send_group_msg", "send_private_msg", "delete_msg", "set_group_ban", "set_group_kick", "get_group_member_info"} {
		if _, declared := ob.RawActions[name]; !declared {
			t.Fatalf("onebot RawActions 缺少 %s", name)
		}
	}
	for _, name := range []string{EventNamePoke, EventNameGroupJoined, EventNameGroupMemberJoined, EventNameGroupLeave, EventNameFriendJoined, EventNameFriendRequest, EventNameGroupRequest} {
		if _, declared := ob.EmitEvents[name]; !declared {
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
	_, err := pa.RawAction(t.Context(), "get_group_member_info", map[string]any{"group_id": 1, "user_id": 2})
	if err == nil {
		t.Fatal("未连接时应报错")
	}
	// 参数类型错误应报错而非 panic
	_, err = pa.RawAction(t.Context(), "get_group_member_info", map[string]any{"group_id": "abc", "user_id": 2})
	if err == nil {
		t.Fatal("非法参数应报错")
	}
}

func TestMilkyExtendedRawActions(t *testing.T) {
	mk, ok := GetAdapterCapabilities("milky")
	if !ok {
		t.Fatal("milky 能力未注册")
	}
	for _, name := range []string{
		"get_login_info", "get_group_list", "get_group_info", "get_group_member_list",
		"set_group_member_mute", "set_group_whole_mute", "set_group_member_card",
		"set_group_member_admin", "kick_group_member", "quit_group",
		"recall_group_message", "recall_private_message", "set_group_essence_message",
		"accept_friend_request", "reject_friend_request",
	} {
		if _, ok := mk.RawActions[name]; !ok {
			t.Fatalf("milky RawActions 缺少 %s", name)
		}
	}
	if _, ok := mk.EmitEvents[EventNameMessageDeleted]; !ok {
		t.Fatal("milky EmitEvents 缺少 message.deleted")
	}
}

func TestOnebotExtendedRawActions(t *testing.T) {
	ob, ok := GetAdapterCapabilities("onebot")
	if !ok {
		t.Fatal("onebot 能力未注册")
	}
	for _, name := range []string{"send_msg", "get_msg", "get_login_info", "set_group_whole_ban", "set_group_leave", "set_friend_add_request", "set_group_add_request", "set_group_card", "set_group_special_title", "set_group_admin", "set_group_essence_msg"} {
		if _, ok := ob.RawActions[name]; !ok {
			t.Fatalf("onebot RawActions 缺少 %s", name)
		}
	}
	if _, ok := ob.EmitEvents[EventNameGroupMuted]; !ok {
		t.Fatal("onebot EmitEvents 缺少 group.muted")
	}
	if _, ok := ob.EmitEvents[EventNameMessageDeleted]; !ok {
		t.Fatal("onebot EmitEvents 缺少 message.deleted")
	}
}

func TestPlatformCapabilityDeclarations(t *testing.T) {
	cases := map[string]string{"tg": "TG", "kook": "KOOK", "official": "QQ"}
	for key, platform := range cases {
		set, ok := GetAdapterCapabilities(key)
		if !ok {
			t.Fatalf("能力 %s 未注册", key)
		}
		if set.Platform != platform {
			t.Fatalf("能力 %s 的平台应为 %s，实为 %s", key, platform, set.Platform)
		}
	}
}
