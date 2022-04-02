package main

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func MainLoggerInit(path string, enableConsoleLog bool) {
	lumlog := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10, // megabytes
		MaxBackups: 3,  // number of log files
		MaxAge:     7,  // days
	}

	encoder := getEncoder()
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(lumlog), zapcore.DebugLevel),
	}

	if enableConsoleLog {
		pe2 := zap.NewProductionEncoderConfig()
		pe2.EncodeTime = zapcore.ISO8601TimeEncoder

		consoleEncoder := zapcore.NewConsoleEncoder(pe2)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel))
	}

	core := zapcore.NewTee(cores...)

	loggerRaw := zap.New(core, zap.AddCaller())
	defer loggerRaw.Sync() // flushes buffer, if any

	logger = loggerRaw.Sugar()
	logger.Infow("核心日志开始记录")
}

var logger *zap.SugaredLogger

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}
