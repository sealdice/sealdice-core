package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	key := os.Getenv("SEAL_MOD_PRIVATE_KEY")
	if key == "" {
		// 环境变量优先，没有时尝试读取文件夹中的私钥文件
		keyData, err := os.ReadFile("seal_mod.private.pem")
		if err != nil {
			log.Println("SEAL_MOD_PRIVATE_KEY in env or seal_mod.private.pem not found")
			return
		}
		key = string(keyData)
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		pureName := filepath.Base(entry.Name())
		if entry.IsDir() || strings.HasPrefix(pureName, "[signed]") {
			continue
		}
		// 对文件夹中的 js/json/yaml/toml 文件签名
		ext := filepath.Ext(entry.Name())
		switch ext {
		case ".js", ".json":
			signModFile(key, entry.Name(), pureName, "// sign %s\n")
		case ".yml", ".yaml", ".toml":
			signModFile(key, entry.Name(), pureName, "# sign %s\n")
		}
	}
}

func signModFile(privateKey string, path, name string, signStrTmpl string) {
	if privateKey == "" {
		return
	}
	data, _ := os.ReadFile(path)
	sign, err := rsaSign(data, privateKey)
	if err != nil {
		return
	}
	newData := []byte(fmt.Sprintf(signStrTmpl, sign))
	err = os.WriteFile("[signed]"+name, newData, 0644)
	if err != nil {
		return
	}
	log.Printf("完成对 %s 的签名，签名后的文件为 [signed]%s\n", name, name)
}

// rsaSign 签名
func rsaSign(data []byte, privateKey string) (string, error) {
	key := readPrivateKey(privateKey)
	hashed := calculateSHA512(data)
	sign, err := rsa.SignPSS(rand.Reader, key, crypto.SHA512, hashed, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}

func calculateSHA512(data []byte) []byte {
	hashInstance := crypto.SHA512.New()
	hashInstance.Write(data)
	hashed := hashInstance.Sum(nil)
	return hashed
}

// readPrivateKey 读取私钥
func readPrivateKey(privateKey string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil
	}
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil
	}
	key := privateKeyInterface.(*rsa.PrivateKey)
	return key
}
