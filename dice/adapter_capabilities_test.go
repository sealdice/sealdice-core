//nolint:testpackage
package dice

import "testing"

func TestAdapterCapabilitiesRegistry(t *testing.T) {
	set := AdapterCapabilitySet{
		ProtocolType: "fake-caps-test",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNamePoke: {Name: EventNamePoke, Description: "戳一戳"},
		},
		RawActions: map[string]AdapterRawActionSpec{
			"get_group_member_info": {Name: "get_group_member_info", Description: "获取群成员信息"},
		},
	}
	RegisterAdapterCapabilities(set)

	got, ok := GetAdapterCapabilities("fake-caps-test")
	if !ok {
		t.Fatal("按协议类型查询能力失败")
	}
	if _, ok := got.EmitEvents[EventNamePoke]; !ok {
		t.Fatal("EmitEvents 缺少 poke")
	}
	if _, ok := got.RawActions["get_group_member_info"]; !ok {
		t.Fatal("RawActions 缺少 get_group_member_info")
	}

	merged := GetAdapterCapabilitiesByPlatform("QQ")
	if len(merged) == 0 {
		t.Fatal("按平台聚合查询能力失败")
	}

	if _, ok := GetAdapterCapabilities("not-exist"); ok {
		t.Fatal("不存在的协议类型应返回不存在")
	}
}
