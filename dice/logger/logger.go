package logger

import (
	"fmt"

	log "sealdice-core/utils/kratos"
)

type LogInfo struct {
	Logger *log.Helper
	WX     *log.WriterX
}

func Init() *LogInfo {
	// KV输出
	loghelper := log.NewHelper(log.GetLogger(), log.WithSprintf(func(format string, a ...interface{}) string {
		return fmt.Sprintf("DICE日志: %s", fmt.Sprintf(format, a...))
	}))
	loghelper.Info("Dice日志开始记录")
	return &LogInfo{
		Logger: loghelper,
		WX:     log.GetWriterX(),
	}
}
