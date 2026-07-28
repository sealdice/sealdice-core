package v2_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	v161 "sealdice-core/migrate/v2/v161"
	"sealdice-core/migrate/v2/v2test"
	"sealdice-core/utils/constant"
	upgrade "sealdice-core/utils/upgrader"
	"sealdice-core/utils/upgrader/store"
)

func TestV161NoticeIDsMigrationRunsOnlyOnce(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	workDir := t.TempDir()
	configDir := filepath.Join(workDir, "data", "default")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "serve.yaml")
	if err := os.WriteFile(configPath, []byte("diceMasters: [QQ:10001]\nnoticeIds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(workDir)
	t.Cleanup(func() { t.Chdir(oldWorkDir) })

	mgr := &upgrade.Manager{
		Store:    store.NewGormStore(op.GetDataDB(constant.WRITE)),
		Database: op,
	}
	mgr.Register(v161.V161NoticeIDsMigration)
	if err = mgr.ApplyAll(); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}

	// 模拟用户在迁移完成后手动清空通知列表。
	if err = os.WriteFile(configPath, []byte("diceMasters: [QQ:10001]\nnoticeIds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err = mgr.ApplyAll(); err != nil {
		t.Fatalf("第二次启动失败: %v", err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		NoticeIDs []string `yaml:"noticeIds"`
	}
	if err = yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.NoticeIDs) != 0 {
		t.Fatalf("迁移完成后用户手动清空的通知列表不应再次填充，实际 %v", got.NoticeIDs)
	}
}
