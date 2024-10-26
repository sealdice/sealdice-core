// copied from github.com/go-kratos/kratos/contrib/log/zap/v2
package log

import (
	"fmt"

	"go.uber.org/zap"
)

var _ Logger = (*ZapLogger)(nil)

type ZapLogger struct {
	log    *zap.Logger
	msgKey string
}

func NewZapLogger(zlog *zap.Logger) *ZapLogger {
	return &ZapLogger{
		log:    zlog,
		msgKey: DefaultMessageKey,
	}
}

// 保留给Helper使用

type ZapOption func(*ZapLogger)

// WithZapMessageKey with message key.
func WithZapMessageKey(key string) ZapOption {
	return func(l *ZapLogger) {
		l.msgKey = key
	}
}

func (l *ZapLogger) Log(level Level, keyvals ...interface{}) error {
	var (
		msg    = ""
		keylen = len(keyvals)
	)
	if keylen == 0 || keylen%2 != 0 {
		l.log.Warn(fmt.Sprint("Keyvalues must appear in pairs: ", keyvals))
		return nil
	}

	data := make([]zap.Field, 0, (keylen/2)+1)
	for i := 0; i < keylen; i += 2 {
		if keyvals[i].(string) == l.msgKey {
			msg, _ = keyvals[i+1].(string)
			continue
		}
		data = append(data, zap.Any(fmt.Sprint(keyvals[i]), keyvals[i+1]))
	}

	switch level {
	case LevelDebug:
		l.log.Debug(msg, data...)
	case LevelInfo:
		l.log.Info(msg, data...)
	case LevelWarn:
		l.log.Warn(msg, data...)
	case LevelError:
		l.log.Error(msg, data...)
	case LevelFatal:
		l.log.Fatal(msg, data...)
	}
	return nil
}

func (l *ZapLogger) Sync() error {
	return l.log.Sync()
}

func (l *ZapLogger) Close() error {
	return l.Sync()
}
