package constants

// 日志导出时使用的常量
const (
	LogExportTxtFilename    = "raw-log.txt"
	LogExportJsonFilename   = "sealdice-standard-log-page.json"
	LogExportReadmeFilename = "README.txt"
	LogExportReadmeContent  = LogExportTxtFilename + ": 纯文本 Log\n" + LogExportJsonFilename + ": 海豹标准 Log, 粘贴到染色器可格式化\n"
)
