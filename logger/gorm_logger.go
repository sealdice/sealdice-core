package logger

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	infoStr      = "[info] "
	warnStr      = "[warn] "
	errStr       = "[error] "
	traceStr     = "[%.3fms] [rows:%v] %s"
	traceWarnStr = "%s\n[%.3fms] [rows:%v] %s"
	traceErrStr  = "%s\n[%.3fms] [rows:%v] %s"
)

const (
	maxGORMLogParamBytes        = 10 * 1024
	maxGORMLogTextPrefixBytes   = 1024
	maxGORMLogBinaryPrefixBytes = 64
	maxGORMLogSQLBytes          = 10 * 1024
)

type ContextFn func(ctx context.Context) []zapcore.Field

type GORMLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
	SkipCallerLookup          bool
	Context                   ContextFn
}

func NewGormLogger(zapLogger *zap.Logger) GORMLogger {
	return GORMLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  gormlogger.Info,
		SlowThreshold:             300 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
		Context:                   nil,
	}
}

func (l GORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return GORMLogger{
		ZapLogger:                 l.ZapLogger,
		SlowThreshold:             l.SlowThreshold,
		LogLevel:                  level,
		SkipCallerLookup:          l.SkipCallerLookup,
		IgnoreRecordNotFoundError: l.IgnoreRecordNotFoundError,
		Context:                   l.Context,
	}
}

func (l GORMLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel < gormlogger.Info {
		return
	}
	// GORM 内部回调替换日志属于查询噪声，只写入 database.log。
	if strings.Contains(msg, "replacing callback") {
		l.queryLogger(ctx).Sugar().Debugf(infoStr+msg, args...)
		return
	}
	l.logger(ctx).Sugar().Infof(infoStr+msg, args...)
}

func (l GORMLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel < gormlogger.Warn {
		return
	}
	l.logger(ctx).Sugar().Warnf(warnStr+msg, args...)
}

func (l GORMLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel < gormlogger.Error {
		return
	}
	l.logger(ctx).Sugar().Errorf(errStr+msg, args...)
}

// ParamsFilter bounds values before GORM interpolates them into a log-only SQL
// string. The original parameters used to execute the statement are untouched.
func (l GORMLogger) ParamsFilter(_ context.Context, sql string, params ...interface{}) (string, []interface{}) {
	filtered := params
	for index, param := range params {
		var replacement string
		switch value := param.(type) {
		case string:
			if len(value) <= maxGORMLogParamBytes {
				continue
			}
			replacement = fmt.Sprintf("%s...[truncated original_bytes=%d]", utf8BytePrefix(value, maxGORMLogTextPrefixBytes), len(value))
		case []byte:
			if len(value) <= maxGORMLogParamBytes {
				continue
			}
			prefixLength := min(len(value), maxGORMLogBinaryPrefixBytes)
			replacement = fmt.Sprintf("<bytes prefix_hex=%x original_bytes=%d>", value[:prefixLength], len(value))
		default:
			continue
		}

		if len(filtered) > 0 && &filtered[0] == &params[0] {
			filtered = append([]interface{}(nil), params...)
		}
		filtered[index] = replacement
	}
	return sql, filtered
}

func utf8BytePrefix(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	prefix := value[:maxBytes]
	for len(prefix) > 0 && !utf8.ValidString(prefix) {
		prefix = prefix[:len(prefix)-1]
	}
	return prefix
}

func truncateGORMLogSQL(sql string) string {
	if len(sql) <= maxGORMLogSQLBytes {
		return sql
	}
	suffix := fmt.Sprintf("...[truncated original_bytes=%d]", len(sql))
	return utf8BytePrefix(sql, maxGORMLogSQLBytes-len(suffix)) + suffix
}

func boundedTraceSQL(fc func() (string, int64)) (string, int64) {
	sql, rows := fc()
	return truncateGORMLogSQL(sql), rows
}

func (l GORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	logger := l.logger(ctx)

	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := boundedTraceSQL(fc)
		if rows == -1 {
			logger.Sugar().Errorf(traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			return
		}
		logger.Sugar().Errorf(traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn:
		sql, rows := boundedTraceSQL(fc)
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			logger.Sugar().Warnf(traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			return
		}
		logger.Sugar().Warnf(traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	case l.LogLevel >= gormlogger.Info:
		sql, rows := boundedTraceSQL(fc)
		queryLogger := l.queryLogger(ctx)
		if rows == -1 {
			queryLogger.Sugar().Debugf(traceStr, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			return
		}
		queryLogger.Sugar().Debugf(traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}

var (
	gormPackage = filepath.Join("gorm.io", "gorm")
)

func (l GORMLogger) logger(ctx context.Context) *zap.Logger {
	return l.namedLogger(ctx, LogKeyDatabase)
}

func (l GORMLogger) queryLogger(ctx context.Context) *zap.Logger {
	return l.namedLogger(ctx, LogKeyDatabaseQuery)
}

func (l GORMLogger) namedLogger(ctx context.Context, name string) *zap.Logger {
	logger := l.baseLogger()
	logger = logger.Named(name)
	if l.Context != nil {
		fields := l.Context(ctx)
		logger = logger.With(fields...)
	}

	if l.SkipCallerLookup {
		return logger
	}

	for i := 2; i < 15; i++ {
		_, file, _, ok := runtime.Caller(i)
		switch {
		case !ok:
		case strings.HasSuffix(file, "_test.go"):
		case strings.Contains(file, gormPackage):
		default:
			return logger.WithOptions(zap.AddCallerSkip(i))
		}
	}
	return logger
}

func (l GORMLogger) baseLogger() *zap.Logger {
	if l.ZapLogger != nil {
		return l.ZapLogger
	}
	return zap.L()
}
