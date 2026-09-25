//nolint:testpackage
package dice

import (
	"context"
	"testing"
)

type fakeRawAdapter struct {
	PlatformAdapter // 嵌入接口零值即可满足类型断言
}

func (f *fakeRawAdapter) RawAction(ctx context.Context, action string, params map[string]any) (any, error) {
	return map[string]any{"echo": params["user_id"]}, nil
}

type fakeNoRawAdapter struct {
	PlatformAdapter
}

func TestDiceSendRaw(t *testing.T) {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "fakeproto",
		Platform:     "QQ",
		RawActions: map[string]AdapterRawActionSpec{
			"get_group_member_info": {Name: "get_group_member_info"},
		},
	})

	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{
		{
			EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "fakeproto", Enable: true, State: StateConnected},
			Adapter:          &fakeRawAdapter{},
		},
	}}

	ret, err := d.SendRaw("QQ", "get_group_member_info", map[string]any{"user_id": 123})
	if err != nil {
		t.Fatalf("SendRaw 失败: %v", err)
	}
	m, _ := ret.(map[string]any)
	if m["echo"] != float64(123) && m["echo"] != 123 {
		t.Fatalf("返回值不符: %v", ret)
	}

	// 不在能力清单中的动作必须报错
	if _, err := d.SendRaw("QQ", "no_such_action", nil); err == nil {
		t.Fatal("未声明的动作应报错")
	}
	// 无匹配端点
	if _, err := d.SendRaw("TG", "get_group_member_info", nil); err == nil {
		t.Fatal("无匹配端点应报错")
	}
	// 端点未实现 RawActionAdapter
	d2 := &Dice{}
	d2.ImSession = &IMSession{Parent: d2, EndPoints: []*EndPointInfo{
		{
			EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "fakeproto", Enable: true, State: StateConnected},
			Adapter:          fakeNoRawAdapter{},
		},
	}}
	if _, err := d2.SendRaw("QQ", "get_group_member_info", nil); err == nil {
		t.Fatal("未实现 RawAction 的适配器应报错")
	}
}
