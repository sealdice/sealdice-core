//nolint:testpackage // These tests need access to unexported routing cores.
package logger

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	gormlogger "gorm.io/gorm/logger"
)

type testWriteSyncer struct {
	strings.Builder
}

func (w *testWriteSyncer) Sync() error {
	return nil
}

func TestNewGormLoggerNilLoggerFallsBack(t *testing.T) {
	l := NewGormLogger(nil)

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected nil logger fallback, got panic: %v", r)
			}
		}()
		l.Info(t.Context(), "hello %s", "gorm")
	}()
}

func TestGORMLoggerRoutesBySeverity(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	base := zap.New(core)
	l := NewGormLogger(base)

	l.Info(t.Context(), "replacing callback %s", "gorm:query")
	l.Trace(t.Context(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)
	l.Trace(t.Context(), time.Now().Add(-time.Second), func() (string, int64) {
		return "SELECT 2", 1
	}, nil)
	l.Trace(t.Context(), time.Now(), func() (string, int64) {
		return "SELECT 3", 1
	}, errors.New("boom"))

	entries := observed.All()
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	if entries[0].LoggerName != LogKeyDatabaseQuery || entries[0].Level != zapcore.DebugLevel {
		t.Fatalf("expected callback noise to use %s debug, got logger=%q level=%s", LogKeyDatabaseQuery, entries[0].LoggerName, entries[0].Level)
	}
	if entries[1].LoggerName != LogKeyDatabaseQuery || entries[1].Level != zapcore.DebugLevel {
		t.Fatalf("expected normal trace to use %s debug, got logger=%q level=%s", LogKeyDatabaseQuery, entries[1].LoggerName, entries[1].Level)
	}
	if entries[2].LoggerName != LogKeyDatabase || entries[2].Level != zapcore.WarnLevel {
		t.Fatalf("expected slow trace to use %s warn, got logger=%q level=%s", LogKeyDatabase, entries[2].LoggerName, entries[2].Level)
	}
	if entries[3].LoggerName != LogKeyDatabase || entries[3].Level != zapcore.ErrorLevel {
		t.Fatalf("expected error trace to use %s error, got logger=%q level=%s", LogKeyDatabase, entries[3].LoggerName, entries[3].Level)
	}
}

func TestVisibilityCoreHidesQueryLogger(t *testing.T) {
	consoleSink := &testWriteSyncer{}
	consoleCore := zapcore.NewCore(newEncoder(true), consoleSink, zapcore.DebugLevel)
	visibleCore := newVisibilityCore(consoleCore, zapcore.InfoLevel)

	queryEntry := zapcore.Entry{LoggerName: LogKeyDatabaseQuery, Level: zapcore.DebugLevel, Message: "query"}
	if checked := visibleCore.Check(queryEntry, nil); checked != nil {
		t.Fatalf("expected query logger debug entry to be hidden from visibility core")
	}
	if err := visibleCore.Write(queryEntry, nil); err != nil {
		t.Fatalf("expected hidden query logger write to be ignored, got error: %v", err)
	}
	if got := consoleSink.String(); got != "" {
		t.Fatalf("expected no visible output for query logger, got %q", got)
	}

	dbWarnEntry := zapcore.Entry{LoggerName: LogKeyDatabase, Level: zapcore.WarnLevel, Message: "slow query"}
	checked := visibleCore.Check(dbWarnEntry, &zapcore.CheckedEntry{})
	if checked == nil {
		t.Fatalf("expected database warn entry to remain visible")
	}
	if err := visibleCore.Write(dbWarnEntry, nil); err != nil {
		t.Fatalf("expected database warn write to succeed, got error: %v", err)
	}
	if !strings.Contains(consoleSink.String(), "slow query") {
		t.Fatalf("expected visible database warn output, got %q", consoleSink.String())
	}
}

func TestDynamicFileCoreWritesQueryLoggerIntoDatabaseFile(t *testing.T) {
	fileCore := newDynamicFileCore(t.TempDir(), newEncoder(true), map[string]zapcore.Level{
		LogKeyMain:          zapcore.InfoLevel,
		LogKeyDatabase:      zapcore.DebugLevel,
		LogKeyDatabaseQuery: zapcore.DebugLevel,
	}, zapcore.InfoLevel)
	defer func() {
		if err := fileCore.Close(); err != nil {
			t.Errorf("close file core: %v", err)
		}
	}()

	entry := zapcore.Entry{
		LoggerName: LogKeyDatabaseQuery,
		Level:      zapcore.DebugLevel,
		Message:    "SELECT 1",
		Time:       time.Unix(0, 0),
	}
	if err := fileCore.Write(entry, nil); err != nil {
		t.Fatalf("expected query logger write to succeed, got error: %v", err)
	}

	fileCore.mu.RLock()
	defer fileCore.mu.RUnlock()
	if _, exists := fileCore.writerMap[LogKeyDatabaseQuery]; exists {
		t.Fatalf("expected query logger to reuse database file writer, got dedicated writer for %s", LogKeyDatabaseQuery)
	}
	if _, exists := fileCore.writerMap[LogKeyDatabase]; !exists {
		t.Fatalf("expected query logger to write through %s file writer", LogKeyDatabase)
	}
}

var _ gormlogger.Interface = GORMLogger{}
