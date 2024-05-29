package dice

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/Milly/go-base2048"
	"github.com/vmihailenco/msgpack"

	"sealdice-core/utils/crypto"
)

var (
	// SealTrustedClientPrivateKey 可信客户端私钥
	SealTrustedClientPrivateKey = ``
)

func initVerify() {
	// 优先读取环境变量中的可信客户端私钥
	key := os.Getenv("SEAL_TRUSTED_PRIVATE_KEY")
	if len(key) > 0 {
		SealTrustedClientPrivateKey = key
	} else {
		fmt.Println("SEAL_TRUSTED_PRIVATE_KEY not found, maybe in development mode")
	}
}

type payload struct {
	Version   string `msgpack:"version,omitempty"`
	Timestamp int64  `msgpack:"timestamp,omitempty"`
	Platform  string `msgpack:"platform,omitempty"`
	Uid       string `msgpack:"uid,omitempty"`
	Username  string `msgpack:"username,omitempty"`
}

type data struct {
	Payload payload `msgpack:"payload,omitempty"`
	Sign    []byte  `msgpack:"sign,omitempty"`
}

// GenerateVerificationCode 生成海豹校验码
func GenerateVerificationCode(platform string, userID string, username string, useBase64 bool) string {
	if len(SealTrustedClientPrivateKey) == 0 {
		return ""
	}
	// 海豹校验码格式：SEAL<data>
	p := payload{
		Version:   VERSION.String(),
		Timestamp: time.Now().Unix(),
		Platform:  platform,
		Uid:       userID,
		Username:  username,
	}
	pp, _ := msgpack.Marshal(p)
	sign, err := crypto.EcdsaSignRow(pp, SealTrustedClientPrivateKey)
	if err != nil {
		return ""
	}

	d := data{
		Payload: p,
		Sign:    sign,
	}
	dp, _ := msgpack.Marshal(d)
	if useBase64 {
		return fmt.Sprintf("SEAL-%s", base64.StdEncoding.EncodeToString(dp))
	} else {
		return fmt.Sprintf("SEAL%s", base2048.DefaultEncoding.EncodeToString(dp))
	}
}
