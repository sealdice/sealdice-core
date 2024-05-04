package dice

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

func Base64ToImageFunc(logger *zap.SugaredLogger) func(string) (string, error) {
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

func ErrorLogFunc(logger *zap.SugaredLogger) func(string) {
	return func(s string) {
		logger.Error(s)
	}
}

func (i *ExtInfo) FileAppend(name string, ctx string) error {
	if !isNotABS(name, i.dice.Logger, i.Name) {
		// 如果不是绝对路径返回 true
		// 取否一下
		return errors.New("")
		// 拒绝向下执行
	}
	reg := regexp.MustCompile(`/`)
	path := filepath.Join("data", "default", "extensions", i.Name)
	// 如果检测到分隔符，单独处理
	if reg.MatchString(name) {
		err := os.MkdirAll(path+"/"+filepath.Dir(name), 0777)
		if err != nil {
			return errors.New("非法路径:" + err.Error())
		}
		// 继续执行
	} else {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return errors.New("非法路径:" + err.Error())
		}
	}
	file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY, 0777) // 创建或打开文件
	if err != nil {
		return errors.New("创建文件出错:" + err.Error())
	}
	defer func(file *os.File) {
		errClose := file.Close()
		if errClose != nil {
			i.dice.Logger.Errorf("关闭文件出错%s", errClose.Error())
		}
	}(file)
	_, err = file.WriteString(ctx) // 将内容写入文件
	if err != nil {
		return errors.New("写入文件出错:" + err.Error())
	}
	return nil
}

func (i *ExtInfo) FileOverwrite(name string, ctx string) error {
	if !isNotABS(name, i.dice.Logger, i.Name) {
		// 如果不是绝对路径返回 true
		// 取否一下
		return errors.New("")
		// 拒绝向下执行
	}
	reg := regexp.MustCompile(`/`)
	path := filepath.Join("data", "default", "extensions", i.Name)
	// 如果检测到分隔符，单独处理
	if reg.MatchString(name) {
		err := os.MkdirAll(path+"/"+filepath.Dir(name), 0777)
		if err != nil {
			return errors.New("非法路径:" + err.Error())
		}
		// 继续执行
	} else {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return errors.New("非法路径:" + err.Error())
		}
	}
	file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777) // 创建或打开文件
	if err != nil {
		return errors.New("创建文件出错:" + err.Error())
	}
	defer func(file *os.File) {
		errClose := file.Close()
		if errClose != nil {
			i.dice.Logger.Errorf("关闭文件出错%s", errClose.Error())
		}
	}(file)
	_, err = file.WriteString(ctx) // 将内容写入文件
	if err != nil {
		return errors.New("写入文件出错:" + err.Error())
	}
	return nil
}

func (i *ExtInfo) FileWrite(name string, mod string, ctx string) error {
	if !isNotABS(name, i.dice.Logger, i.Name) {
		// 如果不是绝对路径返回 true
		// 取否一下
		return errors.New("")
		// 拒绝向下执行
	}
	reg := regexp.MustCompile(`/`)
	path := filepath.Join("data", "default", "extensions", i.Name)
	// 如果检测到分隔符，单独处理
	if reg.MatchString(name) {
		err := os.MkdirAll(path+"/"+filepath.Dir(name), 0777)
		if err != nil {
			return errors.New("非法路径:" + err.Error())
		}
		// 继续执行
	} else {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return errors.New("非法路径:" + err.Error())
		}
	}
	if mod == "append" || mod == "add" {
		file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY, 0777) // 创建或打开文件
		if err != nil {
			return errors.New("创建文件出错:" + err.Error())
		}
		defer func(file *os.File) {
			errClose := file.Close()
			if errClose != nil {
				i.dice.Logger.Errorf("关闭文件出错%s", errClose.Error())
			}
		}(file)
		_, err = file.WriteString(ctx) // 将内容写入文件
		if err != nil {
			return errors.New("写入文件出错:" + err.Error())
		}
	} else if mod == "trunc" || mod == "overwrite" {
		file, err := os.OpenFile(path+"/"+name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777) // 创建或打开文件
		if err != nil {
			return errors.New("创建文件出错:" + err.Error())
		}
		defer func(file *os.File) {
			errClose := file.Close()
			if errClose != nil {
				i.dice.Logger.Errorf("关闭文件出错%s", errClose.Error())
			}
		}(file)
		_, err = file.WriteString(ctx) // 将内容写入文件
		if err != nil {
			return errors.New("写入文件出错:" + err.Error())
		}
	} else {
		return errors.New("未知模式，允许使用模式追加: add||append,覆写: trunc||overwrite")
	}
	return nil
}

func (i *ExtInfo) FileRead(name string) string {
	if !isNotABS(name, i.dice.Logger, i.Name) {
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

func (i *ExtInfo) FileExists(name string) (bool, error) {
	if !isNotABS(name, i.dice.Logger, i.Name) {
		// 如果是绝对路径或者存在越权行为
		return false, errors.New("拒绝执行：存在越权行为或所写路径为绝对路径")
	}
	path := filepath.Join("data", "default", "extensions", i.Name, name)
	_, statErr := os.Stat(path)
	return os.IsNotExist(statErr), nil
}

func isNotABS(name string, logger *zap.SugaredLogger, extname string) bool {
	// 统一在这里处理相对路径转换为绝对路径
	//  不传入 i.Name ，单独开个变量接受
	if filepath.IsAbs(name) {
		// 如果路径为绝对路径
		// 拒绝执行
		logger.Error("出于安全原因，拒绝文件通过绝对路径调用，请使用文件名称或相对路径+文件名称调用")
		return false
	}
	name = "data/default/extensions/" + extname + name
	absolutePath, err := filepath.Abs(name)
	if err != nil {
		return false
	}
	cwd, _ := os.Getwd()
	cwd = filepath.ToSlash(cwd)
	absolutePath = filepath.ToSlash(absolutePath)

	return strings.HasPrefix(absolutePath, cwd+"/data/default/extensions")
}

func (i *ExtInfo) FileDelete(name string) error {
	if !isNotABS(name, i.dice.Logger, i.Name) {
		// 如果是绝对路径或者存在越权行为
		return errors.New("拒绝删除：存在越权行为或所写路径为绝对路径")
	}
	path := filepath.Join("data", "default", "extensions", i.Name, name)
	err := os.Remove(path)
	if err != nil {
		return errors.New("删除文件出错:" + err.Error())
	}
	return nil
}

func (i *ExtInfo) DirDelete(name string) error {
	if !isNotABS(name, i.dice.Logger, i.Name) {
		// 如果是绝对路径或者存在越权行为
		return errors.New("拒绝删除：存在越权行为或所写路径为绝对路径")
	}
	path := filepath.Join("data", "default", "extensions", i.Name, name)
	err := os.RemoveAll(path)
	if err != nil {
		return errors.New("删除文件出错：" + err.Error())
	}

	return nil
}
