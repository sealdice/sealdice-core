package dice //nolint:testpackage

import (
	"path/filepath"
	"testing"
)

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
