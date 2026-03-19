package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// RSASign RSA 签名
func RSASign(data []byte, privateKey string) (string, error) {
	key := ReadPrivateKey[rsa.PrivateKey](privateKey)
	hashed := CalculateSHA512(data)
	sign, err := rsa.SignPSS(rand.Reader, key, crypto.SHA512, hashed, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}

// RSAVerify 验证 RSA 签名
func RSAVerify(data []byte, base64Sig, publicKey string) error {
	sign, err := base64.StdEncoding.DecodeString(base64Sig)
	key := ReadPublicKey[rsa.PublicKey](publicKey)
	if err != nil {
		return err
	}
	hashed := CalculateSHA512(data)
	return rsa.VerifyPSS(key, crypto.SHA512, hashed, sign, nil)
}

// RSASign256 RSA 签名, 使用 SHA256
func RSASign256(data []byte, privateKey string) (string, error) {
	key := ReadPrivateKey[rsa.PrivateKey](privateKey)
	hashed := CalculateSHA256(data)
	sign, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, hashed, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}

// RSAVerify256 验证 RSA 签名, 使用 SHA256
func RSAVerify256(data []byte, base64Sig, publicKey string) error {
	sign, err := base64.StdEncoding.DecodeString(base64Sig)
	key := ReadPublicKey[rsa.PublicKey](publicKey)
	if err != nil {
		return err
	}
	hashed := CalculateSHA256(data)
	return rsa.VerifyPSS(key, crypto.SHA256, hashed, sign, nil)
}

// RSAEncryptOAEP 使用 RSA 公钥加密数据（OAEP + SHA256），返回 base64 编码的密文。
// 对于超出单次加密长度限制的数据会自动分块加密。
func RSAEncryptOAEP(plaintext []byte, publicKey string) (string, error) {
	key := ReadPublicKey[rsa.PublicKey](publicKey)
	if key == nil {
		return "", errors.New("failed to parse RSA public key")
	}
	hash := sha256.New()
	// 单次加密最大长度 = keySize - 2*hashSize - 2
	maxChunkSize := key.Size() - 2*hash.Size() - 2
	var ciphertext []byte
	for len(plaintext) > 0 {
		chunk := plaintext
		if len(chunk) > maxChunkSize {
			chunk = plaintext[:maxChunkSize]
		}
		encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, chunk, nil)
		if err != nil {
			return "", err
		}
		ciphertext = append(ciphertext, encrypted...)
		plaintext = plaintext[len(chunk):]
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
