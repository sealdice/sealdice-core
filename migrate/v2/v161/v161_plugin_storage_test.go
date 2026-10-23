package v161_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tidwall/buntdb"

	v161 "sealdice-core/migrate/v2/v161"
	"sealdice-core/migrate/v2/v2test"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/gormkv"
)

func TestV161PluginStorageMigrate(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	workDir := t.TempDir()
	t.Chdir(workDir)
	legacyPath := filepath.Join(workDir, "data", "default", "extensions", "demo", "storage.db")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy, err := buntdb.Open(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = legacy.Update(func(tx *buntdb.Tx) error {
		if _, _, setErr := tx.Set("normal", "value", nil); setErr != nil {
			return setErr
		}
		_, _, emptyErr := tx.Set("empty", "", nil)
		return emptyErr
	}); err != nil {
		t.Fatal(err)
	}
	if err = legacy.Close(); err != nil {
		t.Fatal(err)
	}

	var logs []string
	if err = v161.V161PluginStorageMigrate(op, func(line string) {
		logs = append(logs, line)
	}); err != nil {
		t.Fatal(err)
	}

	backend, err := gormkv.NewBackend(gormkv.BackendOptions{
		ReadDB:  op.GetPluginDB(constant.READ),
		WriteDB: op.GetPluginDB(constant.WRITE),
	})
	if err != nil {
		t.Fatal(err)
	}
	repo, err := backend.Namespace("demo")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"normal": "value", "empty": ""} {
		record, found, getErr := repo.Get(t.Context(), key)
		if getErr != nil || !found || string(record.Value) != want {
			t.Fatalf("key %q = %q found=%v err=%v, want %q", key, record.Value, found, getErr, want)
		}
	}
	if len(logs) == 0 {
		t.Fatal("migration should emit progress logs")
	}

	if err = v161.V161PluginStorageMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("direct retry should be idempotent: %v", err)
	}
	count, err := repo.Count(t.Context())
	if err != nil || count != 2 {
		t.Fatalf("row count after retry = %d, err=%v", count, err)
	}
}
