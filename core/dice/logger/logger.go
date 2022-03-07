package logger

import (
	"encoding/json"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type LogItem struct {
	Level  string  `json:"level"`
	Ts     float64 `json:"ts"`
	Caller string  `json:"caller"`
	Msg    string  `json:"msg"`
}

type WriterX struct {
	Items []*LogItem
}

type LogInfo struct {
	LoggerRaw *zap.Logger
	Logger    *zap.SugaredLogger
	WX        *WriterX
}

func (w *WriterX) Write(p []byte) (n int, err error) {
	var a LogItem
	err2 := json.Unmarshal(p, &a)
	if err2 == nil {
		w.Items = append(w.Items, &a)
		if len(w.Items) > 32 {
			w.Items = w.Items[16:]
		}
	}
	return len(p), nil
}

func LoggerInit(path string, name string, enableConsoleLog bool) *LogInfo {
	lumlog := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10, // megabytes
		MaxBackups: 3,  // number of log files
		MaxAge:     7,  // days
	}

	encoder := getEncoder()

	pe := zap.NewProductionEncoderConfig()
	wx := &WriterX{}

	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(lumlog), zapcore.DebugLevel),
		zapcore.NewCore(zapcore.NewJSONEncoder(pe), zapcore.AddSync(wx), zapcore.InfoLevel),
	}

	if enableConsoleLog {
		pe2 := zap.NewProductionEncoderConfig()
		pe2.EncodeTime = zapcore.ISO8601TimeEncoder

		consoleEncoder := zapcore.NewConsoleEncoder(pe2)
		consoleEncoder.AddString("dice", name)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel))
	}

	core := zapcore.NewTee(cores...)

	loggerRaw := zap.New(core, zap.AddCaller())
	defer loggerRaw.Sync() // flushes buffer, if any

	logger := loggerRaw.Sugar()
	logger.Infow("程序启动")

	return &LogInfo{
		LoggerRaw: loggerRaw,
		Logger:    logger,
		WX:        wx,
	}
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}
