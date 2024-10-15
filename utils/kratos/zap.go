package log

import (
	"encoding/json"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"moul.io/zapfilter"
)

// TODO：或许有更好的方案,目前只是保证能够使用了
// 搬运过来WriterX，然后默认初始化，给一个方式获取那个WriterX

var logLimitDefault int64 = 100
var originZapLogger *zap.Logger

// GetLoggerRaw 特殊情况下，获取原生的LOGGER进行处理
func GetLoggerRaw() *zap.Logger {
	return originZapLogger
}

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

// InitZapWithKartosLog 将所有的信息都会输出到main.log，以及输出到控制台
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
	// 输出到文件的配置部分
	mainLogCore := zapcore.NewCore(encoder, zapcore.AddSync(lumlog), zapcore.DebugLevel)
	// 创建控制台的日志编码器
	consoleCoreRaw := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), enabledLevel)
	// 适配隐藏控制台输出的部分，重新设置日志级别，并输出除了HIDE以外的所有情况。这里ByNamespaces注意要先定义”全部选择“，然后定义”HIDE的不要“。
	consoleCore := zapfilter.NewFilteringCore(consoleCoreRaw, zapfilter.All(zapfilter.MinimumLevel(enabledLevel), zapfilter.ByNamespaces("*,-HIDE.*")))

	// 创建日志核心，将日志写入lumberjack的文件中，并设置日志级别为Debug
	cores := []zapcore.Core{
		// 默认输出到main.log的，全量日志文件
		mainLogCore,
		// 默认输出到UI的，只输出Info级别
		// This outputs to WebUI, DO NOT apply enabledLevel
		zapcore.NewCore(zapcore.NewJSONEncoder(pe), zapcore.AddSync(global.wx), zapcore.InfoLevel),
		consoleCore,
	}

	// 将多个日志核心组合到一起，以同时记录到文件和控制台
	core := zapcore.NewTee(cores...)

	// 创建带有调用者信息的日志记录器，注意跳过两层，这样就能正常提供给log
	originZapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	// 设置全局日志记录器，默认全局记录器为SEAL命名空间
	global.SetLogger(NewZapLogger(originZapLogger.Named(LOG_SEAL)))
	Info("海豹全局日志核心已启动。当海豹出现非闪退问题时，可考虑提供data/main.log至开发者或提交至ISSUE.")
}

func GetWebLogger() *Helper {
	webpath := "./data/web.log"
	weblumlog := &lumberjack.Logger{
		Filename:   webpath, // 日志文件的名称和路径
		MaxSize:    10,      // 每个日志文件最大10MB
		MaxBackups: 3,       // 最多保留3个旧日志文件
		MaxAge:     7,       // 日志文件保存7天
	}
	webCore := zapcore.NewCore(getEncoder(), zapcore.AddSync(weblumlog), zapcore.DebugLevel)
	webZapLogger := zap.New(webCore, zap.WithCaller(false))
	return NewHelper(NewZapLogger(webZapLogger.Named("WEB")))
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}
