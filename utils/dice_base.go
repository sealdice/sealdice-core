//nolint:gosec
package utils

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func Base64ToImageFunc(logger *zap.SugaredLogger) func(string) string {
	return func(b64 string) string {
		// use logger here
		// 解码 Base64 值
		data, err := base64.URLEncoding.DecodeString(b64)
		if err != nil {
			logger.Errorf("不合法的base64值：%s", b64)
			// 出现错误，拒绝向下执行
		}
		// 计算 MD5 哈希值作为文件名
		hash := md5.Sum(data)
		filename := fmt.Sprintf("%x", hash)
		tempDir := os.TempDir()
		// 构建文件路径
		imageurlPath := filepath.Join(tempDir, filename)
		imageurlPath = filepath.ToSlash(imageurlPath)
		// 将数据写入文件
		fi, err := os.OpenFile(imageurlPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
		if err != nil {
			logger.Errorf("创建文件出错%s", err.Error())
			return ""
		}
		defer func(fi *os.File) {
			err := fi.Close()
			if err != nil {
				logger.Errorf("创建文件出错%s", err.Error())
			}
		}(fi)
		_, err = fi.Write(data)
		if err != nil {
			logger.Errorf("写入文件出错%s", err.Error())
			return ""
		}
		logger.Info("File saved to:", imageurlPath)
		return "file://" + imageurlPath
	}
}

