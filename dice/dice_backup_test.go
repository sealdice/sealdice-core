//nolint:testpackage
package dice

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.uber.org/zap"
	"gorm.io/gorm"

	sealdiceLogger "sealdice-core/logger"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
)

type backupTestOperator struct {
	dbType       string
	databaseFile map[string]string
}

func (o *backupTestOperator) Init(_ context.Context) error { return nil }

func (o *backupTestOperator) Type() string { return o.dbType }

func (o *backupTestOperator) DBCheck() {}

func (o *backupTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB { return nil }

func (o *backupTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB { return nil }

func (o *backupTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return nil }

func (o *backupTestOperator) GetBackupInfo() (string, map[string]string) {
	return o.dbType, maps.Clone(o.databaseFile)
}

func (o *backupTestOperator) Close() {}

func TestBackupSkipsDatabaseFilesForNonSQLite(t *testing.T) {
	t.Setenv("DATADIR", "")

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	var err error

	for _, dir := range []string{
		"data/default/configs",
		"data/default/extensions/reply",
		"data/default/scripts",
		"data/decks",
		"data/helpdoc",
		"data/censor",
		"data/names",
		"data/images",
	} {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err = os.WriteFile("data/dice.yaml", []byte("serveAddress: 0.0.0.0:3211\n"), 0o644); err != nil {
		t.Fatalf("write dice.yaml: %v", err)
	}
	if err = os.WriteFile("data/default/serve.yaml", []byte("name: default\n"), 0o644); err != nil {
		t.Fatalf("write serve.yaml: %v", err)
	}
	if err = os.WriteFile("data/default/configs/text-template.yaml", []byte(""), 0o644); err != nil {
		t.Fatalf("write text-template.yaml: %v", err)
	}
	for _, name := range []string{"data.db", "data-logs.db", "data-censor.db"} {
		if err = os.WriteFile(filepath.Join("data/default", name), []byte("stale sqlite"), 0o644); err != nil {
			t.Fatalf("write stale db %s: %v", name, err)
		}
	}

	dm := newBackupTestDiceManager(&backupTestOperator{
		dbType:       constant.MYSQL,
		databaseFile: nil,
	})

	backupPath, err := dm.Backup(BackupSelectionBasic, false)
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	entries, info := readBackupArchive(t, backupPath)
	for _, name := range []string{"data/default/data.db", "data/default/data-logs.db", "data/default/data-censor.db"} {
		if slices.Contains(entries, name) {
			t.Fatalf("backup unexpectedly contains %s", name)
		}
	}
	if info.DatabaseIncluded {
		t.Fatalf("backup_info.json databaseIncluded = true, want false")
	}
	if info.DatabaseType != constant.MYSQL {
		t.Fatalf("backup_info.json databaseType = %q, want %q", info.DatabaseType, constant.MYSQL)
	}
}

func TestBackupIncludesSQLiteFilesFromOperatorDataDir(t *testing.T) {
	t.Setenv("DATADIR", "")

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	var err error

	for _, dir := range []string{
		"data/default/configs",
		"data/default/extensions/reply",
		"data/default/scripts",
		"data/decks",
		"data/helpdoc",
		"data/censor",
		"data/names",
		"data/images",
	} {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err = os.WriteFile("data/dice.yaml", []byte("serveAddress: 0.0.0.0:3211\n"), 0o644); err != nil {
		t.Fatalf("write dice.yaml: %v", err)
	}
	if err = os.WriteFile("data/default/serve.yaml", []byte("name: default\n"), 0o644); err != nil {
		t.Fatalf("write serve.yaml: %v", err)
	}
	if err = os.WriteFile("data/default/configs/text-template.yaml", []byte(""), 0o644); err != nil {
		t.Fatalf("write text-template.yaml: %v", err)
	}

	dbDir := filepath.Join(tmpDir, "external-sqlite")
	if err = os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir dbDir: %v", err)
	}
	dbFiles := map[string]string{
		"data":   filepath.Join(dbDir, "data.db"),
		"logs":   filepath.Join(dbDir, "data-logs.db"),
		"censor": filepath.Join(dbDir, "data-censor.db"),
	}
	for key, path := range dbFiles {
		if err = os.WriteFile(path, []byte(key+"-content"), 0o644); err != nil {
			t.Fatalf("write sqlite file %s: %v", key, err)
		}
	}

	dm := newBackupTestDiceManager(&backupTestOperator{
		dbType:       constant.SQLITE,
		databaseFile: dbFiles,
	})
	dm.Dice[0].CensorManager = &CensorManager{DB: dm.Operator}

	backupPath, err := dm.Backup(BackupSelectionBasic, false)
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	entries, info := readBackupArchive(t, backupPath)
	for _, name := range []string{"external-sqlite/data.db", "external-sqlite/data-logs.db", "external-sqlite/data-censor.db"} {
		if !slices.Contains(entries, name) {
			t.Fatalf("backup missing %s; entries=%v", name, entries)
		}
	}
	if info.DatabaseIncluded != true {
		t.Fatalf("backup_info.json databaseIncluded = false, want true")
	}
	if info.DatabaseType != constant.SQLITE {
		t.Fatalf("backup_info.json databaseType = %q, want %q", info.DatabaseType, constant.SQLITE)
	}
}

type backupInfoMetadata struct {
	Config           json.RawMessage `json:"config"`
	Version          string          `json:"version"`
	VersionCode      int64           `json:"versionCode"`
	DatabaseType     string          `json:"databaseType"`
	DatabaseIncluded bool            `json:"databaseIncluded"`
}

func newBackupTestDiceManager(operator engine.DatabaseOperator) *DiceManager {
	d := &Dice{
		BaseConfig: BaseConfig{
			Name:    "default",
			DataDir: filepath.Join("data", "default"),
		},
		Logger:     zap.NewNop().Sugar(),
		LogWriter:  sealdiceLogger.NewUIWriter(),
		DBOperator: operator,
		ImSession: &IMSession{
			EndPoints: []*EndPointInfo{},
		},
	}
	dm := &DiceManager{
		Dice:     []*Dice{d},
		Operator: operator,
	}
	d.Parent = dm
	return dm
}

func readBackupArchive(t *testing.T, backupPath string) ([]string, backupInfoMetadata) {
	t.Helper()

	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatalf("open backup zip: %v", err)
	}
	defer reader.Close()

	entries := make([]string, 0, len(reader.File))
	var info backupInfoMetadata
	for _, file := range reader.File {
		entries = append(entries, file.Name)
		if file.Name != "backup_info.json" {
			continue
		}
		rc, openErr := file.Open()
		if openErr != nil {
			t.Fatalf("open backup_info.json: %v", openErr)
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			t.Fatalf("read backup_info.json: %v", readErr)
		}
		if err = json.Unmarshal(data, &info); err != nil {
			t.Fatalf("unmarshal backup_info.json: %v", err)
		}
	}
	return entries, info
}
