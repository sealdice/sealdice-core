package dice

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "sealdice-core/utils/kratos"
)

func Base64ToImageFunc(logger *log.Helper) func(string) (string, error) {
	return func(b64 string) (string, error) {
		// 解码 Base64 值
		data, e := base64.StdEncoding.DecodeString(b64)
		if e != nil {
			// 出现错误，拒绝向下执行
			return "", errors.New("不合法的base64值：" + b64)
		}
		// 计算 MD5 哈希值作为文件名
		hash := md5.Sum(data) //nolint:gosec
		filename := fmt.Sprintf("%x", hash)
		tempDir := os.TempDir()
		// 构建文件路径
		imageurlPath := filepath.Join(tempDir, filename)
		imageurlPath = filepath.ToSlash(imageurlPath)
		// 将数据写入文件
		fi, err := os.OpenFile(imageurlPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
		if err != nil {
			return "", errors.New("创建文件出错:" + err.Error())
		}
		defer func(fi *os.File) {
			errClose := fi.Close()
			if errClose != nil {
				logger.Errorf("关闭文件出错:%s", errClose.Error())
			}
		}(fi)
		_, err = fi.Write(data)
		if err != nil {
			return "", errors.New("写入文件出错:" + err.Error())
		}
		logger.Info("File saved to:", imageurlPath)
		return "file://" + imageurlPath, nil
	}
}
