package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sealcrypto "sealdice-core/utils/crypto"
)

func main() {
	modKey := os.Getenv("SEAL_MOD_PRIVATE_KEY")
	storeKey := os.Getenv("SEAL_STORE_PRIVATE_KEY")

	// 环境变量优先，没有时尝试读取文件夹中的私钥文件
	if modKey == "" {
		keyData, err := os.ReadFile("seal_mod.private.pem")
		if err != nil {
			log.Println("SEAL_MOD_PRIVATE_KEY in env or seal_mod.private.pem not found")
		}
		modKey = string(keyData)
	}
	if storeKey == "" {
		keyData, err := os.ReadFile("seal_store.private.pem")
		if err != nil {
			log.Println("SEAL_STORE_PRIVATE_KEY in env or seal_store.private.pem not found")
		}
		storeKey = string(keyData)
	}

	if len(modKey) != 0 {
		signDir("official", modKey)
	}
	if len(storeKey) != 0 {
		signDir("store", storeKey)
	}
}

func signDir(dir string, privateKey string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		pureName := filepath.Base(entry.Name())
		if entry.IsDir() || strings.HasPrefix(pureName, "[signed]") {
			continue
		}
		// 对文件夹中的 js/json/jsonc/yaml/toml 文件签名
		ext := filepath.Ext(entry.Name())
		switch ext {
		case ".js", ".json", ".jsonc":
			signModFile(dir, privateKey, entry.Name(), pureName, "// sign ")
		case ".yml", ".yaml", ".toml":
			signModFile(dir, privateKey, entry.Name(), pureName, "# sign ")
		}
	}
}

func signModFile(root, privateKey string, path, name string, signStrTmpl string) {
	if privateKey == "" {
		return
	}
	data, _ := os.ReadFile(filepath.Join(root, path))
	data = bytes.Trim(data, "\xef\xbb\xbf")
	if len(data) == 0 {
		log.Printf("%s 文件为空，跳过\n", name)
		return
	}
	sign, err := sealcrypto.RSASign(data, privateKey)
	if err != nil {
		log.Printf("%s 文件签名失败，跳过\n", name)
		return
	}

	dir := filepath.Join(root, "signed")
	_ = os.MkdirAll(dir, 0755)
	newData := []byte(fmt.Sprintln(signStrTmpl+sign) + string(data))
	target := filepath.Join(dir, name)
	if filepath.Ext(target) == ".json" {
		target += "c" // json 转为 jsonc
	}
	err = os.WriteFile(target, newData, 0644)
	if err != nil {
		return
	}
	log.Printf("完成对 %s 的签名，签名后的文件为 [signed]%s\n", name, name)
}
