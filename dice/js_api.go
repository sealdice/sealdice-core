//nolint:gosec
package dice

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"go.uber.org/zap"
)

func Base64ToImageFunc(logger *zap.SugaredLogger) func(string) string {
	return func(b64 string) string {
		// use logger here
		// 解码 Base64 值
		data, e := base64.StdEncoding.DecodeString(b64)
		if e != nil {
			logger.Errorf("不合法的base64值：%s", b64)
			// 出现错误，拒绝向下执行
			return ""
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
			errClose := fi.Close()
			if errClose != nil {
				logger.Errorf("关闭文件出错%s", errClose.Error())
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

func Log(logger *zap.SugaredLogger) func(string) {
	return func(s string) {
		logger.Info(s)
	}
}
func ErrorLog(logger *zap.SugaredLogger) func(string) {
	return func(s string) {
		logger.Errorf(s)
	}
}

func FileWrite(logger *zap.SugaredLogger) func(ei *ExtInfo, name string, ctx string) {
	return func(ei *ExtInfo, name string, ctx string) {
		re := regexp.MustCompile(`^[A-Z]:`)
		if re.MatchString(name) {
			logger.Errorf("出于安全原因，拒绝文件通过绝对路径调用，请使用文件名称或相对路径+文件名称调用，使用相对路径时不要用\".\\\"")
			return
		}
		// 没有办法获取插件名称，强制把 ExtInfo 塞进去
		// 出于安全，仅允许 js 插件文件 io 限制在 default/extensions/<ext> 文件夹

		path := filepath.Join("data", "default", "extensions", ei.Name)
		path = filepath.ToSlash(path)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			fmt.Println("非法路径:", err)
			return
		}

		file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY, 0644) // 创建或打开文件
		if err != nil {
			logger.Errorf("创建文件出错%s", err.Error())
			return
		}
		defer func(file *os.File) {
			errClose := file.Close()
			if errClose != nil {
				logger.Errorf("关闭文件出错%s", errClose.Error())
			}
		}(file)

		_, err = file.WriteString(ctx) // 将内容写入文件
		if err != nil {
			logger.Errorf("写入文件出错:%s", err)
			return
		}
	}
}
