package dice

import (
	"errors"

	"github.com/dop251/goja"
)

var errBusNotReady = errors.New("事件总线未初始化")

// registerBusObject 组装 seal.bus 对象。
// versionID 为当前 JS 事件循环版本号（调用方取自 dice_jsvm.go 中既有 versionID 变量）。
func registerBusObject(vm *goja.Runtime, bus *goja.Object, d *Dice, versionID int64) error {
	if err := bus.Set("onEvent", func(name string, handler func(ctx *MsgContext, ev *AdapterEvent)) error {
		if d.EventBus == nil {
			return errBusNotReady
		}
		return d.EventBus.OnEventJS(name, versionID, handler)
	}); err != nil {
		return err
	}
	if err := bus.Set("getCapabilities", func(platform string) []AdapterCapabilitySet {
		return GetAdapterCapabilitiesByPlatform(platform)
	}); err != nil {
		return err
	}
	if err := bus.Set("sendRaw", func(platform string, action string, params map[string]any) (any, error) {
		return d.SendRaw(platform, action, params)
	}); err != nil {
		return err
	}
	return nil
}
