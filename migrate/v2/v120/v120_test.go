package v120

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine/sqlite"
)

type logCollector struct {
	lines []string
}

func (c *logCollector) logf(msg string) { c.lines = append(c.lines, msg) }

func TestV120Migration_SelfGuard_SkipsWhenAttrsExists(t *testing.T) {
	// V120 的 bdbPath 是硬编码相对路径 ./data/default/data.bdb，
	// 需要 chdir 到临时目录，使得路径和 DATADIR 都指向同一个位置。
	tmpDir := t.TempDir()
	defaultDir := filepath.Join(tmpDir, "data", "default")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	t.Chdir(tmpDir)
	t.Setenv("DATADIR", defaultDir)

	op := &sqlite.SQLiteEngine{}
	if err := op.Init(context.Background()); err != nil {
		t.Fatalf("Init SQLiteEngine 失败: %v", err)
	}
	t.Cleanup(op.Close)

	// 在 data.db 里造一个 attrs 表，模拟 V150 已执行过
	dataDB := op.GetDataDB(constant.WRITE)
	if err := dataDB.Exec("CREATE TABLE attrs (id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("建 attrs 表失败: %v", err)
	}

	// 造一个假的 data.bdb 文件（V120 的触发条件）
	bdbPath := "./data/default/data.bdb"
	if err := os.WriteFile(bdbPath, []byte("fake-bolt-data"), 0644); err != nil {
		t.Fatalf("创建 data.bdb 失败: %v", err)
	}

	// 执行 V120（应走自检分支，跳过迁移并重命名）
	logf := &logCollector{}
	if err := V120Migration.Apply(logf.logf, op); err != nil {
		t.Fatalf("V120Migration.Apply 失败: %v", err)
	}

	// data.bdb 应已被重命名
	if _, err := os.Stat(bdbPath); !os.IsNotExist(err) {
		t.Fatal("data.bdb 应已被重命名（不再存在）")
	}
	migratedPath := bdbPath + ".migrated"
	if _, err := os.Stat(migratedPath); err != nil {
		t.Fatalf("data.bdb.migrated 应已存在: %v", err)
	}

	// 应走自检分支而非"不存在"分支
	for _, l := range logf.lines {
		if l == "[INFO] V120升级已经被应用过或版本为新版本，无需应用升级" {
			t.Fatal("不应进入 data.bdb 不存在的分支（attrs 表应触发自检跳过）")
		}
	}
}
