package sqlite //nolint:testpackage // 需要访问未导出的内部结构

import (
	"os"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
)

func TestInitCreatesPluginDatabase(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	engine := &SQLiteEngine{}
	if err := engine.Init(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)

	if _, err := os.Stat(filepath.Join(dataDir, "data_plugins.db")); err != nil {
		t.Fatalf("data_plugins.db was not created: %v", err)
	}
	if engine.GetPluginDB(constant.READ) == nil || engine.GetPluginDB(constant.WRITE) == nil {
		t.Fatal("plugin database handles were not initialized")
	}
}

func TestInitDoesNotInstallGlobalQueryCache(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	engine := &SQLiteEngine{}
	if err := engine.Init(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)

	for _, mode := range []constant.DBMode{constant.READ, constant.WRITE} {
		for name, db := range map[string]*gorm.DB{
			"data":    engine.GetDataDB(mode),
			"logs":    engine.GetLogDB(mode),
			"censor":  engine.GetCensorDB(mode),
			"plugins": engine.GetPluginDB(mode),
		} {
			if _, exists := db.Plugins["gorm:caches"]; exists {
				t.Errorf("%s %v database installed global query cache", name, mode)
			}
		}
	}
}

// TestPluginDatabaseUsesGlobalPragmas ensures the plugin database shares the
// global SQLite pragmas (a 4 MiB page cache) instead of overriding them with a
// per-database cache size.
func TestPluginDatabaseUsesGlobalPragmas(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	engine := &SQLiteEngine{}
	if err := engine.Init(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)

	for _, mode := range []constant.DBMode{constant.READ, constant.WRITE} {
		pool, err := engine.GetPluginDB(mode).DB()
		if err != nil {
			t.Fatal(err)
		}
		connection, err := pool.Conn(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		var cacheSize int
		scanErr := connection.QueryRowContext(t.Context(), "PRAGMA cache_size").Scan(&cacheSize)
		_ = connection.Close()
		if scanErr != nil {
			t.Fatal(scanErr)
		}
		if cacheSize != -4096 {
			t.Errorf("%v plugin cache_size = %d, want -4096", mode, cacheSize)
		}
	}
}
