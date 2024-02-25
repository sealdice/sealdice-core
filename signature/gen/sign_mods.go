package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sealcrypto "sealdice-core/utils/crypto"
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
	sign, err := sealcrypto.RSASign(data, privateKey)
	if err != nil {
		return
	}
	newData := []byte(fmt.Sprintf(signStrTmpl, sign) + string(data))
	err = os.WriteFile("[signed]"+name, newData, 0o644)
	if err != nil {
		return
	}
	log.Printf("完成对 %s 的签名，签名后的文件为 [signed]%s\n", name, name)
}
