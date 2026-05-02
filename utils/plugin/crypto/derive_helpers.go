package sealcrypto

import (
	"errors"
	"fmt"
	"io"

	"github.com/dop251/goja"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

func deriveBitsPBKDF2(rt *goja.Runtime, algObj *goja.Object, baseKey *cryptoKeyHandle, lengthBits int) ([]byte, error) {
	if len(baseKey.SecretKey) == 0 {
		return nil, errors.New("PBKDF2 requires a secret key")
	}
	if algObj == nil {
		return nil, errors.New("PBKDF2 algorithm parameters are required")
	}
	salt, err := requiredBufferProperty(rt, algObj, "salt")
	if err != nil {
		return nil, err
	}
	iterations, err := intProperty(algObj, "iterations")
	if err != nil {
		return nil, err
	}
	if iterations <= 0 {
		return nil, errors.New("iterations must be > 0")
	}
	hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
	if err != nil {
		return nil, err
	}
	hf, err := hashFactory(hashName)
	if err != nil {
		return nil, err
	}
	return pbkdf2.Key(baseKey.SecretKey, salt, iterations, lengthBits/8, hf), nil
}

func deriveBitsHKDF(rt *goja.Runtime, algObj *goja.Object, baseKey *cryptoKeyHandle, lengthBits int) ([]byte, error) {
	if len(baseKey.SecretKey) == 0 {
		return nil, errors.New("HKDF requires a secret key")
	}
	if algObj == nil {
		return nil, errors.New("HKDF algorithm parameters are required")
	}
	hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
	if err != nil {
		return nil, err
	}
	hf, err := hashFactory(hashName)
	if err != nil {
		return nil, err
	}
	salt := []byte(nil)
	if v := algObj.Get("salt"); isValuePresent(v) {
		salt, err = bufferSourceBytes(rt, v, true, false)
		if err != nil {
			return nil, err
		}
	}
	info := []byte(nil)
	if v := algObj.Get("info"); isValuePresent(v) {
		info, err = bufferSourceBytes(rt, v, true, false)
		if err != nil {
			return nil, err
		}
	}
	reader := hkdf.New(hf, baseKey.SecretKey, salt, info)
	out := make([]byte, lengthBits/8)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, err
	}
	return out, nil
}

func deriveBitsECDH(rt *goja.Runtime, algObj *goja.Object, baseKey *cryptoKeyHandle, lengthBits int) ([]byte, error) {
	if algObj == nil {
		return nil, errors.New("ECDH algorithm parameters are required")
	}
	priv := baseKey.ECDHPrivate
	if priv == nil {
		return nil, errors.New("ECDH baseKey must be an ECDH private key")
	}
	pubVal := algObj.Get("public")
	if !isValuePresent(pubVal) {
		return nil, errors.New("algorithm.public is required")
	}
	pubHandle, err := extractCryptoKeyHandle(rt, pubVal)
	if err != nil {
		return nil, err
	}
	pub := pubHandle.ECDHPublic
	if pub == nil && pubHandle.ECDHPrivate != nil {
		pub = &pubHandle.ECDHPrivate.PublicKey
	}
	if pub == nil {
		return nil, errors.New("algorithm.public must be an ECDH key")
	}
	if pub.Curve != priv.Curve {
		return nil, errors.New("ECDH curve mismatch")
	}
	x, _ := pub.ScalarMult(pub.X, pub.Y, priv.D.Bytes())
	if x == nil {
		return nil, errors.New("ECDH derive failed")
	}
	size := (pub.Curve.Params().BitSize + 7) / 8
	shared := leftPad(x.Bytes(), size)
	if lengthBits > len(shared)*8 {
		return nil, errors.New("requested length exceeds shared secret size")
	}
	return shared[:lengthBits/8], nil
}

func deriveBitsX25519(rt *goja.Runtime, algObj *goja.Object, baseKey *cryptoKeyHandle, lengthBits int) ([]byte, error) {
	if algObj == nil {
		return nil, errors.New("X25519 algorithm parameters are required")
	}
	priv := baseKey.X25519Private
	if priv == nil {
		return nil, errors.New("X25519 baseKey must be an X25519 private key")
	}
	pubVal := algObj.Get("public")
	if !isValuePresent(pubVal) {
		return nil, errors.New("algorithm.public is required")
	}
	pubHandle, err := extractCryptoKeyHandle(rt, pubVal)
	if err != nil {
		return nil, err
	}
	pub := pubHandle.X25519Public
	if pub == nil && pubHandle.X25519Private != nil {
		pub = pubHandle.X25519Private.PublicKey()
	}
	if pub == nil {
		return nil, errors.New("algorithm.public must be an X25519 key")
	}
	shared, err := priv.ECDH(pub)
	if err != nil {
		return nil, err
	}
	if lengthBits > len(shared)*8 {
		return nil, errors.New("requested length exceeds shared secret size")
	}
	return shared[:lengthBits/8], nil
}

func deriveKeyLengthBits(rt *goja.Runtime, algorithm string, algObj *goja.Object) (int, error) {
	switch algorithm {
	case "AES-CBC", "AES-GCM", "AES-CTR", "AES-KW":
		return intProperty(algObj, "length")
	case "DES-CBC":
		return 64, nil
	case "3DES-CBC":
		if algObj == nil {
			return 192, nil
		}
		if v := algObj.Get("length"); isValuePresent(v) {
			length := int(v.ToInteger())
			if length != 128 && length != 192 {
				return 0, errors.New("3DES-CBC length must be 128 or 192")
			}
			return length, nil
		}
		return 192, nil
	case "HMAC":
		if algObj == nil {
			return 0, errors.New("HMAC algorithm parameters are required")
		}
		if v := algObj.Get("length"); isValuePresent(v) {
			length := int(v.ToInteger())
			if length <= 0 {
				return 0, errors.New("HMAC length must be > 0")
			}
			return length, nil
		}
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return 0, err
		}
		return digestLengthBytes(hashName) * 8, nil
	default:
		return 0, fmt.Errorf("unsupported derived key algorithm: %s", algorithm)
	}
}
