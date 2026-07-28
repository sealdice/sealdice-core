package dice //nolint:testpackage // Tests exercise the unexported restore transaction implementation.

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeRestoreTestArchive(t *testing.T, filename string, versionCode int64, files map[string]string) {
	t.Helper()
	archive, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	manifest, err := json.Marshal(map[string]any{
		"version":     "1.0.0",
		"versionCode": versionCode,
		"config": map[string]any{
			"decks": true,
			"dices": map[string]any{
				"default": map[string]any{"jsScripts": true},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestFile, err := writer.Create("backup_info.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manifestFile.Write(manifest); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		file, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, createErr = file.Write([]byte(content)); createErr != nil {
			t.Fatal(createErr)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyScheduledRestoreRollsBackOnRenameFailure(t *testing.T) {
	source := prepareRestoreTest(t)
	if err := os.WriteFile("data/dice.yaml", []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "new",
		"data/new.txt":   "added",
	})
	pending := restorePending{Phase: "pending", OperationID: "op", SourceName: "source.zip", SafetyBackupName: "safety.zip"}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	originalRename := restoreRename
	failed := false
	restoreRename = func(oldPath, newPath string) error {
		if !failed && filepath.Base(oldPath) == "new.txt" && filepath.Clean(newPath) == filepath.Clean("data/new.txt") {
			failed = true
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	}
	t.Cleanup(func() { restoreRename = originalRename })
	if err := ApplyScheduledRestore(); err == nil {
		t.Fatal("ApplyScheduledRestore() unexpectedly succeeded")
	}
	assertRestoreTestFile(t, "data/dice.yaml", "old")
	if _, err := os.Stat("data/new.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new file remains after rollback: %v", err)
	}
}

func TestRestoreTransactionIncludesSQLiteSidecars(t *testing.T) {
	source := prepareRestoreTest(t)
	for name, content := range map[string]string{
		"data/dice.yaml":      "old-config",
		"data/default.db":     "old-db",
		"data/default.db-wal": "old-wal",
		"data/default.db-shm": "old-shm",
	} {
		if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml":  "new-config",
		"data/default.db": "new-db",
	})
	if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: "pending", OperationID: "op", SourceName: "source.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	for _, sidecar := range []string{"data/default.db-wal", "data/default.db-shm"} {
		if _, err := os.Stat(sidecar); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("SQLite sidecar was not removed: %s (%v)", sidecar, err)
		}
	}
	if err := RecoverInterruptedRestore(); err != nil {
		t.Fatal(err)
	}
	assertRestoreTestFile(t, "data/default.db-wal", "old-wal")
	assertRestoreTestFile(t, "data/default.db-shm", "old-shm")
}

func TestRecoverInterruptedRestoreMigratesLegacySwitchingState(t *testing.T) {
	prepareRestoreTest(t)
	pending := restorePending{Phase: "switching", SourceName: "source.zip", SafetyBackupName: "safety.zip"}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	if err := RecoverInterruptedRestore(); err != nil {
		t.Fatal(err)
	}
	updated, err := readPendingRestore()
	if err != nil {
		t.Fatal(err)
	}
	if updated.Phase != "pending" {
		t.Fatalf("phase = %q, want pending", updated.Phase)
	}
}

func TestRecoverInterruptedRestoreStopsWhenLegacyOldDataExists(t *testing.T) {
	prepareRestoreTest(t)
	if err := os.MkdirAll(filepath.Join(restoreDir(), restoreLegacyOldDataName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: "switching", SafetyBackupName: "safety.zip"}); err != nil {
		t.Fatal(err)
	}
	if err := RecoverInterruptedRestore(); err == nil {
		t.Fatal("RecoverInterruptedRestore() should stop for legacy old-data")
	}
}

func TestReleaseRetryablePendingRestore(t *testing.T) {
	prepareRestoreTest(t)
	if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := setRestoreStatus(RestoreStatus{State: "failed"}); err != nil {
		t.Fatal(err)
	}
	if err := releaseRetryablePendingRestore(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(restorePendingPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending restore was not released: %v", err)
	}
}

func TestReleaseRetryablePendingRestoreKeepsUnsafeState(t *testing.T) {
	tests := []struct {
		name    string
		phase   string
		state   string
		oldData bool
		journal string
	}{
		{name: "active task", phase: "pending", state: "pending"},
		{name: "legacy old data", phase: "pending", state: "failed", oldData: true},
		{name: "unfinished journal", phase: "pending", state: "failed", journal: "applying"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepareRestoreTest(t)
			if err := writeJSONAtomic(restorePendingPath(), restorePending{Phase: test.phase}); err != nil {
				t.Fatal(err)
			}
			if err := setRestoreStatus(RestoreStatus{State: test.state}); err != nil {
				t.Fatal(err)
			}
			if test.oldData {
				if err := os.MkdirAll(filepath.Join(restoreDir(), restoreLegacyOldDataName), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if test.journal != "" {
				if err := writeRestoreJournal(&restoreJournal{State: test.journal}); err != nil {
					t.Fatal(err)
				}
			}
			if err := releaseRetryablePendingRestore(); err == nil {
				t.Fatal("unsafe pending restore was released")
			}
			if _, err := os.Stat(restorePendingPath()); err != nil {
				t.Fatalf("pending restore marker was removed: %v", err)
			}
		})
	}
}

func TestRestoreStatusTokenSurvivesCommit(t *testing.T) {
	prepareRestoreTest(t)
	token := "temporary-token"
	digest := sha256.Sum256([]byte(token))
	pending := restorePending{Phase: "applied", OperationID: "op", StatusTokenHash: hex.EncodeToString(digest[:])}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(restoreStatusAuthPath(), restoreStatusAuth{OperationID: "op", TokenHash: pending.StatusTokenHash, ExpiresAt: time.Now().Add(time.Minute).Unix()}); err != nil {
		t.Fatal(err)
	}
	if !ValidateRestoreStatusToken("op", token) {
		t.Fatal("token should be valid while pending")
	}
	if err := os.Remove(restorePendingPath()); err != nil {
		t.Fatal(err)
	}
	if !ValidateRestoreStatusToken("op", token) {
		t.Fatal("token should remain valid after commit")
	}
}

func TestRollbackRestoreJournalHandlesCrashWindows(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{name: "after original moved", state: "backing_up"},
		{name: "after restored file moved", state: "installing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepareRestoreTest(t)
			target := filepath.Join("data", "dice.yaml")
			rollback := filepath.Join(restoreDir(), restoreRollbackName, target)
			if err := os.MkdirAll(filepath.Dir(rollback), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(rollback, []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}
			if test.state == "installing" {
				if err := os.WriteFile(target, []byte("new"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			journal := &restoreJournal{State: "applying", Entries: []restoreJournalEntry{{
				Target: target, Rollback: rollback, HadOriginal: true, State: test.state,
			}}}
			if err := rollbackRestoreJournal(journal); err != nil {
				t.Fatal(err)
			}
			assertRestoreTestFile(t, target, "old")
		})
	}
}

func prepareRestoreTest(t *testing.T) string {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join(BackupDir, restoreDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(restoreDir(), restoreSourceName)
}

func TestInspectBackupArchive(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "current",
	})
	info, _, err := inspectBackupArchive(source)
	if err != nil {
		t.Fatalf("inspectBackupArchive() error = %v", err)
	}
	if !info.Valid || !info.Restorable {
		t.Fatalf("unexpected archive info: %#v", info)
	}
	wantSelection := int64(BackupSelectionDecks | BackupSelectionJS)
	if info.Selection != wantSelection {
		t.Fatalf("selection = %d, want %d", info.Selection, wantSelection)
	}
}

func TestImportBackupReportsReusedExactArchive(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "current",
	})
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	first, err := ImportBackup(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("first ImportBackup() error = %v", err)
	}
	if first.Reused {
		t.Fatal("first ImportBackup() unexpectedly reused a file")
	}
	second, err := ImportBackup(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("second ImportBackup() error = %v", err)
	}
	if !second.Reused || second.Name != first.Name {
		t.Fatalf("second import = %#v, want reused %q", second, first.Name)
	}
}

func TestImportBackupDoesNotTrustShortHashFilename(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "current",
	})
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	shortDigest := hex.EncodeToString(digest[:])[:8]
	collisionName := filepath.Join(BackupDir, "import_existing_"+shortDigest+".zip")
	if err = os.WriteFile(collisionName, []byte("different content"), 0o600); err != nil {
		t.Fatal(err)
	}
	item, err := ImportBackup(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ImportBackup() error = %v", err)
	}
	if item.Reused || item.Name == filepath.Base(collisionName) {
		t.Fatalf("ImportBackup() incorrectly reused short-hash candidate: %#v", item)
	}
}

func TestInspectBackupArchiveAcceptsWindowsSeparators(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data\\dice.yaml":            "current",
		"data\\images\\sealdice.png": "image",
	})
	info, entries, err := inspectBackupArchive(source)
	if err != nil {
		t.Fatalf("inspectBackupArchive() error = %v", err)
	}
	if !info.Valid {
		t.Fatalf("unexpected archive info: %#v", info)
	}
	if _, exists := entries["data/images/sealdice.png"]; !exists {
		t.Fatalf("Windows path was not normalized: %#v", entries)
	}
}

func TestInspectBackupArchiveRejectsFutureVersion(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE+1, map[string]string{
		"data/dice.yaml": "future",
	})
	if _, _, err := inspectBackupArchive(source); err == nil {
		t.Fatal("inspectBackupArchive() accepted a future backup")
	}
}

func TestInspectBackupArchiveRejectsPathTraversal(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "current",
		"../outside.txt": "unsafe",
	})
	if _, _, err := inspectBackupArchive(source); err == nil {
		t.Fatal("inspectBackupArchive() accepted a path traversal entry")
	}
}

func TestInspectBackupArchiveRejectsWindowsPathTraversal(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data\\dice.yaml":       "current",
		"data\\..\\outside.txt": "unsafe",
	})
	if _, _, err := inspectBackupArchive(source); err == nil {
		t.Fatal("inspectBackupArchive() accepted a Windows path traversal entry")
	}
}

func TestInspectBackupArchiveRejectsCaseCollisions(t *testing.T) {
	source := prepareRestoreTest(t)
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "current",
		"data/Foo.txt":   "first",
		"data/foo.txt":   "second",
	})
	if _, _, err := inspectBackupArchive(source); err == nil {
		t.Fatal("inspectBackupArchive() accepted case-colliding paths")
	}
}

func TestApplyAndCommitPendingRestore(t *testing.T) {
	source := prepareRestoreTest(t)
	if err := os.WriteFile("data/dice.yaml", []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/keep.txt", []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "new",
		"data/new.txt":   "added",
	})
	pending := restorePending{Phase: "pending", SourceName: "source.zip", SafetyBackupName: "safety.zip"}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatalf("ApplyScheduledRestore() error = %v", err)
	}
	assertRestoreTestFile(t, "data/dice.yaml", "new")
	assertRestoreTestFile(t, "data/keep.txt", "keep")
	assertRestoreTestFile(t, "data/new.txt", "added")
	if _, err := os.Stat(filepath.Join(restoreDir(), restoreRollbackName, "data", "dice.yaml")); err != nil {
		t.Fatalf("old file was not retained before commit: %v", err)
	}
	if err := CommitScheduledRestore(); err != nil {
		t.Fatalf("CommitScheduledRestore() error = %v", err)
	}
	if status := GetRestoreStatus(); status.State != "succeeded" {
		t.Fatalf("restore state = %q, want succeeded", status.State)
	}
	if _, err := os.Stat(restorePendingPath()); !os.IsNotExist(err) {
		t.Fatalf("pending marker still exists: %v", err)
	}
}

func TestApplyPendingRestoreRollsBackUncommittedSwap(t *testing.T) {
	source := prepareRestoreTest(t)
	if err := os.WriteFile("data/dice.yaml", []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRestoreTestArchive(t, source, VERSION_CODE, map[string]string{
		"data/dice.yaml": "new",
	})
	pending := restorePending{Phase: "pending", SourceName: "source.zip", SafetyBackupName: "safety.zip"}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		t.Fatal(err)
	}
	if err := ApplyScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	if err := RecoverInterruptedRestore(); err != nil {
		t.Fatalf("RecoverInterruptedRestore() error = %v", err)
	}
	assertRestoreTestFile(t, "data/dice.yaml", "old")
	journal, err := readRestoreJournal()
	if err != nil {
		t.Fatal(err)
	}
	if journal.State != "rolled_back" {
		t.Fatalf("journal state = %q, want rolled_back", journal.State)
	}
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
