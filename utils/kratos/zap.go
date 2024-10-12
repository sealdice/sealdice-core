package log

import (
	"encoding/json"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 搬运过来WriterX，然后默认初始化，给一个方式获取那个WriterX
// TODO：或许有更好的方案，但是现在我没什么想法

var logLimitDefault int64 = 100

type LogItem struct {
	Level  string  `json:"level"`
	TS     float64 `json:"ts"`
	Caller string  `json:"caller"`
	Msg    string  `json:"msg"`
}

type WriterX struct {
	LogLimit int64
	Items    []*LogItem
}

func (w *WriterX) Write(p []byte) (n int, err error) {
	var a LogItem
	err2 := json.Unmarshal(p, &a)
	if err2 == nil {
		w.Items = append(w.Items, &a)
		limit := w.LogLimit
		if limit == 0 {
			w.LogLimit = logLimitDefault
		}
		if len(w.Items) > int(limit) {
			w.Items = w.Items[1:]
		}
	}
	return len(p), nil
}

// InitZapWithKartosLog 将所有的信息都会输出到main.log，以及输出到控制台
var enabledLevel = zap.InfoLevel

func SetEnableLevel(level zapcore.Level) {
	switch level {
	case zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel,
		zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		{
			enabledLevel = level
		}
	default: // no-op
	}
}

func InitZapWithKartosLog(level zapcore.Level) {
	SetEnableLevel(level)
	// 日志文件的路径
	path := "./data/main.log"

	// 使用lumberjack进行日志文件轮转配置
	lumlog := &lumberjack.Logger{
		Filename:   path, // 日志文件的名称和路径
		MaxSize:    10,   // 每个日志文件最大10MB
		MaxBackups: 3,    // 最多保留3个旧日志文件
		MaxAge:     7,    // 日志文件保存7天
	}

	// 获取日志编码器，定义日志的输出格式
	encoder := getEncoder()

	// 输出到UI的配置部分
	pe := zap.NewProductionEncoderConfig()
	global.wx = &WriterX{}

	// 创建日志核心，将日志写入lumberjack的文件中，并设置日志级别为Debug
	cores := []zapcore.Core{
		// 默认输出到main.log的，全量日志文件
		zapcore.NewCore(encoder, zapcore.AddSync(lumlog), zapcore.DebugLevel),
		// 默认输出到UI的，只输出Info级别
		// This outputs to WebUI, DO NOT apply enabledLevel
		zapcore.NewCore(zapcore.NewJSONEncoder(pe), zapcore.AddSync(global.wx), zapcore.InfoLevel),
	}

	stdOutencoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建控制台的日志编码器（以较友好的格式显示日志）
	consoleEncoder := zapcore.NewConsoleEncoder(stdOutencoderConfig)

	// 将控制台输出作为另一个日志核心，日志级别为Info
	cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), enabledLevel))

	// 将多个日志核心组合到一起，以同时记录到文件和控制台
	core := zapcore.NewTee(cores...)

	// 创建带有调用者信息的日志记录器，注意跳过两层，这样就能正常提供给log
	loggerRaw := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	// 设置全局日志记录器
	global.SetLogger(NewZapLogger(loggerRaw))
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}
