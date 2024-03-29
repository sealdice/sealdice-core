//nolint:gosec
package dice

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func Base64ToImage(base64value string) string {
	// 解码 Base64 值
	data, err := base64.StdEncoding.DecodeString(base64value)
	if err != nil {
		fmt.Println("Error decoding Base64:", err)
	}
	// 计算 MD5 哈希值作为文件名
	hash := md5.Sum(data)
	filename := fmt.Sprintf("%x", hash)
	// 获取临时目录路径
	envVarName := "SystemRoot"
	// 通过 os 包中的 Getenv 函数读取环境变量的值
	envVarValue := os.Getenv(envVarName)
	tempDir := envVarValue + "\\Temp"
	// 构建文件路径
	filePath := filepath.Join(tempDir, filename+".png")
	filePath = filepath.ToSlash(filePath)
	// 将数据写入文件
	fi, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		fmt.Println(err)
	}
	defer fi.Close()
	_, err = fi.WriteString(string(data))
	if err != nil {
		fmt.Println("Error writing file:", err)
	}
	fmt.Println("File saved to:", filePath)
	return "file://" + filePath
}
