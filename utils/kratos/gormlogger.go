package log

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
	gormlogger "gorm.io/gorm/logger"
)

// gorm的格式化字符串抄过来
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
	// 要被传入的KartosLogger
	ZapLogger *Helper
	// 原本的Logger有
	LogLevel gormlogger.LogLevel
	// 原本的logger有
	SlowThreshold time.Duration
	// 原本的logger有
	IgnoreRecordNotFoundError bool
	// logger缺少的
	ParameterizedQueries bool
	SkipCallerLookup     bool
	Context              ContextFn
}

func NewGormLogger(zapLogger *Helper) GORMLogger {
	return GORMLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  gormlogger.Warn,
		SlowThreshold:             100 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
		Context:                   nil,
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
	}
}

func (l GORMLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Infof(infoStr+msg, args...)
	}
}

func (l GORMLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Warnf(warnStr+msg, args...)
	}
}

func (l GORMLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ZapLogger.Errorf(errStr+msg, args...)
	}
}

func (l GORMLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!errors.Is(err, gormlogger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.ZapLogger.Errorf(traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.ZapLogger.Errorf(traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.ZapLogger.Warnf(traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.ZapLogger.Warnf(traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == gormlogger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.ZapLogger.Debugf(traceStr, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.ZapLogger.Debugf(traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
