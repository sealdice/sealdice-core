package log

import (
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

func wrapLogger(zaplogger *zap.Logger) zapgorm2.Logger {
	wraper := zapgorm2.New(zaplogger)
	wraper.IgnoreRecordNotFoundError = true
	wraper.LogLevel = gormlogger.Info
	wraper.SkipCallerLookup = true
	return wraper
}
