package dice

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Milly/go-base2048"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/crypto/ed25519"

	"sealdice-core/logger"
	"sealdice-core/utils/crypto"
)

var (
	// SealTrustedClientPrivateKey 可信客户端私钥
	SealTrustedClientPrivateKey = ``
	// SealSignClientPrivateKey 拉起 海豹v3 签名用私钥
	SealSignClientPrivateKey = ``
)

func initVerify() {
	log := logger.M()
	// 优先读取环境变量中的可信客户端私钥
	key := os.Getenv("SEAL_TRUSTED_PRIVATE_KEY")
	if len(key) > 0 {
		SealTrustedClientPrivateKey = key
	} else if len(SealTrustedClientPrivateKey) == 0 {
		log.Warn("SEAL_TRUSTED_PRIVATE_KEY not found, maybe in development mode")
	}
	keySign := os.Getenv("SEAL_SIGN_PRIVATE_KEY")
	if len(keySign) > 0 {
		SealSignClientPrivateKey = keySign
	} else if len(SealSignClientPrivateKey) == 0 {
		log.Warn("SEAL_SIGN_PRIVATE_KEY not found, maybe in development mode")
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
	Payload []byte `msgpack:"payload,omitempty"`
	Sign    []byte `msgpack:"sign,omitempty"`
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
		Payload: pp,
		Sign:    sign,
	}
	dp, _ := msgpack.Marshal(d)
	if useBase64 {
		return fmt.Sprintf("SEAL#%s", base64.StdEncoding.EncodeToString(dp))
	} else {
		return fmt.Sprintf("SEAL%%%s", base2048.DefaultEncoding.EncodeToString(dp))
	}
}

type payloadPublicDice struct {
	Version string `msgpack:"version,omitempty"`
	Sign    []byte `msgpack:"sign,omitempty"`
}

func GenerateVerificationKeyForPublicDice(data any) string {
	doEcdsaSign := len(SealTrustedClientPrivateKey) > 0
	pp, _ := msgpack.Marshal(data)

	var sign []byte
	if doEcdsaSign {
		var err error
		sign, err = crypto.EcdsaSignRow(pp, SealTrustedClientPrivateKey)
		if err != nil {
			return ""
		}
	} else {
		h := sha256.New()
		h.Write(pp)
		sign = h.Sum(nil)
	}

	d := payloadPublicDice{
		Version: VERSION.String(),
		Sign:    sign,
	}

	dp, _ := msgpack.Marshal(d)
	if doEcdsaSign {
		return fmt.Sprintf("SEAL#%s", base64.StdEncoding.EncodeToString(dp))
	}
	return fmt.Sprintf("SEAL~%s", base64.StdEncoding.EncodeToString(dp))
}

func BuildSignature(uin uint64) string {
	decoded, err2 := hex.DecodeString(strings.TrimSpace(SealSignClientPrivateKey))
	if err2 != nil {
		return ""
	}
	if len(decoded) != ed25519.PrivateKeySize {
		return ""
	}
	var msg [16]byte
	binary.BigEndian.PutUint64(msg[0:8], uin)
	binary.BigEndian.PutUint64(msg[8:16], uint64(time.Now().Unix()))

	sig := ed25519.Sign(decoded, msg[:])

	var sigPayload [80]byte
	copy(sigPayload[0:16], msg[:])
	copy(sigPayload[16:80], sig)

	return base64.StdEncoding.EncodeToString(sigPayload[:])
}
