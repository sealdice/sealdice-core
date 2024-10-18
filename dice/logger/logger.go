package logger

import (
	log "sealdice-core/utils/kratos"
)

type LogInfo struct {
	Logger *log.Helper
	WX     *log.WriterX
}

func Init() *LogInfo {
	// KV输出
	loghelper := log.NewCustomHelper(log.LOG_DICE, false, nil)
	loghelper.Info("Dice日志开始记录")
	return &LogInfo{
		Logger: loghelper,
		WX:     log.GetWriterX(),
	}
}
