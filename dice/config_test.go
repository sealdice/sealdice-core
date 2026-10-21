package dice //nolint:testpackage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManagerSetConfigsRejectsInvalidBatchWithoutChanges(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	task := cm.NewConfigItem("schedule", "08:00", "")
	task.Type = "task:daily"
	cm.RegisterPluginConfig("p", strItem(cm, "name", "old", ""), task)
	before, err := os.ReadFile(cm.filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []interface{}{123, "25:00", "not a time"} {
		err = cm.SetConfigs("p", map[string]interface{}{"name": "new", "schedule": invalid})
		if !errors.Is(err, ErrInvalidPluginConfig) {
			t.Fatalf("expected validation error for %v, got %v", invalid, err)
		}
		if cm.Plugins["p"].Configs["name"].Value != "old" || task.Value != "08:00" {
			t.Fatal("invalid batch changed in-memory values")
		}
		after, readErr := os.ReadFile(cm.filename)
		if readErr != nil || !bytes.Equal(before, after) {
			t.Fatalf("invalid batch changed persisted data: %v", readErr)
		}
	}
	if err := cm.SetConfigs("p", map[string]interface{}{"unknown": true}); !errors.Is(err, ErrInvalidPluginConfig) {
		t.Fatalf("unknown key accepted: %v", err)
	}
	if err := cm.SetConfigs("missing", map[string]interface{}{"name": "new"}); !errors.Is(err, ErrInvalidPluginConfig) {
		t.Fatalf("unknown plugin accepted: %v", err)
	}
}

func TestConfigManagerBatchSaveAndResetPersistTogether(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	task := cm.NewConfigItem("schedule", "08:00", "")
	task.Type = "task:daily"
	task.task = &JsScriptTask{taskType: "daily", rawValue: "08:00", cronExpr: "0 8 * * *"}
	cm.RegisterPluginConfig("p", strItem(cm, "name", "old", ""), task)
	if err := cm.SetConfigs("p", map[string]interface{}{"name": "new", "schedule": "09:30"}); err != nil {
		t.Fatal(err)
	}
	reloaded := NewConfigManager(cm.filename)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if reloaded.Plugins["p"].Configs["name"].Value != "new" || reloaded.Plugins["p"].Configs["schedule"].Value != "09:30" {
		t.Fatal("batch was not persisted")
	}
	if task.task.cronExpr != "30 9 * * *" {
		t.Fatal("persisted schedule was not applied")
	}
	if err := cm.ResetConfigs("p", []string{"name", "schedule"}); err != nil {
		t.Fatal(err)
	}
	if task.Value != "08:00" || task.task.cronExpr != "0 8 * * *" || cm.Plugins["p"].Configs["name"].Value != "old" {
		t.Fatal("reset did not restore defaults")
	}
	if err := reloaded.Load(); err != nil || reloaded.Plugins["p"].Configs["name"].Value != "old" {
		t.Fatalf("reset not persisted: %v", err)
	}
}

func TestConfigManagerBatchFailureKeepsValuesAndSchedule(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	task := cm.NewConfigItem("schedule", "08:00", "")
	task.Type = "task:daily"
	task.Value = "09:30"
	task.task = &JsScriptTask{taskType: "daily", rawValue: "09:30", cronExpr: "30 9 * * *"}
	cm.RegisterPluginConfig("p", strItem(cm, "name", "old", ""), task)
	cm.filename = t.TempDir() // A directory cannot be replaced by the configuration file.
	for _, operation := range []func() error{
		func() error { return cm.SetConfigs("p", map[string]interface{}{"name": "new", "schedule": "10:45"}) },
		func() error { return cm.ResetConfigs("p", []string{"name", "schedule"}) },
	} {
		if err := operation(); err == nil {
			t.Fatal("expected persistence failure")
		}
		if cm.Plugins["p"].Configs["name"].Value != "old" || task.Value != "09:30" || task.task.cronExpr != "30 9 * * *" {
			t.Fatal("failed persistence changed values or running schedule")
		}
	}
}

func TestConfigManagerSerializationFailureKeepsPreviousFile(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	cm.RegisterPluginConfig("p", strItem(cm, "name", "old", ""))
	before, err := os.ReadFile(cm.filename)
	if err != nil {
		t.Fatal(err)
	}
	if saveErr := cm.SetConfigs("p", map[string]interface{}{"name": make(chan int)}); saveErr == nil {
		t.Fatal("expected unsupported JSON value to fail")
	}
	after, err := os.ReadFile(cm.filename)
	if err != nil || !bytes.Equal(before, after) || cm.Plugins["p"].Configs["name"].Value != "old" {
		t.Fatalf("serialization failure damaged saved configuration: %v", err)
	}
}

func strItem(cm *ConfigManager, key, value, desc string) *ConfigItem {
	item := cm.NewConfigItem(key, value, desc)
	item.Type = "string"
	return item
}

func TestConfigManagerRemovePluginConfig(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	cm.RegisterPluginConfig(
		"dead",
		strItem(cm, "k1", "v1", "desc"),
		strItem(cm, "k2", "v2", "desc"),
	)
	cm.RegisterPluginConfig("alive", strItem(cm, "k", "v", "desc"))
	if len(cm.Plugins) != 2 {
		t.Fatalf("expected 2 plugins registered, got %d", len(cm.Plugins))
	}

	// 整体删除死配置
	if err := cm.RemovePluginConfig("dead"); err != nil {
		t.Fatalf("remove dead plugin failed: %v", err)
	}
	if len(cm.Plugins) != 1 {
		t.Fatalf("expected dead plugin removed, got %d entries", len(cm.Plugins))
	}
	if _, ok := cm.Plugins["alive"]; !ok {
		t.Fatalf("expected alive plugin kept")
	}

	// 重复删除与删除不存在的插件都应是安全的空操作
	if err := cm.RemovePluginConfig("dead"); err != nil {
		t.Fatalf("repeated removal failed: %v", err)
	}
	if err := cm.RemovePluginConfig("ghost"); err != nil {
		t.Fatalf("missing plugin removal failed: %v", err)
	}
	if len(cm.Plugins) != 1 {
		t.Fatalf("expected 1 entry after no-op removals, got %d", len(cm.Plugins))
	}

	// 删除应持久化到文件，重新加载后不再出现
	reloaded := NewConfigManager(cm.filename)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if _, ok := reloaded.Plugins["dead"]; ok {
		t.Fatalf("expected dead plugin gone after reload")
	}
	if _, ok := reloaded.Plugins["alive"]; !ok {
		t.Fatalf("expected alive plugin after reload")
	}
}

// UnregisterConfig 只按 keys 删除指定配置项，不传 keys 时不应删掉任何内容；
// 删除整个插件的配置应使用 RemovePluginConfig。这里是行为回归保护。
func TestConfigManagerRemovePluginConfigRestoresOnSaveFailure(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	cm.RegisterPluginConfig("dead", strItem(cm, "k", "v", "desc"))
	cm.filename = t.TempDir()

	if err := cm.RemovePluginConfig("dead"); err == nil {
		t.Fatal("expected save error")
	}
	if _, ok := cm.Plugins["dead"]; !ok {
		t.Fatal("expected plugin restored after save failure")
	}
}

func TestConfigManagerUnregisterConfigWithoutKeysKeepsPlugin(t *testing.T) {
	cm := NewConfigManager(filepath.Join(t.TempDir(), "plugins.json"))
	cm.RegisterPluginConfig("p", strItem(cm, "k", "v", "desc"))

	cm.UnregisterConfig("p")
	if len(cm.Plugins) != 1 {
		t.Fatalf("expected plugin kept when no keys given, got %d entries", len(cm.Plugins))
	}

	cm.UnregisterConfig("p", "k")
	if len(cm.Plugins) != 0 {
		t.Fatalf("expected plugin removed after last key deleted, got %d entries", len(cm.Plugins))
	}
}
