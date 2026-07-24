// v2test 包含 migrate/v2 测试的共享工具函数，供各 *_test 包（v2_test、v120_test 等）使用。
package v2test

import (
	"os"
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
	"sealdice-core/utils/constant"
	engine "sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/engine/sqlite"
	upgrade "sealdice-core/utils/upgrader"
	"sealdice-core/utils/upgrader/store"
)

// NewTestSQLiteEngine 在临时目录中创建一个可用的 SQLite DatabaseOperator，
// 返回 operator 及其数据目录。连接会在测试结束时自动关闭。
func NewTestSQLiteEngine(t *testing.T) (engine.DatabaseOperator, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	op := &sqlite.SQLiteEngine{}
	if err := op.Init(t.Context()); err != nil {
		t.Fatalf("初始化 SQLiteEngine 失败: %v", err)
	}
	t.Cleanup(op.Close)
	return op, op.DataDir
}

// ExecSQLFile 将一个 .sql 文件按分号拆分后逐条在给定 *gorm.DB 上执行，
// 用于把 testdata 下的库结构/数据"灌"进测试库。
func ExecSQLFile(t *testing.T, db *gorm.DB, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 SQL 文件 %s 失败: %v", path, err)
	}
	cleaned := StripSQLLineComments(string(raw))
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

// StripSQLLineComments 去掉 SQL 中以 "--" 开头的行注释。
func StripSQLLineComments(s string) string {
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

// NewTestManager 复刻 enter.go 的注册顺序，升级记录存入 data.db（GormStore）。
func NewTestManager(t *testing.T, op engine.DatabaseOperator) *upgrade.Manager {
	t.Helper()
	storer := store.NewGormStore(op.GetDataDB(constant.WRITE))
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

// SilentLogf 是一个什么都不做的 logf，用于直接调用底层迁移函数时占位。
func SilentLogf(string) {}

// MustExec 直接执行一条 SQL，失败即 fatal。
func MustExec(t *testing.T, db *gorm.DB, sql string, args ...interface{}) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("执行 SQL 失败: %v\nSQL: %s", err, sql)
	}
}

// ScanLogSizes 扫描 logs 表返回 id -> size 的映射。
func ScanLogSizes(t *testing.T, db *gorm.DB) map[int]int {
	t.Helper()
	type row struct {
		ID   int  `gorm:"column:id"`
		Size *int `gorm:"column:size"`
	}
	var rows []row
	if err := db.Raw("SELECT id, size FROM logs ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("查询 logs.size 失败: %v", err)
	}
	m := make(map[int]int, len(rows))
	for _, r := range rows {
		v := 0
		if r.Size != nil {
			v = *r.Size
		}
		m[r.ID] = v
	}
	return m
}

// LogCollector 收集所有日志，便于断言迁移过程中的提示。
type LogCollector struct {
	Lines []string
}

func (c *LogCollector) Logf(msg string) { c.Lines = append(c.Lines, msg) }
