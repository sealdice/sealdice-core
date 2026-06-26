package logger

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

type ContextFn func(ctx context.Context) []zapcore.Field

type GORMLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
	SkipCallerLookup          bool
	Context                   ContextFn
	JSONFormat                bool
}

func NewGormLogger(zapLogger *zap.Logger) GORMLogger {
	return GORMLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  gormlogger.Warn,
		SlowThreshold:             100 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
		Context:                   nil,
		JSONFormat:                false,
	}
}

func (l GORMLogger) SetAsDefault() {
	gormlogger.Default = l
}

func (l GORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return GORMLogger{
		ZapLogger:                 l.ZapLogger,
		SlowThreshold:             l.SlowThreshold,
		LogLevel:                  level,
		SkipCallerLookup:          l.SkipCallerLookup,
		IgnoreRecordNotFoundError: l.IgnoreRecordNotFoundError,
		Context:                   l.Context,
		JSONFormat:                l.JSONFormat,
	}
}

func (l GORMLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel < gormlogger.Info {
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

func (l GORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	logger := l.logger(ctx)

	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := fc()
		if l.JSONFormat {
			logger.Error("trace", zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
			return
		}
		if rows == -1 {
			logger.Sugar().Errorf(traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			return
		}
		logger.Sugar().Errorf(traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		if l.JSONFormat {
			logger.Warn("trace", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
			return
		}
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			logger.Sugar().Warnf(traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			return
		}
		logger.Sugar().Warnf(traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	case l.LogLevel >= gormlogger.Info:
		sql, rows := fc()
		if l.JSONFormat {
			logger.Debug("trace", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
			return
		}
		if rows == -1 {
			logger.Sugar().Debugf(traceStr, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			return
		}
		logger.Sugar().Debugf(traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}

var (
	gormPackage    = filepath.Join("gorm.io", "gorm")
	zapgormPackage = filepath.Join("moul.io", "zapgorm2")
)

func (l GORMLogger) logger(ctx context.Context) *zap.Logger {
	logger := l.ZapLogger
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
		case strings.Contains(file, zapgormPackage):
		default:
			return logger.WithOptions(zap.AddCallerSkip(i))
		}
	}
	return logger
}
