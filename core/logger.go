package core

import (
	"github.com/natefinch/lumberjack"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var loggerRaw *zap.Logger
var logger *zap.SugaredLogger

func LoggerInit() {
	lumlog := &lumberjack.Logger{
		Filename:   "./record.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,  // number of log files
		MaxAge:     7,  // days
	}

	encoder := getEncoder()
	core := zapcore.NewCore(encoder, zapcore.AddSync(lumlog), zapcore.DebugLevel)

	loggerRaw = zap.New(core, zap.AddCaller())
	defer loggerRaw.Sync() // flushes buffer, if any

	logger = loggerRaw.Sugar()
	logger.Infow("程序启动")
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}
