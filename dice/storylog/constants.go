package storylog

type StoryVersion int

// 日志导出时使用的常量
const (
	ExportTxtFilename     = "raw-log.txt"
	ExportJsonFilename    = "sealdice-standard-log.json"
	ExportParquetFilename = "sealdice-standard-log.parquet"
	ExportReadmeFilename  = "README.txt"
	ExportReadmeContent   = ExportTxtFilename + ": 纯文本 Log\n" + ExportJsonFilename + ": 海豹标准 Log, 粘贴到染色器可格式化。若是Parquet的格式，暂时无法处理。\n"

	StoryVersionV1   StoryVersion = 101
	StoryVersionV105 StoryVersion = 105

	StoryVersionV1Str   = "V1"
	StoryVersionV105Str = "V1.5"
)
