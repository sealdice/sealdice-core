package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// EcdsaSign Ecdsa 签名
func EcdsaSign(data []byte, privateKey string) (string, error) {
	key := ReadPrivateKey[ecdsa.PrivateKey](privateKey)
	hashed := CalculateSHA1(data)
	sign, err := ecdsa.SignASN1(rand.Reader, key, hashed)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}

func EcdsaSignRow(data []byte, privateKey string) ([]byte, error) {
	key := ReadPrivateKey[ecdsa.PrivateKey](privateKey)
	hashed := CalculateSHA1(data)
	sign, err := ecdsa.SignASN1(rand.Reader, key, hashed)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

// EcdsaVerify 验证 Ecdsa 签名
func EcdsaVerify(data []byte, base64Sig, publicKey string) error {
	sign, err := base64.StdEncoding.DecodeString(base64Sig)
	if err != nil {
		return err
	}
	key := ReadPublicKey[ecdsa.PublicKey](publicKey)
	hashed := CalculateSHA1(data)
	if ok := ecdsa.VerifyASN1(key, hashed, sign); ok {
		return nil
	}
	return fmt.Errorf("verify failed")
}

func EcdsaVerifyRow(data []byte, sign []byte, publicKey string) error {
	key := ReadPublicKey[ecdsa.PublicKey](publicKey)
	hashed := CalculateSHA1(data)
	if ok := ecdsa.VerifyASN1(key, hashed, sign); ok {
		return nil
	}
	return fmt.Errorf("verify failed")
}
