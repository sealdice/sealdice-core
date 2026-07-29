package dice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureRestoreTargetSafe(target string) error {
	clean := filepath.Clean(target)
	relative, err := filepath.Rel("data", clean)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("恢复目标不在 data 目录内: %s", target)
	}
	parts := []string{"data"}
	if relative != "." {
		parts = append(parts, strings.Split(relative, string(filepath.Separator))...)
	}
	current := ""
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("检查恢复目标 %s: %w", current, statErr)
		}
		if isLinkedRestorePath(info) {
			return fmt.Errorf("恢复目标包含符号链接或重解析点: %s", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("恢复目标的父路径不是目录: %s", current)
		}
	}
	return nil
}
