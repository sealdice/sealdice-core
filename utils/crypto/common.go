package crypto

import (
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// ReadPublicKey 读取公钥
func ReadPublicKey[T any](publicKey string) *T {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil
	}
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil
	}
	key := publicKeyInterface.(*T)
	return key
}

// ReadSshPublicKey 读取 ssh 格式的公钥
func ReadSshPublicKey[T any](publicKey string) *T {
	parsed, _ := ssh.ParsePublicKey([]byte(publicKey))
	parsedCryptoKey := parsed.(ssh.CryptoPublicKey)
	cryptoKey := parsedCryptoKey.CryptoPublicKey()
	key := cryptoKey.(*T)
	return key
}

// ReadPrivateKey 读取私钥
func ReadPrivateKey[T any](privateKey string) *T {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil
	}
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil
	}
	key := privateKeyInterface.(*T)
	return key
}
