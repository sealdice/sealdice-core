package logger

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestGORMLoggerTraceDebugLogIncludesSQLAndRows(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := NewGormLogger(zap.New(core))
	logger = logger.LogMode(gormlogger.Info).(GORMLogger)
	logger.SkipCallerLookup = true

	logger.Trace(context.Background(), time.Now().Add(-15*time.Millisecond), func() (string, int64) {
		return "SELECT 1", 3
	}, nil)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.DebugLevel {
		t.Fatalf("expected debug level, got %s", entries[0].Level)
	}
	if !strings.Contains(entries[0].Message, "[rows:3]") {
		t.Fatalf("expected rows in message, got %q", entries[0].Message)
	}
	if !strings.Contains(entries[0].Message, "SELECT 1") {
		t.Fatalf("expected sql in message, got %q", entries[0].Message)
	}
}

func TestGORMLoggerTraceWarnsOnSlowSQL(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	logger := NewGormLogger(zap.New(core))
	logger.LogLevel = gormlogger.Warn
	logger.SlowThreshold = 5 * time.Millisecond
	logger.SkipCallerLookup = true

	logger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "SELECT 2", 7
	}, nil)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.WarnLevel {
		t.Fatalf("expected warn level, got %s", entries[0].Level)
	}
	if !strings.Contains(entries[0].Message, "SLOW SQL") {
		t.Fatalf("expected slow sql message, got %q", entries[0].Message)
	}
	if !strings.Contains(entries[0].Message, "SELECT 2") {
		t.Fatalf("expected sql in message, got %q", entries[0].Message)
	}
}

func TestGORMLoggerTraceErrorsOnExecutionError(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	logger := NewGormLogger(zap.New(core))
	logger.LogLevel = gormlogger.Error
	logger.SkipCallerLookup = true

	logger.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "SELECT 3", 1
	}, errors.New("boom"))

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.ErrorLevel {
		t.Fatalf("expected error level, got %s", entries[0].Level)
	}
	if !strings.Contains(entries[0].Message, "boom") {
		t.Fatalf("expected error in message, got %q", entries[0].Message)
	}
	if !strings.Contains(entries[0].Message, "SELECT 3") {
		t.Fatalf("expected sql in message, got %q", entries[0].Message)
	}
}

func TestGORMLoggerTraceIgnoresRecordNotFoundWhenConfigured(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := NewGormLogger(zap.New(core))
	logger.LogLevel = gormlogger.Warn
	logger.IgnoreRecordNotFoundError = true
	logger.SkipCallerLookup = true

	logger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 4", 0
	}, gorm.ErrRecordNotFound)

	if len(recorded.All()) != 0 {
		t.Fatalf("expected no entries, got %d", len(recorded.All()))
	}
}

func TestNewLoggerCoreKeepsDatabaseDebugOutOfConsoleAndUIAtInfoLevel(t *testing.T) {
	tempDir := t.TempDir()
	ui := NewUIWriter()
	var console bytes.Buffer
	core := newLoggerCore(zapcore.InfoLevel, ui, zapcore.AddSync(&console), tempDir)
	log := zap.New(core).Named(LogKeyDatabase)

	log.Debug("sql debug")
	log.Warn("sql warn")
	if err := core.Sync(); err != nil {
		t.Fatalf("sync logger core: %v", err)
	}

	fileContent := readLogFile(t, filepath.Join(tempDir, LogKeyDatabase+".log"))
	if !strings.Contains(fileContent, "sql debug") {
		t.Fatalf("expected debug entry in file, got %q", fileContent)
	}
	if !strings.Contains(fileContent, "sql warn") {
		t.Fatalf("expected warn entry in file, got %q", fileContent)
	}
	if strings.Contains(console.String(), "sql debug") {
		t.Fatalf("expected debug entry hidden from console, got %q", console.String())
	}
	if !strings.Contains(console.String(), "sql warn") {
		t.Fatalf("expected warn entry in console, got %q", console.String())
	}
	if len(ui.Items) != 1 {
		t.Fatalf("expected 1 ui item, got %d", len(ui.Items))
	}
	if ui.Items[0].Msg != "sql warn" {
		t.Fatalf("expected warn item in ui, got %#v", ui.Items[0])
	}
}

func TestNewLoggerCoreShowsDatabaseDebugInConsoleAndUIAtDebugLevel(t *testing.T) {
	tempDir := t.TempDir()
	ui := NewUIWriter()
	var console bytes.Buffer
	core := newLoggerCore(zapcore.DebugLevel, ui, zapcore.AddSync(&console), tempDir)
	log := zap.New(core).Named(LogKeyDatabase)

	log.Debug("sql debug")
	if err := core.Sync(); err != nil {
		t.Fatalf("sync logger core: %v", err)
	}

	fileContent := readLogFile(t, filepath.Join(tempDir, LogKeyDatabase+".log"))
	if !strings.Contains(fileContent, "sql debug") {
		t.Fatalf("expected debug entry in file, got %q", fileContent)
	}
	if !strings.Contains(console.String(), "sql debug") {
		t.Fatalf("expected debug entry in console, got %q", console.String())
	}
	if len(ui.Items) != 1 {
		t.Fatalf("expected 1 ui item, got %d", len(ui.Items))
	}
	if ui.Items[0].Msg != "sql debug" {
		t.Fatalf("expected debug item in ui, got %#v", ui.Items[0])
	}
}

func readLogFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file %s: %v", path, err)
	}
	return string(data)
}
