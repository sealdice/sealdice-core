package sqlite

import (
	"database/sql"
	"fmt"
	"runtime"
	"time"
)

// readMaxOpenConns 返回读连接池的最大连接数。
// 使用 GOMAXPROCS 而非 NumCPU，以在容器中尊重 cgroup CPU 限制；
// 并夹在 [2,4] 之间，避免轻量服务器（1c2g/2c4g）上因宿主机核数过大而占用过多内存。
func readMaxOpenConns() int {
	n := runtime.GOMAXPROCS(0)
	return min(max(n, 2), 4)
}

// readDBDSN 构造读连接的 DSN。读连接不应获取写锁，因此使用 deferred。
func readDBDSN(path string) string {
	return fmt.Sprintf("file:%v?_txlock=deferred", path)
}

// writeDBDSN 构造写连接的 DSN。写连接使用 immediate，避免事务升级时死锁。
func writeDBDSN(path string) string {
	return fmt.Sprintf("file:%v?_txlock=immediate", path)
}

// configureReadPool 配置读连接池，并回收空闲连接以释放各自占用的 page cache。
// MaxIdleConns 与 MaxOpenConns 保持一致，避免并发下频繁关闭/重开连接
// （每个新连接都会执行一次连接级 pragma 钩子，churn 代价较高）。
func configureReadPool(pool *sql.DB) {
	maxConns := readMaxOpenConns()
	pool.SetMaxOpenConns(maxConns)
	pool.SetMaxIdleConns(maxConns)
	pool.SetConnMaxIdleTime(5 * time.Minute)
}

// configureWritePool 配置写连接池：只允许一个写连接。
func configureWritePool(pool *sql.DB) {
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
}
