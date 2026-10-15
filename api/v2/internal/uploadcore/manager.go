// Package uploadcore 提供上传落盘所需的通用工具。
//
// 分片上传逻辑已迁移到基于 tus 协议的 api/v2/uploadgw，
// 此包现在只保留被多个模块复用的原子替换助手。
package uploadcore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ReplaceFileAtomic 用同目录 staging 文件替换 dstPath：
// 目标不存在时直接 rename；目标存在时先备份、rename 新文件、失败回滚。
func ReplaceFileAtomic(stagedPath string, dstPath string) error {
	if _, err := os.Stat(dstPath); errors.Is(err, os.ErrNotExist) {
		return os.Rename(stagedPath, dstPath)
	} else if err != nil {
		return err
	}

	backup, err := os.CreateTemp(filepath.Dir(dstPath), "."+filepath.Base(dstPath)+".backup-*")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupPath)
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := os.Rename(dstPath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(stagedPath, dstPath); err != nil {
		if rollbackErr := os.Rename(backupPath, dstPath); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("replace upload target: %w", err),
				fmt.Errorf("rollback upload target: %w", rollbackErr),
			)
		}
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}
