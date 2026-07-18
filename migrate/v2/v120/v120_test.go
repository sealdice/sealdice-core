package v120_test

import (
	"os"
	"path/filepath"
	"testing"

	v120 "sealdice-core/migrate/v2/v120"
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
	if err := op.Init(t.Context()); err != nil {
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
	if err := v120.V120Migration.Apply(logf.logf, op); err != nil {
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

	// 应走自检分支而非"不存在"分支，并且应输出自检跳过日志
	// 注：日志文案需与 v120.go 中保持一致；该包未为此日志定义常量，故硬编码字面量。
	const selfGuardLog = "[INFO] 新版 attrs 表已存在，data.bdb 数据已迁移，将旧文件重命名为 data.bdb.migrated 作为备份"
	var selfGuardLogFound bool
	for _, l := range logf.lines {
		if l == selfGuardLog {
			selfGuardLogFound = true
		}
		if l == "[INFO] V120升级已经被应用过或版本为新版本，无需应用升级" {
			t.Fatal("不应进入 data.bdb 不存在的分支（attrs 表应触发自检跳过）")
		}
	}
	if !selfGuardLogFound {
		t.Fatalf("未找到自检跳过日志 %q，实际日志: %v", selfGuardLog, logf.lines)
	}
}
