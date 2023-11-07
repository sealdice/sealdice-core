package dice

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// RSASign 签名
func RSASign(data []byte, privateKey string) (string, error) {
	key := ReadPrivateKey(privateKey)
	hashed := CalculateSHA512(data)
	sign, err := rsa.SignPSS(rand.Reader, key, crypto.SHA512, hashed, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}

// RSAVerify 验证签名
func RSAVerify(data []byte, base64Sig, publicKey string) error {
	sign, err := base64.StdEncoding.DecodeString(base64Sig)
	key := ReadPublicKey(publicKey)
	if err != nil {
		return err
	}
	hashed := CalculateSHA512(data)
	return rsa.VerifyPSS(key, crypto.SHA512, hashed, sign, nil)
}

// ReadPublicKey 读取公钥
func ReadPublicKey(publicKey string) *rsa.PublicKey {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil
	}
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil
	}
	key := publicKeyInterface.(*rsa.PublicKey)
	return key
}

// ReadSshPublicKey 读取 ssh 格式的公钥
func ReadSshPublicKey(publicKey string) *rsa.PublicKey {
	parsed, _ := ssh.ParsePublicKey([]byte(publicKey))
	parsedCryptoKey := parsed.(ssh.CryptoPublicKey)
	cryptoKey := parsedCryptoKey.CryptoPublicKey()
	key := cryptoKey.(*rsa.PublicKey)
	return key
}

// ReadPrivateKey 读取私钥
func ReadPrivateKey(privateKey string) *rsa.PrivateKey {
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

func CalculateSHA512Str(data []byte) string {
	return hex.EncodeToString(CalculateSHA512(data))
}

func CalculateSHA512(data []byte) []byte {
	hashInstance := crypto.SHA512.New()
	hashInstance.Write(data)
	hashed := hashInstance.Sum(nil)
	return hashed
}
