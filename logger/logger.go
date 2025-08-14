package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"moul.io/zapfilter"
)

const (
	LogKeyMain     = "main"
	LogKeyDatabase = "database"
	LogKeyWeb      = "web"
	LogKeyAdapter  = "adapter"
)

func InitLogger(level zapcore.Level, ui *UIWriter) *zap.SugaredLogger {
	consoleEncoder := newEncoder(true)
	jsonEncoder := newEncoder(false)

	consoleWriter := zapfilter.NewFilteringCore(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		zapfilter.All(
			zapfilter.MinimumLevel(level),
			zapfilter.ByNamespaces("*,-database"),
		),
	)
	uiWriter := zapfilter.NewFilteringCore(
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(ui), zapcore.InfoLevel),
		zapfilter.ByNamespaces("*,-database"),
	)
	core := zapcore.NewTee(
		newDynamicFileCore("data", level, consoleEncoder),
		consoleWriter,
		uiWriter,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	zap.ReplaceGlobals(logger)

	newGormLogger(logger.Sugar().Named(LogKeyDatabase)).SetAsDefault()
	return logger.Sugar()
}

func M() *zap.SugaredLogger {
	return zap.S().Named(LogKeyMain)
}

type dynamicFileCore struct {
	zapcore.LevelEnabler
	rootDir     string
	encoder     zapcore.Encoder
	mu          sync.RWMutex
	writerMap   map[string]zapcore.WriteSyncer
	defaultName string
}

var _ zapcore.Core = (*dynamicFileCore)(nil)

func newDynamicFileCore(rootDir string, enabler zapcore.LevelEnabler, encoder zapcore.Encoder) *dynamicFileCore {
	return &dynamicFileCore{
		LevelEnabler: enabler,
		rootDir:      rootDir,
		encoder:      encoder,
		writerMap:    make(map[string]zapcore.WriteSyncer),
		defaultName:  LogKeyMain,
	}
}

func (c *dynamicFileCore) With(_ []zapcore.Field) zapcore.Core {
	return c // not implemented but not needed currently
}

func (c *dynamicFileCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return nil
}

func (c *dynamicFileCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var loggerName string
	if entry.LoggerName != "" {
		loggerName = entry.LoggerName
	} else {
		loggerName = c.defaultName
	}
	loggerName = strings.ToLower(loggerName)

	c.mu.RLock()
	writer, ok := c.writerMap[loggerName]
	c.mu.RUnlock()

	if !ok {
		c.mu.Lock()
		writer, ok = c.writerMap[loggerName]
		if !ok {
			logFile := filepath.Join(c.rootDir, loggerName+".log")
			important := true
			if loggerName == LogKeyWeb {
				important = false
			}
			logWriter := newLumberjackWriter(logFile, important)
			writer = zapcore.AddSync(logWriter)
			c.writerMap[loggerName] = writer
		}
		c.mu.Unlock()
	}

	buf, err := c.encoder.EncodeEntry(entry, fields)
	if err != nil {
		return err
	}
	_, err = writer.Write(buf.Bytes())
	buf.Free()
	return err
}

func (c *dynamicFileCore) Sync() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var err error
	for _, writer := range c.writerMap {
		if e := writer.Sync(); e != nil {
			err = e
		}
	}
	return err
}

func newLumberjackWriter(filepath string, important bool) io.Writer {
	if important {
		return &lumberjack.Logger{
			Filename:   filepath,
			MaxSize:    10, // 每个日志文件最大 10 MB
			MaxBackups: 3,  // 最多保留 3 个旧日志文件
			MaxAge:     7,  // 日志文件保存 7 天
		}
	}
	return &lumberjack.Logger{
		Filename:   filepath,
		MaxSize:    5, // 每个日志文件最大 5 MB
		MaxBackups: 3, // 最多保留 3 个旧日志文件
		MaxAge:     3, // 日志文件保存 3 天
	}
}

func newEncoder(console bool) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.NameKey = "module"
	encoderConfig.EncodeName = zapcore.FullNameEncoder
	if console {
		return zapcore.NewConsoleEncoder(encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}
