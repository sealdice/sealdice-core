package uploadgw

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	tusd "github.com/tus/tusd/v2/pkg/handler"
)

const (
	ModuleCensor = "censor"

	// maxCensorUploadBytes 与 api/v2/censor 原上限一致（20 MiB）。
	maxCensorUploadBytes int64 = 20 << 20
)

// censorDestination 收养原 /sd-api/v2/censor/files/upload 的词库文件落盘语义。
type censorDestination struct{}

func (d *censorDestination) Validate(meta map[string]string, size int64) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if !isAllowedWordFile(filename) {
		return tusd.NewError("ERR_INVALID_FILE_TYPE", "仅支持 txt 和 toml 词库文件", http.StatusBadRequest)
	}
	if size > maxCensorUploadBytes {
		return tusd.NewError("ERR_FILE_TOO_LARGE", "词库文件过大", http.StatusBadRequest)
	}
	return nil
}

func (d *censorDestination) Install(_ context.Context, stagedPath string, meta map[string]string) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if err := installFileAtomic(stagedPath, filepath.Join("data", "censor", filename)); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", "写入词库文件失败", http.StatusInternalServerError)
	}
	return nil
}

func isAllowedWordFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt", ".toml":
		return true
	default:
		return false
	}
}
