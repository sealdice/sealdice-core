package uploadgw

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	tusd "github.com/tus/tusd/v2/pkg/handler"
)

const (
	ModuleResource = "resource"

	// maxResourceUploadBytes 与 api/v2/resource 原上限一致（50 MiB）。
	maxResourceUploadBytes int64 = 50 << 20
)

// resourceDestination 收养原 /sd-api/v2/resource/upload 的图片落盘语义。
type resourceDestination struct{}

func (d *resourceDestination) Validate(meta map[string]string, size int64) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if !isAllowedImageFile(filename) {
		return tusd.NewError("ERR_INVALID_FILE_TYPE", "仅支持图片文件（png/jpg/jpeg/gif）", http.StatusBadRequest)
	}
	if size > maxResourceUploadBytes {
		return tusd.NewError("ERR_FILE_TOO_LARGE", "图片过大", http.StatusBadRequest)
	}
	return nil
}

func (d *resourceDestination) Install(_ context.Context, stagedPath string, meta map[string]string) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if err := installFileAtomic(stagedPath, filepath.Join("data", "images", filename)); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", "写入资源文件失败", http.StatusInternalServerError)
	}
	return nil
}

func isAllowedImageFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	default:
		return false
	}
}
