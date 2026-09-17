package sqlite

import (
	"database/sql"
	"fmt"
)

// hookedDriverName 是注册了“连接级 pragma”钩子的自定义 SQLite 驱动名。
// 同一个进程内只需要注册一次。
const hookedDriverName = "seal_sqlite"

// defaultPragmas 是所有需要确保生效的 pragma。
// 其中 journal_mode 是持久化在数据库文件里的设置；
// 其余 pragma 是连接级的，只作用于执行它们的那个连接，见 connLocalPragmas。
var defaultPragmas = map[string]string{
	"journal_mode": "wal",   // https://www.sqlite.org/pragma.html#pragma_journal_mode
	"busy_timeout": "15000", // https://www.sqlite.org/pragma.html#pragma_busy_timeout
	// 在 WAL 模式下使用 synchronous=NORMAL 提交的事务可能会在断电或系统崩溃后回滚。
	// 无论同步设置或日志模式如何，事务在应用程序崩溃时都是持久的。
	// 对于在 WAL 模式下运行的大多数应用程序来说，synchronous=NORMAL 设置是一个不错的选择。
	"synchronous": "1",     // NORMAL --> https://www.sqlite.org/pragma.html#pragma_synchronous
	"cache_size":  "-4096", // Negative values are KiB, so this caps the page cache at 4 MiB.
	// 限制 checkpoint 后 WAL 文件回缩到的上限，避免 -wal 无限增长。
	"journal_size_limit": "67108864", // 64 MiB --> https://www.sqlite.org/pragma.html#pragma_journal_size_limit
}

// connLocalPragmas 是连接级 pragma 的名字。它们必须在每个新建连接上重新设置，
// 否则连接池后续新建的连接会退回 SQLite 默认值。
var connLocalPragmas = []string{"busy_timeout", "synchronous", "cache_size", "journal_size_limit"}

// connPragmaStatements 返回需要在每个新建连接上执行的 pragma 语句。
func connPragmaStatements() []string {
	stmts := make([]string, 0, len(connLocalPragmas))
	for _, k := range connLocalPragmas {
		stmts = append(stmts, fmt.Sprintf("pragma %s = %s", k, defaultPragmas[k]))
	}
	return stmts
}

// SetDefaultPragmas defines some sqlite pragmas for good performance and litestream compatibility
// https://highperformancesqlite.com/articles/sqlite-recommended-pragmas
// https://litestream.io/tips/
// copied from https://github.com/bihe/monorepo
// add PRAGMA optimize=0x10002; from https://github.com/Palats/mastopoof
func SetDefaultPragmas(db *sql.DB) error {
	// journal_mode 是持久化设置，只需设置一次；连接级 pragma 由每个连接的钩子负责。
	stmt := fmt.Sprintf("pragma %s = %s", "journal_mode", defaultPragmas["journal_mode"])
	if _, err := db.Exec(stmt); err != nil {
		return err
	}

	// validate the pragmas
	var val string
	for k := range defaultPragmas {
		row := db.QueryRow(fmt.Sprintf("pragma %s", k))
		err := row.Scan(&val)
		if err != nil {
			return err
		}
		if val != defaultPragmas[k] {
			return fmt.Errorf("could not set pragma %s to %s", k, defaultPragmas[k])
		}
	}
	// 这个不能在上面，因为他没有任何返回值
	// Setup some regular optimization according to sqlite doc:
	//  https://www.sqlite.org/lang_analyze.html
	if _, err := db.Exec("PRAGMA optimize=0x10002;"); err != nil {
		return fmt.Errorf("unable set optimize pragma: %w", err)
	}

	return nil
}
