package dice //nolint:testpackage // Tests exercise the unexported restore transaction implementation.

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mysqlengine "sealdice-core/utils/dboperator/engine/mysql"
	sqliteengine "sealdice-core/utils/dboperator/engine/sqlite"
)

const restoreTestDiceConfig = "diceConfigs:\n  - name: default\n"

func prepareRestoreTest(t *testing.T) string {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("DATADIR", filepath.Join("data", "default"))
	if err := os.MkdirAll(filepath.Join(BackupDir, restoreDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join("data", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	originalProbe := probeRestoreDiskSpace
	probeRestoreDiskSpace = func(string) (packageDiskSpace, error) {
		return packageDiskSpace{Volume: "test", Available: 1 << 50, Total: 1 << 50}, nil
	}
	t.Cleanup(func() { probeRestoreDiskSpace = originalProbe })
	return filepath.Join(restoreDir(), restoreSourceName)
}

func writeTestFile(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeRestoreTestArchive(t *testing.T, filename string, files map[string]string, v2 bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		t.Fatal(err)
	}
	archive, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	manifestFiles := make([]backupManifestFile, 0, len(files))
	for name, content := range files {
		file, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, createErr = file.Write([]byte(content)); createErr != nil {
			t.Fatal(createErr)
		}
		digest := sha256.Sum256([]byte(content))
		manifestFiles = append(manifestFiles, backupManifestFile{
			Path:   filepath.ToSlash(name),
			Size:   uint64(len(content)),
			SHA256: hex.EncodeToString(digest[:]),
		})
	}
	manifest := backupManifest{
		Config:      mustMarshalJSON(map[string]any{"decks": true, "dices": map[string]any{"default": map[string]any{"jsScripts": true}}}),
		Version:     mustMarshalJSON("test"),
		VersionCode: VERSION_CODE,
	}
	if v2 {
		manifest.FormatVersion = backupManifestFormatVersion
		manifest.RestorePolicy = backupRestorePolicyOverlay
		manifest.DatabaseType = "sqlite"
		manifest.Files = manifestFiles
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestFile, err := writer.Create("backup_info.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manifestFile.Write(manifestData); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
}

func newRestoreSQLiteManager(t *testing.T) *DiceManager {
	t.Helper()
	writeTestFile(t, "data/dice.yaml", restoreTestDiceConfig)
	writeTestFile(t, "data/default/serve.yaml", "serve config")
	operator := &sqliteengine.SQLiteEngine{}
	if err := operator.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(operator.Close)
	d := &Dice{
		BaseConfig: BaseConfig{Name: "default", DataDir: filepath.Join("data", "default")},
		DBOperator: operator,
		ImSession:  &IMSession{},
	}
	return &DiceManager{Operator: operator, Dice: []*Dice{d}}
}

func assertRestoreTestFile(t *testing.T, filename, want string) {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", filename, data, want)
	}
}

func TestInspectBackupArchiveSupportsLegacyAndV2(t *testing.T) {
	for _, test := range []struct {
		name string
		v2   bool
	}{
		{name: "legacy", v2: false},
		{name: "v2", v2: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := prepareRestoreTest(t)
			writeRestoreTestArchive(t, source, map[string]string{"data/dice.yaml": restoreTestDiceConfig}, test.v2)
			info, _, err := inspectBackupArchive(source)
			if err != nil {
				t.Fatal(err)
			}
			if !info.Valid || !info.Restorable {
				t.Fatalf("unexpected archive info: %#v", info)
			}
			if test.v2 && info.FormatVersion != backupManifestFormatVersion {
				t.Fatalf("formatVersion = %d", info.FormatVersion)
			}
			if err = validateBackupData(source); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateBackupDataRejectsV2HashMismatch(t *testing.T) {
	source := prepareRestoreTest(t)
	archive, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	payload, err := writer.Create("data/dice.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = payload.Write([]byte(restoreTestDiceConfig))
	manifest := backupManifest{
		FormatVersion: backupManifestFormatVersion,
		RestorePolicy: backupRestorePolicyOverlay,
		DatabaseType:  "sqlite",
		Files: []backupManifestFile{{
			Path: "data/dice.yaml", Size: uint64(len(restoreTestDiceConfig)), SHA256: strings.Repeat("0", sha256.Size*2),
		}},
		Config: mustMarshalJSON(map[string]any{}), Version: mustMarshalJSON("test"), VersionCode: VERSION_CODE,
	}
	manifestData, _ := json.Marshal(manifest)
	manifestEntry, _ := writer.Create("backup_info.json")
	_, _ = manifestEntry.Write(manifestData)
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = validateBackupData(source); err == nil {
		t.Fatal("validateBackupData accepted corrupt v2 payload")
	}
}

func TestInspectBackupArchiveRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "traversal", path: "../outside.txt"},
		{name: "windows traversal", path: `data\..\outside.txt`},
		{name: "UNC", path: `\\server\share\file.txt`},
		{name: "drive", path: `C:\data\file.txt`},
		{name: "ADS", path: "data/file.txt:stream"},
		{name: "reserved", path: "data/CON.txt"},
		{name: "reserved superscript COM", path: "data/COM¹.txt"},
		{name: "reserved superscript LPT", path: "data/LPT².log"},
		{name: "reserved console input", path: "data/CONIN$"},
		{name: "reserved console output", path: "data/CONOUT$.txt"},
		{name: "reserved before extension", path: "data/CON .txt"},
		{name: "trailing dot", path: "data/name."},
		{name: "trailing space", path: "data/name "},
		{name: "WAL", path: "data/default/data.db-wal"},
		{name: "SHM", path: "data/default/data.db-shm"},
		{name: "journal", path: "data/default/data.db-journal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := prepareRestoreTest(t)
			writeRestoreTestArchive(t, source, map[string]string{
				"data/dice.yaml": restoreTestDiceConfig,
				test.path:        "unsafe",
			}, false)
			if _, _, err := inspectBackupArchive(source); err == nil {
				t.Fatalf("accepted unsafe path %q", test.path)
			}
		})
	}
}

func TestInspectBackupArchiveRejectsCaseAndFileDirectoryAliases(t *testing.T) {
	for _, files := range []map[string]string{
		{"data/dice.yaml": restoreTestDiceConfig, "data/Foo.txt": "a", "data/foo.txt": "b"},
		{"data/dice.yaml": restoreTestDiceConfig, "data/file": "a", "data/file/child": "b"},
	} {
		source := prepareRestoreTest(t)
		writeRestoreTestArchive(t, source, files, false)
		if _, _, err := inspectBackupArchive(source); err == nil {
			t.Fatal("accepted aliased ZIP paths")
		}
	}
}

func TestInspectBackupArchiveRejectsSymlinkAndSpecialFile(t *testing.T) {
	for _, mode := range []os.FileMode{os.ModeSymlink | 0o777, os.ModeNamedPipe | 0o600} {
		source := prepareRestoreTest(t)
		archive, err := os.Create(source)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(archive)
		config, _ := writer.Create("data/dice.yaml")
		_, _ = config.Write([]byte(restoreTestDiceConfig))
		header := &zip.FileHeader{Name: "data/unsafe"}
		header.SetMode(mode)
		unsafe, createErr := writer.CreateHeader(header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		_, _ = unsafe.Write([]byte("target"))
		manifest, _ := writer.Create("backup_info.json")
		_, _ = manifest.Write([]byte(`{"config":{},"version":"test","versionCode":1}`))
		_ = writer.Close()
		_ = archive.Close()
		if _, _, err = inspectBackupArchive(source); err == nil {
			t.Fatalf("accepted special mode %v", mode)
		}
	}
}

func TestInspectBackupArchiveRejectsZipBombHeader(t *testing.T) {
	source := prepareRestoreTest(t)
	archive, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	config, _ := writer.Create("data/dice.yaml")
	_, _ = config.Write([]byte(restoreTestDiceConfig))
	bombHeader := &zip.FileHeader{
		Name:               "data/bomb.bin",
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE([]byte{'x'}),
		CompressedSize64:   1,
		UncompressedSize64: compressionRatioMinimumSize + 1,
	}
	bomb, err := writer.CreateRaw(bombHeader)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = bomb.Write([]byte{'x'})
	manifest, _ := writer.Create("backup_info.json")
	_, _ = manifest.Write([]byte(`{"config":{},"version":"test","versionCode":1}`))
	_ = writer.Close()
	_ = archive.Close()
	if _, _, err = inspectBackupArchive(source); err == nil {
		t.Fatal("accepted suspicious compression ratio")
	}
}

func TestInspectBackupArchivePreflightsCentralDirectoryEntryCount(t *testing.T) {
	source := prepareRestoreTest(t)
	var header [46]byte
	binary.LittleEndian.PutUint32(header[:4], 0x02014b50)
	entryCount := maxBackupEntries + 1
	central := bytes.Repeat(header[:], entryCount)
	eocd := make([]byte, 22)
	binary.LittleEndian.PutUint32(eocd[:4], 0x06054b50)
	binary.LittleEndian.PutUint16(eocd[8:10], uint16(entryCount))  //nolint:gosec // ZIP32 stores the count modulo 65536.
	binary.LittleEndian.PutUint16(eocd[10:12], uint16(entryCount)) //nolint:gosec // Deliberately truncated malicious count.
	binary.LittleEndian.PutUint32(eocd[12:16], uint32(len(central)))
	if err := os.WriteFile(source, append(central, eocd...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := inspectBackupArchive(source); err == nil || !strings.Contains(err.Error(), "条目超过") {
		t.Fatalf("central directory count preflight did not reject archive: %v", err)
	}
}

func TestValidateBackupDataRejectsUnsafeDiceConfigNames(t *testing.T) {
	configs := []string{
		"diceConfigs:\n  - name: ../../outside\n",
		"diceConfigs:\n  - name: ''\n",
		"diceConfigs:\n  - name: default\n  - name: DEFAULT\n",
	}
	for _, v2 := range []bool{false, true} {
		for index, config := range configs {
			t.Run(fmt.Sprintf("v2-%t-%d", v2, index), func(t *testing.T) {
				source := prepareRestoreTest(t)
				writeRestoreTestArchive(t, source, map[string]string{"data/dice.yaml": config}, v2)
				if err := validateBackupData(source); err == nil {
					t.Fatal("validateBackupData accepted unsafe dice config name")
				}
			})
		}
	}
}

func TestInspectBackupArchiveMarksProcessOwnedFilesNonRestorable(t *testing.T) {
	for _, filename := range []string{"data/main.log", "data/panic.log", "data/runtime.lock"} {
		source := prepareRestoreTest(t)
		writeRestoreTestArchive(t, source, map[string]string{"data/dice.yaml": restoreTestDiceConfig, filename: "busy"}, false)
		info, _, err := inspectBackupArchive(source)
		if err != nil {
			t.Fatal(err)
		}
		if info.Restorable || !strings.Contains(info.RestoreError, filename) {
			t.Fatalf("process file should be non-restorable: %#v", info)
		}
	}
}

func TestScheduleRestoreIsIdempotentAndDoesNotCreateSafetyBackup(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	first, err := dm.ScheduleRestore("source.zip", "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Reused || first.SafetyBackupName != "" || first.ExpiresAt == 0 {
		t.Fatalf("unexpected first operation: %#v", first)
	}
	if restoreStatusTokenValidDuration != 24*time.Hour {
		t.Fatalf("status token validity = %v, want 24h", restoreStatusTokenValidDuration)
	}
	expiresIn := time.Until(time.Unix(first.ExpiresAt, 0))
	if expiresIn < 23*time.Hour+59*time.Minute || expiresIn > restoreStatusTokenValidDuration {
		t.Fatalf("status token expiry = %v, want approximately %v", expiresIn, restoreStatusTokenValidDuration)
	}
	second, err := dm.ScheduleRestore("source.zip", "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused || second.OperationID != first.OperationID || second.StatusToken != first.StatusToken || second.ExpiresAt != first.ExpiresAt {
		t.Fatalf("idempotent operation mismatch: first=%#v second=%#v", first, second)
	}
	pending, err := readPendingRestore()
	if err != nil {
		t.Fatal(err)
	}
	if pending.Phase != "scheduled" || pending.SafetyBackupName != "" {
		t.Fatalf("schedule performed active-runtime backup: %#v", pending)
	}
	if !ValidateRestoreStatusToken(first.OperationID, first.StatusToken) {
		t.Fatal("stored status token did not validate")
	}
	if len(first.StatusToken) < 43 {
		t.Fatalf("status token is too short: %d", len(first.StatusToken))
	}
}

func TestScheduleRestoreRefreshesExpiredTokenAndBindsAuthorizedStatus(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	first, err := dm.ScheduleRestore("source.zip", "request-refresh")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := readRestoreStatusAuth()
	if err != nil {
		t.Fatal(err)
	}
	pending, err := readPendingRestore()
	if err != nil {
		t.Fatal(err)
	}
	auth.ExpiresAt = time.Now().Add(-time.Minute).Unix()
	pending.ExpiresAt = auth.ExpiresAt
	if err = writeJSONAtomic(restoreStatusAuthPath(), auth); err != nil {
		t.Fatal(err)
	}
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	if err = setRestoreStatus(RestoreStatus{State: "rolled_back", OperationID: first.OperationID, SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	second, err := dm.ScheduleRestore("source.zip", first.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused || second.StatusToken == first.StatusToken || second.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("expired token was not refreshed: first=%#v second=%#v", first, second)
	}
	if ValidateRestoreStatusToken(first.OperationID, first.StatusToken) {
		t.Fatal("expired token still validates after refresh")
	}
	status, ok := GetRestoreStatusAuthorized(second.OperationID, second.StatusToken)
	if !ok || status.OperationID != second.OperationID || status.State != "pending" {
		t.Fatalf("authorized status = %#v, %v", status, ok)
	}
	if _, ok = GetRestoreStatusAuthorized("another-operation", second.StatusToken); ok {
		t.Fatal("token authorized a different operation")
	}
}

func TestScheduleRestoreRefreshesExpiredTerminalToken(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	first, err := dm.ScheduleRestore("source.zip", "request-terminal")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := readRestoreStatusAuth()
	if err != nil {
		t.Fatal(err)
	}
	auth.ExpiresAt = time.Now().Add(-time.Minute).Unix()
	if err = writeJSONAtomic(restoreStatusAuthPath(), auth); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(restorePendingPath()); err != nil {
		t.Fatal(err)
	}
	if err = setRestoreStatus(RestoreStatus{State: "succeeded", OperationID: first.OperationID, SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	second, err := dm.ScheduleRestore("source.zip", first.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused || second.StatusToken == first.StatusToken || !ValidateRestoreStatusToken(second.OperationID, second.StatusToken) {
		t.Fatalf("terminal token was not refreshed: %#v", second)
	}
}

func TestScheduleRestoreRepairsPendingBeforeAuthAndStatus(t *testing.T) {
	for _, phase := range []string{"scheduled", "prepared"} {
		t.Run(phase, func(t *testing.T) {
			queuedSource := prepareRestoreTest(t)
			dm := newRestoreSQLiteManager(t)
			writeRestoreTestArchive(t, queuedSource, map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
			writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
			pending := restorePending{Phase: phase, OperationID: "request-crash", SourceName: "source.zip"}
			if phase == "prepared" {
				pending.SafetyBackupName = "safety.zip"
				writeRestoreTestArchive(t, filepath.Join(BackupDir, pending.SafetyBackupName), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
			}
			if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
				t.Fatal(err)
			}
			operation, err := dm.ScheduleRestore("source.zip", "request-crash")
			if err != nil {
				t.Fatal(err)
			}
			if !operation.Reused || operation.StatusToken == "" || operation.ExpiresAt == 0 {
				t.Fatalf("operation was not repaired: %#v", operation)
			}
			if !ValidateRestoreStatusToken(operation.OperationID, operation.StatusToken) {
				t.Fatal("repaired token did not validate")
			}
			second, err := dm.ScheduleRestore("source.zip", "request-crash")
			if err != nil {
				t.Fatal(err)
			}
			if second.StatusToken != operation.StatusToken || second.ExpiresAt != operation.ExpiresAt {
				t.Fatalf("repaired operation was not stable: first=%#v second=%#v", operation, second)
			}
			runnable, err := HasRunnableScheduledRestore()
			if err != nil || !runnable {
				t.Fatalf("HasRunnableScheduledRestore = %v, %v", runnable, err)
			}
			wantState := "pending"
			if phase == "prepared" {
				wantState = "quiescing"
			}
			if status := GetRestoreStatus(); status.State != wantState || status.OperationID != "request-crash" {
				t.Fatalf("status = %#v", status)
			}
			operationID, err := RunnableScheduledRestoreOperationID()
			if err != nil || operationID != "request-crash" {
				t.Fatalf("RunnableScheduledRestoreOperationID = %q, %v", operationID, err)
			}
		})
	}
}

func TestScheduleRestoreRejectsExternalOrEscapedSQLiteData(t *testing.T) {
	prepareRestoreTest(t)
	writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	dm := newRestoreSQLiteManager(t)
	t.Setenv("DATADIR", filepath.Join(t.TempDir(), "external"))
	if _, err := dm.ScheduleRestore("source.zip", "request-1"); err == nil {
		t.Fatal("ScheduleRestore accepted DATADIR outside managed data")
	}
	dm.Operator = nil
	if _, err := dm.ScheduleRestore("source.zip", "request-2"); err == nil {
		t.Fatal("ScheduleRestore accepted nil database operator")
	}
}

func TestScheduleRestoreRejectsReusedOperationAfterDatabaseTypeChanges(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	if _, err := dm.ScheduleRestore("source.zip", "request-db-change"); err != nil {
		t.Fatal(err)
	}
	dm.Operator = &mysqlengine.MYSQLEngine{}
	if _, err := dm.ScheduleRestore("source.zip", "request-db-change"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "mysql") {
		t.Fatalf("reused restore accepted external database: %v", err)
	}
}

func TestPrepareScheduledRestoreCreatesStrictSafetyBackup(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	writeRestoreTestArchive(t, filepath.Join(BackupDir, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	operation, err := dm.ScheduleRestore("source.zip", "request-prepare")
	if err != nil {
		t.Fatal(err)
	}
	if err = PrepareScheduledRestore(dm); err != nil {
		t.Fatal(err)
	}
	pending, err := readPendingRestore()
	if err != nil {
		t.Fatal(err)
	}
	if pending.Phase != "prepared" || pending.SafetyBackupName == "" {
		t.Fatalf("unexpected prepared pending: %#v", pending)
	}
	if status := GetRestoreStatus(); status.State != "quiescing" || status.OperationID != operation.OperationID {
		t.Fatalf("unexpected external status: %#v", status)
	}
	if err = validateSafetyBackup(filepath.Join(BackupDir, pending.SafetyBackupName), dm); err != nil {
		t.Fatal(err)
	}
	_, _, manifest, err := inspectBackupArchiveDetailed(filepath.Join(BackupDir, pending.SafetyBackupName))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range manifest.Files {
		if strings.Contains(file.Path, "\\") {
			t.Fatalf("v2 export path is not slash-normalized: %q", file.Path)
		}
	}
	if err = PrepareScheduledRestore(dm); err != nil {
		t.Fatalf("PrepareScheduledRestore retry failed: %v", err)
	}
}

func TestStrictSafetyBackupUsesManagedDataDirInsteadOfDiceName(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	dm.Dice[0].BaseConfig.Name = "display-name"
	dm.Dice[0].BaseConfig.DataDir = filepath.Join("data", "custom")
	writeTestFile(t, "data/custom/serve.yaml", "serve config")
	backupPath, err := dm.backupStrict(BackupSelectionAll)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateSafetyBackup(backupPath, dm); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedBackupSkipsProcessOwnedFiles(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	writeTestFile(t, "data/default/scripts/main.js", "// script")
	writeTestFile(t, "data/default/scripts/runtime.lock", "locked")
	backupPath, err := dm.Backup(BackupSelectionJS, false)
	if err != nil {
		t.Fatal(err)
	}
	info, entries, err := inspectBackupArchive(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Restorable {
		t.Fatalf("generated backup is not restorable: %#v", info)
	}
	if _, exists := entries["data/default/scripts/runtime.lock"]; exists {
		t.Fatal("generated backup contains process-owned lock file")
	}
	if _, exists := entries["data/default/scripts/main.js"]; !exists {
		t.Fatal("generated backup omitted normal script")
	}
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	for _, entry := range reader.File {
		if entry.Name == "data/default/scripts/main.js" && entry.Mode().Perm() != 0o600 {
			t.Fatalf("generated backup widened file mode: %v", entry.Mode().Perm())
		}
	}
}

func prepareApplyTest(t *testing.T, files map[string]string) {
	t.Helper()
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, files, true)
	pending := restorePending{Phase: "prepared", OperationID: "op", SourceName: "source.zip", SafetyBackupName: "safety.zip"}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	if err := setRestoreStatus(RestoreStatus{State: "quiescing", OperationID: "op", SourceName: "source.zip", SafetyBackupName: "safety.zip"}); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRestoreUsesAppendOnlyJournalAndOverlay(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig, "data/new.txt": "added"})
	writeTestFile(t, "data/dice.yaml", "old")
	writeTestFile(t, "data/keep.txt", "keep")
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	assertRestoreTestFile(t, "data/dice.yaml", restoreTestDiceConfig)
	assertRestoreTestFile(t, "data/new.txt", "added")
	assertRestoreTestFile(t, "data/keep.txt", "keep")
	if os.PathSeparator == '/' {
		info, err := os.Stat("data/new.txt")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("restored file mode = %v, want 0600", info.Mode().Perm())
		}
	}
	journalData, err := os.ReadFile(restoreJournalPath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(journalData, []byte(`"phase":"intent"`)) || !bytes.Contains(journalData, []byte(`"phase":"done"`)) {
		t.Fatalf("journal lacks intent/done records: %s", journalData)
	}
	journal, err := readRestoreJournal()
	if err != nil || journal.State != "applied" {
		t.Fatalf("journal=%#v err=%v", journal, err)
	}
}

func TestApplyScheduledRestoreRollsBackOnRenameFailure(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig, "data/new.txt": "added"})
	writeTestFile(t, "data/dice.yaml", "old")
	originalRename := restoreRename
	restoreRename = func(oldPath, newPath string) error {
		if filepath.Base(oldPath) == "new.txt" && filepath.Clean(newPath) == filepath.Clean("data/new.txt") {
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	}
	t.Cleanup(func() { restoreRename = originalRename })
	if err := ApplyScheduledRestore(); err == nil {
		t.Fatal("ApplyScheduledRestore unexpectedly succeeded")
	}
	assertRestoreTestFile(t, "data/dice.yaml", "old")
	if _, err := os.Stat("data/new.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new file remains after rollback: %v", err)
	}
	journal, err := readRestoreJournal()
	if err != nil || journal.State != "rolled_back" {
		t.Fatalf("journal=%#v err=%v", journal, err)
	}
	status := GetRestoreStatus()
	if status.State != "rolling_back" || !strings.Contains(status.Message, "等待回滚") {
		t.Fatalf("apply failure exposed a terminal status: %#v", status)
	}
}

func TestRestoreTransactionSynthesizesSQLiteSidecars(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig, "data/default/data.db": "new-db"})
	writeTestFile(t, "data/dice.yaml", "old")
	writeTestFile(t, "data/default/data.db", "old-db")
	writeTestFile(t, "data/default/data.db-wal", "old-wal")
	writeTestFile(t, "data/default/data.db-shm", "old-shm")
	writeTestFile(t, "data/default/data.db-journal", "old-journal")
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat("data/default/data.db" + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("sidecar remains after apply: %s", suffix)
		}
	}
	if err := RecoverInterruptedRestore(); err != nil {
		t.Fatal(err)
	}
	assertRestoreTestFile(t, "data/default/data.db-wal", "old-wal")
	assertRestoreTestFile(t, "data/default/data.db-shm", "old-shm")
	assertRestoreTestFile(t, "data/default/data.db-journal", "old-journal")
}

func TestRestoreRollbackRemovesSidecarsCreatedByNewRuntime(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig, "data/default/data.db": "new-db"})
	writeTestFile(t, "data/dice.yaml", "old")
	writeTestFile(t, "data/default/data.db", "old-db")
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		writeTestFile(t, "data/default/data.db"+suffix, "new-runtime-sidecar")
	}
	if err := RollbackScheduledRestore("new Runtime validation failed"); err != nil {
		t.Fatal(err)
	}
	assertRestoreTestFile(t, "data/default/data.db", "old-db")
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat("data/default/data.db" + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("new Runtime sidecar remains after rollback: %s: %v", suffix, err)
		}
	}
}

func TestRecoverInterruptedRestoreAfterRestoreRenameBeforeDone(t *testing.T) {
	for _, test := range []struct {
		name       string
		files      map[string]string
		target     string
		oldContent string
	}{
		{
			name: "ordinary file",
			files: map[string]string{
				"data/dice.yaml": restoreTestDiceConfig,
			},
			target: "data/dice.yaml", oldContent: "old",
		},
		{
			name: "sqlite sidecar",
			files: map[string]string{
				"data/dice.yaml":       restoreTestDiceConfig,
				"data/default/data.db": "new-db",
			},
			target: "data/default/data.db-wal", oldContent: "old-wal",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepareApplyTest(t, test.files)
			writeTestFile(t, "data/dice.yaml", "old")
			if strings.HasSuffix(test.target, "-wal") {
				writeTestFile(t, "data/default/data.db", "old-db")
				writeTestFile(t, test.target, test.oldContent)
			}
			if err := ApplyScheduledRestore(); err != nil {
				t.Fatal(err)
			}
			journal, err := readRestoreJournal()
			if err != nil {
				t.Fatal(err)
			}
			if err = appendJournalState(journal, "rolling_back"); err != nil {
				t.Fatal(err)
			}
			entryIndex := -1
			for index := range journal.Entries {
				if filepath.Clean(journal.Entries[index].Target) == filepath.Clean(test.target) {
					entryIndex = index
					break
				}
			}
			if entryIndex < 0 {
				t.Fatalf("journal lacks target %s", test.target)
			}
			entry := &journal.Entries[entryIndex]
			if err = appendJournalStep(journal, entryIndex, "remove-installed", "intent"); err != nil {
				t.Fatal(err)
			}
			if err = os.Remove(entry.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if err = appendJournalStep(journal, entryIndex, "remove-installed", "done"); err != nil {
				t.Fatal(err)
			}
			if err = appendJournalStep(journal, entryIndex, "restore-original", "intent"); err != nil {
				t.Fatal(err)
			}
			if err = os.Rename(entry.Rollback, entry.Target); err != nil {
				t.Fatal(err)
			}
			// Simulate a crash before restore-original done reaches the journal.
			if err = RecoverInterruptedRestore(); err != nil {
				t.Fatal(err)
			}
			assertRestoreTestFile(t, test.target, test.oldContent)
		})
	}
}

func TestApplyScheduledRestoreRollsBackAfterDirectorySyncFailure(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig})
	writeTestFile(t, "data/dice.yaml", "old")
	originalSync := restoreSyncMutationParents
	failed := false
	restoreSyncMutationParents = func(rootAndFiles ...string) error {
		if err := originalSync(rootAndFiles...); err != nil {
			return err
		}
		if !failed {
			failed = true
			return errors.New("injected directory sync failure")
		}
		return nil
	}
	t.Cleanup(func() { restoreSyncMutationParents = originalSync })
	if err := ApplyScheduledRestore(); err == nil {
		t.Fatal("ApplyScheduledRestore unexpectedly ignored directory sync failure")
	}
	assertRestoreTestFile(t, "data/dice.yaml", "old")
}

func TestRecoverInterruptedRestoreHandlesBackupIntentCrash(t *testing.T) {
	prepareRestoreTest(t)
	writeTestFile(t, "data/dice.yaml", "old")
	pending := restorePending{Phase: "applying", OperationID: "op", SourceName: "source.zip", SafetyBackupName: "safety.zip"}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	rollback := filepath.Join(restoreDir(), restoreRollbackName, "data", "dice.yaml")
	if err := os.MkdirAll(filepath.Dir(rollback), 0o700); err != nil {
		t.Fatal(err)
	}
	journal := &restoreJournal{OperationID: "op", Entries: []restoreJournalEntry{{
		Target: "data/dice.yaml", Rollback: rollback, HadOriginal: true,
	}}}
	if err := createRestoreJournal(journal); err != nil {
		t.Fatal(err)
	}
	if err := appendJournalStep(journal, 0, "backup-original", "intent"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename("data/dice.yaml", rollback); err != nil {
		t.Fatal(err)
	}
	journalFile, err := os.OpenFile(restoreJournalPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = journalFile.WriteString(`{"operationId":"op"`); err != nil {
		t.Fatal(err)
	}
	if err = errors.Join(journalFile.Sync(), journalFile.Close()); err != nil {
		t.Fatal(err)
	}
	if err = RecoverInterruptedRestore(); err != nil {
		t.Fatal(err)
	}
	assertRestoreTestFile(t, "data/dice.yaml", "old")
	if status := GetRestoreStatus(); status.State != "rolled_back" {
		t.Fatalf("status = %#v", status)
	}
	journal, err = readRestoreJournal()
	if err != nil || journal.State != "rolled_back" {
		t.Fatalf("journal remained corrupt after torn tail recovery: %#v, %v", journal, err)
	}
}

func TestRecoverInterruptedRestoreKeepsSafeNoJournalPending(t *testing.T) {
	for _, phase := range []string{"scheduled", "prepared"} {
		t.Run(phase, func(t *testing.T) {
			prepareRestoreTest(t)
			pending := restorePending{Phase: phase, OperationID: "op", SourceName: "source.zip"}
			if phase == "prepared" {
				pending.SafetyBackupName = "safety.zip"
			}
			if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
				t.Fatal(err)
			}
			if err := setRestoreStatus(RestoreStatus{State: "pending", OperationID: "op", SourceName: "source.zip"}); err != nil {
				t.Fatal(err)
			}
			if err := RecoverInterruptedRestore(); err != nil {
				t.Fatal(err)
			}
			updated, err := readPendingRestore()
			if err != nil || updated.Phase != phase {
				t.Fatalf("pending=%#v err=%v", updated, err)
			}
		})
	}
}

func TestRecoverInterruptedRestoreDiscardsTornBeginBeforeDataChanges(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: "scheduled", OperationID: "op", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := setRestoreStatus(RestoreStatus{State: "pending", OperationID: "op", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(restoreJournalPath(), []byte(`{"operationId":"op","type":"begin"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RecoverInterruptedRestore(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(restoreJournalPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("torn begin journal was not discarded: %v", err)
	}
	pending, err := readPendingRestore()
	if err != nil || pending.Phase != "scheduled" {
		t.Fatalf("pending changed after safe torn begin: %#v, %v", pending, err)
	}
	operationID, err := RunnableScheduledRestoreOperationID()
	if err != nil || operationID != "op" {
		t.Fatalf("runnable operation after torn begin = %q, %v", operationID, err)
	}
}

func TestRecoverInterruptedRestoreAcceptsSucceededMetadataAfterJournalCleanup(t *testing.T) {
	prepareRestoreTest(t)
	if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: "applied", OperationID: "op", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := setRestoreStatus(RestoreStatus{State: "succeeded", OperationID: "op", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := RecoverInterruptedRestore(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(restorePendingPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("succeeded pending residue was not removed: %v", err)
	}
}

func TestRollbackScheduledRestoreRejectsUnsafePhaseWithoutJournal(t *testing.T) {
	prepareRestoreTest(t)
	if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: "applying", OperationID: "op", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := RollbackScheduledRestore("must not claim rollback"); err == nil {
		t.Fatal("RollbackScheduledRestore accepted applying pending without journal")
	}
}

func TestBuildRestoreJournalRejectsLinkedTargetDirectory(t *testing.T) {
	prepareRestoreTest(t)
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join("data", "link")); err != nil {
		t.Skipf("current environment cannot create symlink: %v", err)
	}
	staged := filepath.Join(restoreDir(), restoreStagingName)
	writeTestFile(t, filepath.Join(staged, "data", "link", "outside.txt"), "unsafe")
	if _, err := buildRestoreJournal("op", staged); err == nil {
		t.Fatal("buildRestoreJournal accepted linked target directory")
	}
	if _, err := os.Stat(filepath.Join(external, "outside.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restore escaped through linked target: %v", err)
	}
}

func TestRecoverInterruptedRestoreRejectsAmbiguousOrMismatchedOperation(t *testing.T) {
	for _, test := range []struct {
		name       string
		pendingID  string
		journalID  string
		withIntent bool
	}{
		{name: "operation mismatch", pendingID: "pending", journalID: "journal"},
		{name: "ambiguous move", pendingID: "op", journalID: "op", withIntent: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepareRestoreTest(t)
			pending := restorePending{Phase: "applying", OperationID: test.pendingID, SafetyBackupName: "safety.zip"}
			if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
				t.Fatal(err)
			}
			journal := &restoreJournal{OperationID: test.journalID, Entries: []restoreJournalEntry{{
				Target: "data/missing.txt", Rollback: filepath.Join(restoreDir(), restoreRollbackName, "data/missing.txt"), HadOriginal: true,
			}}}
			if err := createRestoreJournal(journal); err != nil {
				t.Fatal(err)
			}
			if test.withIntent {
				if err := appendJournalStep(journal, 0, "backup-original", "intent"); err != nil {
					t.Fatal(err)
				}
			}
			if err := RecoverInterruptedRestore(); err == nil {
				t.Fatal("RecoverInterruptedRestore accepted ambiguous transaction")
			}
		})
	}
}

func TestCommitIsOnlyIrreversiblePointAndSuccessFollowsRuntimePublish(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig})
	writeTestFile(t, "data/dice.yaml", "old")
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	if err := CommitScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	journal, err := readRestoreJournal()
	if err != nil || journal.State != "committed" {
		t.Fatalf("journal=%#v err=%v", journal, err)
	}
	if status := GetRestoreStatus(); status.State == "succeeded" {
		t.Fatal("commit published succeeded before Runtime publication")
	}
	committed, err := HasCommittedRestore()
	if err != nil || !committed {
		t.Fatalf("HasCommittedRestore = %v, %v", committed, err)
	}
	if err = MarkScheduledRestoreSucceeded(); err != nil {
		t.Fatal(err)
	}
	if status := GetRestoreStatus(); status.State != "succeeded" {
		t.Fatalf("status = %#v", status)
	}
	if _, err = os.Stat(restoreJournalPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("journal remains after success: %v", err)
	}
}

func TestCommittedCleanupFailureDoesNotBecomeRollbackFailure(t *testing.T) {
	prepareApplyTest(t, map[string]string{"data/dice.yaml": restoreTestDiceConfig})
	writeTestFile(t, "data/dice.yaml", "old")
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	originalRemoveAll := restoreRemoveAll
	restoreRemoveAll = func(target string) error {
		if filepath.Base(target) == restoreRollbackName {
			return errors.New("injected cleanup failure")
		}
		return os.RemoveAll(target)
	}
	t.Cleanup(func() { restoreRemoveAll = originalRemoveAll })
	if err := CommitScheduledRestore(); err != nil {
		t.Fatalf("commit should remain successful after cleanup failure: %v", err)
	}
	journal, err := readRestoreJournal()
	if err != nil || journal.State != "committed" {
		t.Fatalf("journal=%#v err=%v", journal, err)
	}
	if err = RollbackScheduledRestore("must not rollback"); err == nil {
		t.Fatal("committed restore was rolled back")
	}
	restoreRemoveAll = originalRemoveAll
	if err = MarkScheduledRestoreSucceeded(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAndCleanBackupRespectPendingRestore(t *testing.T) {
	prepareRestoreTest(t)
	writeTestFile(t, filepath.Join(BackupDir, "source.zip"), "source")
	writeTestFile(t, filepath.Join(BackupDir, "old.zip"), "old")
	if err := writeJSONAtomic(restorePendingPath(), restorePending{OperationID: "op", Phase: "scheduled", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteBackup("source.zip"); err == nil {
		t.Fatal("DeleteBackup removed in-use source")
	}
	if err := DeleteBackup("SOURCE.ZIP"); err == nil || !strings.Contains(err.Error(), "使用") {
		t.Fatalf("DeleteBackup did not recognize case-folded in-use source: %v", err)
	}
	dm := &DiceManager{BackupCleanStrategy: BackupCleanStrategyByCount, BackupCleanKeepCount: 0}
	if err := dm.BackupClean(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(BackupDir, "source.zip")); err != nil {
		t.Fatalf("BackupClean removed in-use source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(BackupDir, "old.zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("BackupClean did not remove old backup: %v", err)
	}
}

func TestOpenBackupArchiveReturnsVerifiedHandle(t *testing.T) {
	prepareRestoreTest(t)
	filename := filepath.Join(BackupDir, "source.zip")
	writeTestFile(t, filename, "archive bytes")
	file, info, err := OpenBackupArchive("source.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	openedInfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(info, openedInfo) {
		t.Fatal("returned info does not describe opened archive")
	}
	data, err := io.ReadAll(file)
	if err != nil || string(data) != "archive bytes" {
		t.Fatalf("opened archive = %q, %v", data, err)
	}
}

func TestReleaseRetryablePendingRefusesLegacyArtifactsWithoutPending(t *testing.T) {
	prepareRestoreTest(t)
	legacyOldData := filepath.Join(restoreDir(), restoreLegacyOldDataName)
	writeTestFile(t, filepath.Join(legacyOldData, "data.db"), "keep")
	if err := releaseRetryablePendingRestore(); err == nil {
		t.Fatal("releaseRetryablePendingRestore accepted legacy old-data without pending")
	}
	assertRestoreTestFile(t, filepath.Join(legacyOldData, "data.db"), "keep")

	if err := restoreRemoveAll(restoreDir()); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, restoreLegacyJournalPath(), `{"legacy":true}`)
	if err := releaseRetryablePendingRestore(); err == nil {
		t.Fatal("releaseRetryablePendingRestore accepted legacy journal without pending")
	}
	assertRestoreTestFile(t, restoreLegacyJournalPath(), `{"legacy":true}`)
}

func TestScheduleRestoreRejectsLinkedBackupRootBeforeCleanup(t *testing.T) {
	prepareRestoreTest(t)
	dm := newRestoreSQLiteManager(t)
	external := t.TempDir()
	writeTestFile(t, filepath.Join(external, restoreDirName, "sentinel"), "keep")
	writeRestoreTestArchive(t, filepath.Join(external, "source.zip"), map[string]string{"data/dice.yaml": restoreTestDiceConfig}, true)
	if err := os.RemoveAll(BackupDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, BackupDir); err != nil {
		t.Skipf("current environment cannot create directory symlink: %v", err)
	}
	if _, err := dm.ScheduleRestore("source.zip", "request-linked-root"); err == nil {
		t.Fatal("ScheduleRestore accepted linked backup root")
	}
	assertRestoreTestFile(t, filepath.Join(external, restoreDirName, "sentinel"), "keep")
}

func TestBackupArchivePathRejectsSymlink(t *testing.T) {
	prepareRestoreTest(t)
	external := filepath.Join(t.TempDir(), "external.zip")
	writeTestFile(t, external, "external")
	if err := os.Symlink(external, filepath.Join(BackupDir, "linked.zip")); err != nil {
		t.Skipf("current environment cannot create symlink: %v", err)
	}
	if _, err := BackupArchivePath("linked.zip"); err == nil {
		t.Fatal("BackupArchivePath accepted symlink")
	}
	if file, _, err := OpenBackupArchive("linked.zip"); err == nil {
		_ = file.Close()
		t.Fatal("OpenBackupArchive accepted symlink")
	}
}
