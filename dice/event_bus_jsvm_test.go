//nolint:testpackage
package dice

import (
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

func TestSealBusOnEventJS(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)
	t.Cleanup(func() { _ = d.EventBus.Close(t.Context()) })
	d.ExtLoopManager = NewJsLoopManager()

	loop := eventloop.NewEventLoop()
	version := d.ExtLoopManager.SetLoop(loop)

	got := make(chan string, 1)
	subscribed := make(chan struct{})
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer close(subscribed)
		// 与 dice_jsvm.go 保持一致的字段名映射，jsbind 标签对 JS 可见
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))
		// JS 回调向测试回传结果的辅助函数
		_ = vm.Set("__received", func(s string) { got <- s })
		bus := vm.NewObject()
		_ = registerBusObject(vm, bus, d, version)
		_ = vm.Set("bus", bus)
		_, err := vm.RunString(`
			bus.onEvent("group.muted", function(ctx, ev) {
				__received(ev.name + "|" + ev.groupId + "|" + ev.raw.duration);
			});
		`)
		if err != nil {
			t.Errorf("脚本执行失败: %v", err)
		}
	})
	loop.Start()
	defer loop.StopNoWait()
	<-subscribed // 等注册任务在 loop 上执行完毕，避免发射竞态

	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ"}}
	ctx := &MsgContext{Dice: d, Session: d.ImSession, EndPoint: ep}
	d.ImSession.EmitEvent(&AdapterEvent{
		Name:    EventNameGroupMuted,
		GroupID: "QQ:10001",
		Raw:     map[string]any{"duration": 600},
		Ctx:     ctx,
	})

	select {
	case r := <-got:
		if r != "group.muted|QQ:10001|600" {
			t.Fatalf("JS 收到的事件数据不符: %s", r)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("JS 订阅者未被触发")
	}
}

func TestSealBusSendRawJS(t *testing.T) {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "fakeproto2",
		Platform:     "QQ",
		RawActions: map[string]AdapterRawActionSpec{
			"echo_action": {Name: "echo_action"},
		},
	})
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{
		{
			EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "fakeproto2", Enable: true, State: StateConnected},
			Adapter:          &fakeRawAdapter{},
		},
	}}
	d.EventBus = NewEventBus(d)
	t.Cleanup(func() { _ = d.EventBus.Close(t.Context()) })
	d.ExtLoopManager = NewJsLoopManager()

	loop := eventloop.NewEventLoop()
	version := d.ExtLoopManager.SetLoop(loop)

	errCh := make(chan error, 1)
	done := make(chan struct{})
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer close(done)
		bus := vm.NewObject()
		_ = registerBusObject(vm, bus, d, version)
		_ = vm.Set("bus", bus)
		_, err := vm.RunString(`
			var r = bus.sendRaw("QQ", "echo_action", {user_id: 42});
			if (!r || r.echo !== 42) { throw new Error("sendRaw 返回不符: " + JSON.stringify(r)); }
		`)
		errCh <- err
	})
	loop.Start()
	defer loop.StopNoWait()
	<-done
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("sendRaw JS 调用失败: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sendRaw JS 调用超时")
	}
}
