package js_test

import (
	"path/filepath"
	"testing"

	. "sealdice-core/api/v2/js"
	"sealdice-core/dice"
	cmn "sealdice-core/model/common/request"
)

func newDeadConfigService(t *testing.T) (*dice.DiceManager, *dice.Dice, *Service, string) {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "plugins.json")
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}
	d.ConfigManager = dice.NewConfigManager(filename)
	d.Config.JsEnable = true
	d.JsExtRegistry = new(dice.SyncMap[string, *dice.ExtInfo])
	return dm, d, NewService(dm), filename
}

func registerTestPlugin(t *testing.T, d *dice.Dice, name string) {
	t.Helper()
	item := d.ConfigManager.NewConfigItem("k", "v", "desc")
	item.Type = "string"
	d.ConfigManager.RegisterPluginConfig(name, item)
}

// 已注册扩展的插件（存活）不应被列为死配置，只有未注册的才是死配置。
func TestGetDeadConfigsOnlyListsUnregisteredPlugins(t *testing.T) {
	_, d, svc, _ := newDeadConfigService(t)
	registerTestPlugin(t, d, "alive")
	registerTestPlugin(t, d, "dead")
	d.JsExtRegistry.Store("alive", &dice.ExtInfo{Name: "alive"})

	resp, err := svc.GetDeadConfigs(t.Context(), &cmn.Empty{})
	if err != nil {
		t.Fatalf("GetDeadConfigs returned error: %v", err)
	}
	configs := resp.Body.Item.Configs
	if len(configs) != 1 || configs[0].Name != "dead" {
		t.Fatalf("expected only 'dead' listed, got %+v", configs)
	}
}

// 活插件配置不可被删除；死插件整体删除并持久化。
func TestDeleteDeadConfigsRemovesOnlyDeadPlugins(t *testing.T) {
	_, d, svc, filename := newDeadConfigService(t)
	registerTestPlugin(t, d, "alive")
	registerTestPlugin(t, d, "dead")
	d.JsExtRegistry.Store("alive", &dice.ExtInfo{Name: "alive"})

	if _, err := svc.DeleteDeadConfigs(t.Context(), &DeleteDeadConfigsReq{
		Body: JsDeleteDeadConfigsReqBody{Names: []string{"alive"}},
	}); err != nil {
		t.Fatalf("DeleteDeadConfigs returned error: %v", err)
	}
	if _, ok := d.ConfigManager.Plugins["alive"]; !ok {
		t.Fatal("expected alive plugin kept")
	}

	if _, err := svc.DeleteDeadConfigs(t.Context(), &DeleteDeadConfigsReq{
		Body: JsDeleteDeadConfigsReqBody{Names: []string{"dead"}},
	}); err != nil {
		t.Fatalf("DeleteDeadConfigs returned error: %v", err)
	}
	if _, ok := d.ConfigManager.Plugins["dead"]; ok {
		t.Fatal("expected dead plugin removed")
	}

	reloaded := dice.NewConfigManager(filename)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if _, ok := reloaded.Plugins["dead"]; ok {
		t.Fatal("expected dead plugin gone after reload")
	}
	if _, ok := reloaded.Plugins["alive"]; !ok {
		t.Fatal("expected alive plugin after reload")
	}
}

// JS 环境未初始化时无法确认插件死活：不应列出死配置，也不应允许删除。
func TestDeadConfigsRequireInitializedJsRegistry(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "plugins.json")
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}
	d.ConfigManager = dice.NewConfigManager(filename)
	d.Config.JsEnable = true
	registerTestPlugin(t, d, "anything")
	svc := NewService(dm)

	resp, err := svc.GetDeadConfigs(t.Context(), &cmn.Empty{})
	if err != nil {
		t.Fatalf("GetDeadConfigs returned error: %v", err)
	}
	if len(resp.Body.Item.Configs) != 0 {
		t.Fatalf("expected no dead configs when registry uninitialized, got %+v", resp.Body.Item.Configs)
	}

	if _, err := svc.DeleteDeadConfigs(t.Context(), &DeleteDeadConfigsReq{
		Body: JsDeleteDeadConfigsReqBody{Names: []string{"anything"}},
	}); err == nil {
		t.Fatal("expected error when registry uninitialized")
	}
	if _, ok := d.ConfigManager.Plugins["anything"]; !ok {
		t.Fatal("expected plugin config kept when delete rejected")
	}
}

func TestDeadConfigsRequireStableJsLifecycle(t *testing.T) {
	states := []struct {
		name string
		set  func(*dice.Dice)
	}{
		{name: "js disabled", set: func(d *dice.Dice) { d.Config.JsEnable = false }},
		{name: "reload in progress", set: func(d *dice.Dice) { d.JsReloading = true }},
		{name: "disabled script", set: func(d *dice.Dice) {
			d.JsScriptList = []*dice.JsScriptInfo{{Name: "broken-or-disabled", Enable: false}}
		}},
	}
	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			_, d, svc, _ := newDeadConfigService(t)
			registerTestPlugin(t, d, "dead")
			state.set(d)

			resp, err := svc.GetDeadConfigs(t.Context(), &cmn.Empty{})
			if err != nil {
				t.Fatalf("GetDeadConfigs returned error: %v", err)
			}
			if len(resp.Body.Item.Configs) != 0 {
				t.Fatalf("expected no dead configs during %s, got %+v", state.name, resp.Body.Item.Configs)
			}
			if _, err := svc.DeleteDeadConfigs(t.Context(), &DeleteDeadConfigsReq{
				Body: JsDeleteDeadConfigsReqBody{Names: []string{"dead"}},
			}); err == nil {
				t.Fatalf("expected delete rejection during %s", state.name)
			}
			if _, ok := d.ConfigManager.Plugins["dead"]; !ok {
				t.Fatalf("expected config kept during %s", state.name)
			}
		})
	}
}
