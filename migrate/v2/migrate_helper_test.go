package v2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"

	v120 "sealdice-core/migrate/v2/v120"
	v131 "sealdice-core/migrate/v2/v131"
	v141 "sealdice-core/migrate/v2/v141"
	v144 "sealdice-core/migrate/v2/v144"
	v150 "sealdice-core/migrate/v2/v150"
	v151 "sealdice-core/migrate/v2/v151"
	v160 "sealdice-core/migrate/v2/v160"
	engine "sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/engine/sqlite"
	upgrade "sealdice-core/utils/upgrader"
	"sealdice-core/utils/upgrader/store"
)

// newTestSQLiteEngine 在临时目录中创建一个可用的 SQLite DatabaseOperator，
// 返回 operator 及其数据目录（数据目录可用于放置 upgrade_metadata.json）。
// 连接会在测试结束时自动关闭。
func newTestSQLiteEngine(t *testing.T) (engine.DatabaseOperator, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	op := &sqlite.SQLiteEngine{}
	if err := op.Init(context.Background()); err != nil {
		t.Fatalf("初始化 SQLiteEngine 失败: %v", err)
	}
	t.Cleanup(op.Close)
	return op, op.DataDir
}

// execSQLFile 将一个 .sql 文件按分号拆分后逐条在给定 *gorm.DB 上执行，
// 用于把 testdata 下的库结构/数据“灌”进测试库。
func execSQLFile(t *testing.T, db *gorm.DB, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 SQL 文件 %s 失败: %v", path, err)
	}
	// 去掉行注释，再按 ";" 切分。测试用 fixture 均不含带分号的字符串字面量。
	cleaned := stripSQLLineComments(string(raw))
	for _, stmt := range strings.Split(cleaned, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("执行 SQL 语句失败 (%s): %v\n语句: %s", path, err, stmt)
		}
	}
}

func stripSQLLineComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// newTestManager 复刻 enter.go 的注册顺序，但 Store 落在临时目录里，便于测试隔离。
func newTestManager(t *testing.T, op engine.DatabaseOperator, dataDir string) *upgrade.Manager {
	t.Helper()
	storer := store.NewJSONStore(filepath.Join(dataDir, "upgrade_metadata.json"))
	mgr := &upgrade.Manager{Store: storer, Database: op}
	mgr.Register(v120.V120Migration)
	mgr.Register(v120.V120LogMessageMigration)
	mgr.Register(v131.V131ConfigUpdateMigration)
	mgr.Register(v141.V141ConfigUpdateMigration)
	mgr.Register(v144.V144RemoveOldHelpDocMigration)
	mgr.Register(v150.V150UpgradeAttrsMigration)
	mgr.Register(v150.V150FixGroupInfoMigration)
	mgr.Register(v151.V151GORMCleanMigration)
	mgr.Register(v160.V160LogIDZeroCleanMigration)
	mgr.Register(v160.V160LogRawMsgIDIndexMigration)
	mgr.Register(v160.V160LogSizeRepairMigration)
	return mgr
}

// silentLogf 是一个什么都不做的 logf，用于直接调用底层迁移函数时占位。
func silentLogf(string) {}

// logCollector 收集所有日志，便于断言迁移过程中的提示。
type logCollector struct {
	lines []string
}

func (c *logCollector) logf(msg string) { c.lines = append(c.lines, msg) }

// mustExec 直接执行一条 SQL，失败即 fatal。
func mustExec(t *testing.T, db *gorm.DB, sql string, args ...interface{}) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("执行 SQL 失败: %v\nSQL: %s", err, sql)
	}
}
