package dice

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Milly/go-base2048"
	"github.com/vmihailenco/msgpack"

	"sealdice-core/logger"
	"sealdice-core/utils/crypto"
)

var (
	// SealTrustedClientPrivateKey 可信客户端私钥
	SealTrustedClientPrivateKey = ``
)

// recoverKeyRSAPublicKey 用来 dump 上面那个 key 用的 key, 这个是 RSA 8192, 私钥只有我有, 这个功能很快就会移除, 别想着在这上面做文章了, 洗洗睡吧
const recoverKeyRSAPublicKey = `-----BEGIN PUBLIC KEY-----
MIIEIjANBgkqhkiG9w0BAQEFAAOCBA8AMIIECgKCBAEAxdYpWeo5XhmHTrKXcTi3
WxSsfjqEtrNTUtvGO/3UpuJuj1uMy0z9hqCsdilGnl1YMtvNMpS8E8q0rTAdVBpK
kLHdTRmW6KSVZM6SVyes3BCKwSr56gLtDks1lsy+TyTyLrD3UfR0rJGKFOUimoBt
wDG3b+U4ychEdVSwB983isE+WM+i9z5AH9An46Fx4AB92ubLGn6Y4j48vb1kGJUR
MZ0J7gNevdac48M2pTJpoF/HGKBY104wn1qiF2/67nZtZSUFzhhOCIuOfNXOlZNj
pp7rxgLzlDjAMyvdxF4QB8+k8vB75yg7U38Q+b7vH2Zgsw5nT4iGx9dQRqcxAzxm
A87djCNbwHH43ml10lhMUKbnbMpzeCAFcThNTafrmIw+gqsbAe+n4b71xFmeo0LK
ckqQawL03teq8USAiAms9xfjpxWMn8sFOA1jixMjT7yvccBh+4z8MRjXcObS4s3L
rm5Rlb6lWisa83Yzh2zvDTvfzWb5x6iIXD3l3l3pgN/CzlD2gtRikCyA9rHMc40K
x3c5Gg21xMJwfbklyBIM4JCmSys3aQeNpI40qOgZWZ3F36/sgbDfJ2IPledja13S
xhi5+LQUTarXAp8dKYZ7THSAOiWtqT80heNnMiAmTZnmUaqJd6sxS1XFZbeZcLwq
JI5rNWzR2FZXm2l1g7PqT4wcVr4bNpvY9UhB390f7GXZT2zhKyhcHxWA6sQeXdrV
JNIU99OYIDGlcS1WpUqLG9bJRLAINKqkbTFNOOhxvityFJAkpNt00tS/i2aVrvpo
z2kzJEvsU0Sc/JybjU+t6pA8QhiPa159I6XW7fSGoU+jT3xNbge/T7MdO/ETc3/S
3giap80eIZBsE89H5aLz2eliDAOZUl9r6jROfC0CKGvquvPEb8Msiy6vfWq2/M7J
CbmM0qtWGM6YooZPyILp0aNL5kI2Nw4qQBeOG21Ui8JwOgyq5oEn30ZHqRidhOWy
YueJHM43owJv4iMqKk7w6E0CtznAnQm8N5R9znPDegsFH5tUslKtsuFdelVDphK2
6UcNsx+8tKbe0pwOMGmU/sTDr+rjaR3H7P2/tuURV4GzMpPW+M8UnIA7atGJNdUb
lEbu/GOMJkbQaXQTAz1UCfTMykyJndVdJRJ37YOt5gsyS3vNFwSY1j9BGxs5czhE
uAHBHTDyrypHExtuxQNTNbp3aLjvmbSfb/oW7q9UvhcmWuuAG8bStcZcspe9/CtU
h55dQs+xaS0xoEyEmFVu1JNrEld7xInCNxxnzlcu21K6gjP0AmJsoDcwVNPkL3An
/5vbvH92Exq5P1grOQoaOHL10yO460EH23AioFJ3u5tulnDtbw8UfX60zSWAsTyG
0wIDAQAB
-----END PUBLIC KEY-----`

func initVerify() {
	log := logger.M()
	// 优先读取环境变量中的可信客户端私钥
	key := os.Getenv("SEAL_TRUSTED_PRIVATE_KEY")
	if len(key) > 0 {
		SealTrustedClientPrivateKey = key
	} else if len(SealTrustedClientPrivateKey) == 0 {
		log.Warn("SEAL_TRUSTED_PRIVATE_KEY not found, maybe in development mode")
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

// RecoverSealTrustedClientPrivateKey 使用 recoverKeyRSAPublicKey 加密 SealTrustedClientPrivateKey
func RecoverSealTrustedClientPrivateKey() (string, error) {
	if len(SealTrustedClientPrivateKey) == 0 {
		return "", errors.New("SealTrustedClientPrivateKey 为空，无法恢复")
	}
	if len(recoverKeyRSAPublicKey) == 0 {
		return "", errors.New("recoverKeyRSAPublicKey 未设置，无法恢复")
	}
	encrypted, err := crypto.RSAEncryptOAEP([]byte(SealTrustedClientPrivateKey), recoverKeyRSAPublicKey)
	if err != nil {
		return "", fmt.Errorf("加密失败: %w", err)
	}
	return encrypted, nil
}
