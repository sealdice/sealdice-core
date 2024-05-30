package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
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
