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
		hash := md5.Sum(data) //nolint:gosec
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
		logger.Error(s)
	}
}

func (i *ExtInfo) FileWrite(name string, mod string, ctx string) {
	if !isNotABS(name, i.dice.Logger) {
		// 如果不是绝对路径返回 true
		// 取否一下
		return
		// 拒绝向下执行
	}
	reg := regexp.MustCompile(`/`)
	path := filepath.Join("data", "default", "extensions", i.Name)
	// 如果检测到分隔符，单独处理
	if reg.MatchString(name) {
		err := os.MkdirAll(path+"/"+filepath.Dir(name), 0777)
		if err != nil {
			fmt.Println("非法路径:", err)
			return
		}
		// 继续执行
	} else {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			fmt.Println("非法路径:", err)
			return
		}
	}
	if mod == "append" || mod == "add" {
		file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY, 0777) // 创建或打开文件
		if err != nil {
			i.dice.Logger.Errorf("创建文件出错%s", err.Error())
			return
		}
		defer func(file *os.File) {
			errClose := file.Close()
			if errClose != nil {
				i.dice.Logger.Errorf("关闭文件出错%s", errClose.Error())
			}
		}(file)
		_, err = file.WriteString(ctx) // 将内容写入文件
		if err != nil {
			i.dice.Logger.Errorf("写入文件出错:%s", err)
			return
		}
	} else if mod == "trunc" || mod == "overwrite" {
		file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777) // 创建或打开文件
		if err != nil {
			i.dice.Logger.Errorf("创建文件出错%s", err.Error())
			return
		}
		defer func(file *os.File) {
			errClose := file.Close()
			if errClose != nil {
				i.dice.Logger.Errorf("关闭文件出错%s", errClose.Error())
			}
		}(file)
		_, err = file.WriteString(ctx) // 将内容写入文件
		if err != nil {
			i.dice.Logger.Errorf("写入文件出错:%s", err)
			return
		}
	} else {
		i.dice.Logger.Errorf("未知模式，允许使用模式追加: add||append,覆写: trunc||overwrite")
		return
	}
}
func (i *ExtInfo) FileRead(name string) string {
	if !isNotABS(name, i.dice.Logger) {
		// 如果不是绝对路径返回 true
		// 取否一下
		return ""
		// 拒绝向下执行
	}
	path := filepath.Join("data", "default", "extensions", i.Name, name)
	content, err := os.ReadFile(path)
	if err != nil {
		i.dice.Logger.Errorf("打开文件出错%s", err.Error())
	}
	return string(content)
}

func (i *ExtInfo) FileExists(name string) bool {
	if !isNotABS(name, i.dice.Logger) {
		// 如果不是绝对路径返回 true
		// 取否一下
		return false
		// 拒绝向下执行
	}
	path := filepath.Join("data", "default", "extensions", i.Name, name)
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func isNotABS(name string, logger *zap.SugaredLogger) bool {
	if filepath.IsAbs(name) {
		// 如果路径为绝对路径
		// 拒绝执行
		logger.Error("出于安全原因，拒绝文件通过绝对路径调用，请使用文件名称或相对路径+文件名称调用")
		return false
	}
	filelist := filepath.SplitList(name)
	for i := 0; i < len(filelist); i++ {
		re := regexp.MustCompile(`^\.+`)
		if re.MatchString(filelist[i]) {
			// 被匹配到了
			// 存在访问父级目录的嫌疑
			// 返回假
			logger.Error("存在访问父级目录的嫌疑，拒绝执行")
			return false
		}
	}
	return true
}

func (i *ExtInfo) FileDelete(name string) {
	path := filepath.Join("data", "default", "extensions", i.Name, name)
	err := os.Remove(path)
	if err != nil {
		i.dice.Logger.Error(err.Error())
	}
}
