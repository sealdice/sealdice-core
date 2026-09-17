//nolint:gosec // WebCrypto legacy compatibility intentionally includes MD5/SHA-1/DES/3DES.
package sealcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/dop251/goja"
)

const (
	maxGetRandomValuesBytes = 65536
	keyHandleSlot           = "__sealcrypto_key_handle__"
)

var integerTypedArrayNames = map[string]struct{}{
	"Int8Array":         {},
	"Uint8Array":        {},
	"Uint8ClampedArray": {},
	"Int16Array":        {},
	"Uint16Array":       {},
	"Int32Array":        {},
	"Uint32Array":       {},
	"BigInt64Array":     {},
	"BigUint64Array":    {},
}

type cryptoKeyHandle struct {
	Type        string
	Extractable bool
	Algorithm   map[string]interface{}
	Usages      []string

	SecretKey []byte
	HMACHash  string

	RSAPublic  *rsa.PublicKey
	RSAPrivate *rsa.PrivateKey

	ECDSAPublic  *ecdsa.PublicKey
	ECDSAPrivate *ecdsa.PrivateKey
	ECDHPublic   *ecdsa.PublicKey
	ECDHPrivate  *ecdsa.PrivateKey

	Ed25519Public  ed25519.PublicKey
	Ed25519Private ed25519.PrivateKey
	X25519Public   *ecdh.PublicKey
	X25519Private  *ecdh.PrivateKey
}

func Enable(rt *goja.Runtime) {
	_ = rt.Set("crypto", ensureCryptoObject(rt))
}

func Require(rt *goja.Runtime, module *goja.Object) {
	_ = module.Set("exports", ensureCryptoObject(rt))
}

func ensureCryptoObject(rt *goja.Runtime) *goja.Object {
	ensureTextEncodingGlobals(rt)

	if current := rt.Get("crypto"); !goja.IsUndefined(current) && !goja.IsNull(current) {
		if obj, ok := current.(*goja.Object); ok {
			return obj
		}
	}

	cryptoObj := rt.NewObject()
	subtleObj := rt.NewObject()

	_ = subtleObj.Set("digest", func(call goja.FunctionCall) goja.Value {
		return subtleDigest(rt, call)
	})
	_ = subtleObj.Set("generateKey", func(call goja.FunctionCall) goja.Value {
		return subtleGenerateKey(rt, call)
	})
	_ = subtleObj.Set("importKey", func(call goja.FunctionCall) goja.Value {
		return subtleImportKey(rt, call)
	})
	_ = subtleObj.Set("exportKey", func(call goja.FunctionCall) goja.Value {
		return subtleExportKey(rt, call)
	})
	_ = subtleObj.Set("sign", func(call goja.FunctionCall) goja.Value {
		return subtleSign(rt, call)
	})
	_ = subtleObj.Set("verify", func(call goja.FunctionCall) goja.Value {
		return subtleVerify(rt, call)
	})
	_ = subtleObj.Set("encrypt", func(call goja.FunctionCall) goja.Value {
		return subtleEncrypt(rt, call)
	})
	_ = subtleObj.Set("decrypt", func(call goja.FunctionCall) goja.Value {
		return subtleDecrypt(rt, call)
	})
	_ = subtleObj.Set("deriveBits", func(call goja.FunctionCall) goja.Value {
		return subtleDeriveBits(rt, call)
	})
	_ = subtleObj.Set("deriveKey", func(call goja.FunctionCall) goja.Value {
		return subtleDeriveKey(rt, call)
	})
	_ = subtleObj.Set("wrapKey", func(call goja.FunctionCall) goja.Value {
		return subtleWrapKey(rt, call)
	})
	_ = subtleObj.Set("unwrapKey", func(call goja.FunctionCall) goja.Value {
		return subtleUnwrapKey(rt, call)
	})

	_ = cryptoObj.Set("subtle", subtleObj)
	_ = cryptoObj.Set("getRandomValues", func(call goja.FunctionCall) goja.Value {
		return getRandomValues(rt, call)
	})
	_ = cryptoObj.Set("randomUUID", randomUUID)

	_ = rt.Set("crypto", cryptoObj)
	return cryptoObj
}

func signData(rt *goja.Runtime, algorithm string, algObj *goja.Object, key *cryptoKeyHandle, data []byte) ([]byte, error) {
	switch algorithm {
	case "HMAC":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("HMAC requires a secret key")
		}
		hashName := key.HMACHash
		if hashName == "" {
			hashName = "SHA-256"
		}
		if algObj != nil {
			v := algObj.Get("hash")
			if isValuePresent(v) {
				var err error
				hashName, err = hashFromAlgorithmValue(rt, v)
				if err != nil {
					return nil, err
				}
			}
		}
		hf, err := hashFactory(hashName)
		if err != nil {
			return nil, err
		}
		mac := hmac.New(hf, key.SecretKey)
		_, _ = mac.Write(data)
		return mac.Sum(nil), nil
	case "RSASSA-PKCS1-V1_5":
		if key.RSAPrivate == nil {
			return nil, errors.New("RSASSA-PKCS1-v1_5 requires a private RSA key")
		}
		hashName, err := hashNameForRSA(rt, algObj, key, "SHA-256")
		if err != nil {
			return nil, err
		}
		hashID, digest, err := digestForCryptoHash(hashName, data)
		if err != nil {
			return nil, err
		}
		return rsa.SignPKCS1v15(rand.Reader, key.RSAPrivate, hashID, digest)
	case "RSA-PSS":
		if key.RSAPrivate == nil {
			return nil, errors.New("RSA-PSS requires a private RSA key")
		}
		hashName, err := hashNameForRSA(rt, algObj, key, "SHA-256")
		if err != nil {
			return nil, err
		}
		hashID, digest, err := digestForCryptoHash(hashName, data)
		if err != nil {
			return nil, err
		}
		saltLen := digestLengthBytes(hashName)
		if algObj != nil {
			if v := algObj.Get("saltLength"); isValuePresent(v) {
				saltLen = int(v.ToInteger())
			}
		}
		return rsa.SignPSS(rand.Reader, key.RSAPrivate, hashID, digest, &rsa.PSSOptions{SaltLength: saltLen, Hash: hashID})
	case "ECDSA":
		if key.ECDSAPrivate == nil {
			return nil, errors.New("ECDSA requires a private EC key")
		}
		hashName := "SHA-256"
		if key.Algorithm != nil {
			if hv, ok := key.Algorithm["hash"].(map[string]interface{}); ok {
				if name, ok := hv["name"].(string); ok && name != "" {
					hashName = name
				}
			}
		}
		if algObj != nil {
			if v := algObj.Get("hash"); isValuePresent(v) {
				var err error
				hashName, err = hashFromAlgorithmValue(rt, v)
				if err != nil {
					return nil, err
				}
			}
		}
		_, digest, err := digestForCryptoHash(hashName, data)
		if err != nil {
			return nil, err
		}
		r, s, err := ecdsa.Sign(rand.Reader, key.ECDSAPrivate, digest)
		if err != nil {
			return nil, err
		}
		size := (key.ECDSAPrivate.Curve.Params().BitSize + 7) / 8
		sig := make([]byte, size*2)
		copy(sig[:size], leftPad(r.Bytes(), size))
		copy(sig[size:], leftPad(s.Bytes(), size))
		return sig, nil
	case "Ed25519":
		if len(key.Ed25519Private) == 0 {
			return nil, errors.New("Ed25519 requires a private key")
		}
		return ed25519.Sign(key.Ed25519Private, data), nil
	default:
		return nil, fmt.Errorf("unsupported sign algorithm: %s", algorithm)
	}
}

func verifyData(rt *goja.Runtime, algorithm string, algObj *goja.Object, key *cryptoKeyHandle, signature []byte, data []byte) (bool, error) {
	switch algorithm {
	case "HMAC":
		if len(key.SecretKey) == 0 {
			return false, errors.New("HMAC requires a secret key")
		}
		hashName := key.HMACHash
		if hashName == "" {
			hashName = "SHA-256"
		}
		if algObj != nil {
			v := algObj.Get("hash")
			if isValuePresent(v) {
				var err error
				hashName, err = hashFromAlgorithmValue(rt, v)
				if err != nil {
					return false, err
				}
			}
		}
		hf, err := hashFactory(hashName)
		if err != nil {
			return false, err
		}
		mac := hmac.New(hf, key.SecretKey)
		_, _ = mac.Write(data)
		return hmac.Equal(signature, mac.Sum(nil)), nil
	case "RSASSA-PKCS1-V1_5":
		pub := key.RSAPublic
		if pub == nil && key.RSAPrivate != nil {
			pub = &key.RSAPrivate.PublicKey
		}
		if pub == nil {
			return false, errors.New("RSASSA-PKCS1-v1_5 requires an RSA key")
		}
		hashName, err := hashNameForRSA(rt, algObj, key, "SHA-256")
		if err != nil {
			return false, err
		}
		hashID, digest, err := digestForCryptoHash(hashName, data)
		if err != nil {
			return false, err
		}
		err = rsa.VerifyPKCS1v15(pub, hashID, digest, signature)
		return err == nil, nil
	case "RSA-PSS":
		pub := key.RSAPublic
		if pub == nil && key.RSAPrivate != nil {
			pub = &key.RSAPrivate.PublicKey
		}
		if pub == nil {
			return false, errors.New("RSA-PSS requires an RSA key")
		}
		hashName, err := hashNameForRSA(rt, algObj, key, "SHA-256")
		if err != nil {
			return false, err
		}
		hashID, digest, err := digestForCryptoHash(hashName, data)
		if err != nil {
			return false, err
		}
		saltLen := digestLengthBytes(hashName)
		if algObj != nil {
			if v := algObj.Get("saltLength"); isValuePresent(v) {
				saltLen = int(v.ToInteger())
			}
		}
		err = rsa.VerifyPSS(pub, hashID, digest, signature, &rsa.PSSOptions{SaltLength: saltLen, Hash: hashID})
		return err == nil, nil
	case "ECDSA":
		pub := key.ECDSAPublic
		if pub == nil && key.ECDSAPrivate != nil {
			pub = &key.ECDSAPrivate.PublicKey
		}
		if pub == nil {
			return false, errors.New("ECDSA requires an EC key")
		}
		hashName := "SHA-256"
		if key.Algorithm != nil {
			if hv, ok := key.Algorithm["hash"].(map[string]interface{}); ok {
				if name, ok := hv["name"].(string); ok && name != "" {
					hashName = name
				}
			}
		}
		if algObj != nil {
			if v := algObj.Get("hash"); isValuePresent(v) {
				var err error
				hashName, err = hashFromAlgorithmValue(rt, v)
				if err != nil {
					return false, err
				}
			}
		}
		_, digest, err := digestForCryptoHash(hashName, data)
		if err != nil {
			return false, err
		}
		size := (pub.Curve.Params().BitSize + 7) / 8
		if len(signature) == size*2 {
			r := new(big.Int).SetBytes(signature[:size])
			s := new(big.Int).SetBytes(signature[size:])
			return ecdsa.Verify(pub, digest, r, s), nil
		}
		// 兼容 DER ASN.1 格式
		return ecdsa.VerifyASN1(pub, digest, signature), nil
	case "Ed25519":
		pub := key.Ed25519Public
		if len(pub) == 0 && len(key.Ed25519Private) != 0 {
			pub = key.Ed25519Private.Public().(ed25519.PublicKey)
		}
		if len(pub) == 0 {
			return false, errors.New("Ed25519 requires a key")
		}
		return ed25519.Verify(pub, data, signature), nil
	default:
		return false, fmt.Errorf("unsupported verify algorithm: %s", algorithm)
	}
}

func encryptData(rt *goja.Runtime, algorithm string, algObj *goja.Object, key *cryptoKeyHandle, data []byte) ([]byte, error) {
	switch algorithm {
	case "AES-CBC":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-CBC requires a secret key")
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		if len(iv) != aes.BlockSize {
			return nil, errors.New("AES-CBC iv length must be 16")
		}
		block, err := aes.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		padded := pkcs7Pad(data, aes.BlockSize)
		out := make([]byte, len(padded))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
		return out, nil
	case "DES-CBC":
		if len(key.SecretKey) != 8 {
			return nil, errors.New("DES-CBC requires an 8-byte secret key")
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		if len(iv) != des.BlockSize {
			return nil, errors.New("DES-CBC iv length must be 8")
		}
		block, err := des.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		padded := pkcs7Pad(data, des.BlockSize)
		out := make([]byte, len(padded))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
		return out, nil
	case "3DES-CBC":
		desKey, err := tripleDESKey(key.SecretKey)
		if err != nil {
			return nil, err
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		if len(iv) != des.BlockSize {
			return nil, errors.New("3DES-CBC iv length must be 8")
		}
		block, err := des.NewTripleDESCipher(desKey)
		if err != nil {
			return nil, err
		}
		padded := pkcs7Pad(data, des.BlockSize)
		out := make([]byte, len(padded))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
		return out, nil
	case "AES-GCM":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-GCM requires a secret key")
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		tagLength := 128
		if algObj != nil {
			if v := algObj.Get("tagLength"); isValuePresent(v) {
				tagLength = int(v.ToInteger())
			}
		}
		if valErr := validateAESGCMTagLength(tagLength); valErr != nil {
			return nil, valErr
		}
		block, err := aes.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		gcm, err := newAESGCMForParams(block, len(iv), tagLength)
		if err != nil {
			return nil, err
		}
		aad := []byte(nil)
		if algObj != nil {
			if v := algObj.Get("additionalData"); isValuePresent(v) {
				aad, err = bufferSourceBytes(rt, v, true, false)
				if err != nil {
					return nil, err
				}
			}
		}
		return gcm.Seal(nil, iv, data, aad), nil
	case "AES-CTR":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-CTR requires a secret key")
		}
		counter, err := requiredBufferProperty(rt, algObj, "counter")
		if err != nil {
			return nil, err
		}
		if len(counter) != aes.BlockSize {
			return nil, errors.New("AES-CTR counter length must be 16")
		}
		length, err := intProperty(algObj, "length")
		if err != nil {
			return nil, err
		}
		if length < 1 || length > 128 {
			return nil, errors.New("AES-CTR length must be between 1 and 128")
		}
		block, err := aes.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		iv := make([]byte, len(counter))
		copy(iv, counter)
		out := make([]byte, len(data))
		cipher.NewCTR(block, iv).XORKeyStream(out, data)
		return out, nil
	case "AES-KW":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-KW requires a secret key")
		}
		return aesKeyWrap(key.SecretKey, data)
	case "RSA-OAEP":
		pub := key.RSAPublic
		if pub == nil && key.RSAPrivate != nil {
			pub = &key.RSAPrivate.PublicKey
		}
		if pub == nil {
			return nil, errors.New("RSA-OAEP requires an RSA key")
		}
		hashName, err := hashNameForRSA(rt, algObj, key, "SHA-256")
		if err != nil {
			return nil, err
		}
		hf, err := hashFactory(hashName)
		if err != nil {
			return nil, err
		}
		label := []byte(nil)
		if algObj != nil {
			if v := algObj.Get("label"); isValuePresent(v) {
				label, err = bufferSourceBytes(rt, v, true, false)
				if err != nil {
					return nil, err
				}
			}
		}
		return rsa.EncryptOAEP(hf(), rand.Reader, pub, data, label)
	case "RSAES-PKCS1-V1_5":
		pub := key.RSAPublic
		if pub == nil && key.RSAPrivate != nil {
			pub = &key.RSAPrivate.PublicKey
		}
		if pub == nil {
			return nil, errors.New("RSAES-PKCS1-v1_5 requires an RSA key")
		}
		return rsa.EncryptPKCS1v15(rand.Reader, pub, data)
	default:
		return nil, fmt.Errorf("unsupported encrypt algorithm: %s", algorithm)
	}
}

func decryptData(rt *goja.Runtime, algorithm string, algObj *goja.Object, key *cryptoKeyHandle, data []byte) ([]byte, error) {
	switch algorithm {
	case "AES-CBC":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-CBC requires a secret key")
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		if len(iv) != aes.BlockSize {
			return nil, errors.New("AES-CBC iv length must be 16")
		}
		if len(data)%aes.BlockSize != 0 {
			return nil, errors.New("AES-CBC ciphertext length must be multiple of block size")
		}
		block, err := aes.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(data))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
		return pkcs7Unpad(out, aes.BlockSize)
	case "DES-CBC":
		if len(key.SecretKey) != 8 {
			return nil, errors.New("DES-CBC requires an 8-byte secret key")
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		if len(iv) != des.BlockSize {
			return nil, errors.New("DES-CBC iv length must be 8")
		}
		if len(data)%des.BlockSize != 0 {
			return nil, errors.New("DES-CBC ciphertext length must be multiple of block size")
		}
		block, err := des.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(data))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
		return pkcs7Unpad(out, des.BlockSize)
	case "3DES-CBC":
		desKey, err := tripleDESKey(key.SecretKey)
		if err != nil {
			return nil, err
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		if len(iv) != des.BlockSize {
			return nil, errors.New("3DES-CBC iv length must be 8")
		}
		if len(data)%des.BlockSize != 0 {
			return nil, errors.New("3DES-CBC ciphertext length must be multiple of block size")
		}
		block, err := des.NewTripleDESCipher(desKey)
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(data))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
		return pkcs7Unpad(out, des.BlockSize)
	case "AES-GCM":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-GCM requires a secret key")
		}
		iv, err := requiredBufferProperty(rt, algObj, "iv")
		if err != nil {
			return nil, err
		}
		tagLength := 128
		if algObj != nil {
			if v := algObj.Get("tagLength"); isValuePresent(v) {
				tagLength = int(v.ToInteger())
			}
		}
		if valErr := validateAESGCMTagLength(tagLength); valErr != nil {
			return nil, valErr
		}
		block, err := aes.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		gcm, err := newAESGCMForParams(block, len(iv), tagLength)
		if err != nil {
			return nil, err
		}
		aad := []byte(nil)
		if algObj != nil {
			if v := algObj.Get("additionalData"); isValuePresent(v) {
				aad, err = bufferSourceBytes(rt, v, true, false)
				if err != nil {
					return nil, err
				}
			}
		}
		return gcm.Open(nil, iv, data, aad)
	case "AES-CTR":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-CTR requires a secret key")
		}
		counter, err := requiredBufferProperty(rt, algObj, "counter")
		if err != nil {
			return nil, err
		}
		if len(counter) != aes.BlockSize {
			return nil, errors.New("AES-CTR counter length must be 16")
		}
		length, err := intProperty(algObj, "length")
		if err != nil {
			return nil, err
		}
		if length < 1 || length > 128 {
			return nil, errors.New("AES-CTR length must be between 1 and 128")
		}
		block, err := aes.NewCipher(key.SecretKey)
		if err != nil {
			return nil, err
		}
		iv := make([]byte, len(counter))
		copy(iv, counter)
		out := make([]byte, len(data))
		cipher.NewCTR(block, iv).XORKeyStream(out, data)
		return out, nil
	case "AES-KW":
		if len(key.SecretKey) == 0 {
			return nil, errors.New("AES-KW requires a secret key")
		}
		return aesKeyUnwrap(key.SecretKey, data)
	case "RSA-OAEP":
		if key.RSAPrivate == nil {
			return nil, errors.New("RSA-OAEP requires a private RSA key")
		}
		hashName, err := hashNameForRSA(rt, algObj, key, "SHA-256")
		if err != nil {
			return nil, err
		}
		hf, err := hashFactory(hashName)
		if err != nil {
			return nil, err
		}
		label := []byte(nil)
		if algObj != nil {
			if v := algObj.Get("label"); isValuePresent(v) {
				label, err = bufferSourceBytes(rt, v, true, false)
				if err != nil {
					return nil, err
				}
			}
		}
		return rsa.DecryptOAEP(hf(), rand.Reader, key.RSAPrivate, data, label)
	case "RSAES-PKCS1-V1_5":
		if key.RSAPrivate == nil {
			return nil, errors.New("RSAES-PKCS1-v1_5 requires a private RSA key")
		}
		return rsa.DecryptPKCS1v15(rand.Reader, key.RSAPrivate, data)
	default:
		return nil, fmt.Errorf("unsupported decrypt algorithm: %s", algorithm)
	}
}

func importRawKey(rt *goja.Runtime, algorithm string, algObj *goja.Object, raw []byte, extractable bool, usages []string) (*cryptoKeyHandle, error) {
	cpy := make([]byte, len(raw))
	copy(cpy, raw)

	switch algorithm {
	case "AES-CBC", "AES-GCM", "AES-CTR", "AES-KW":
		length := len(cpy) * 8
		if length != 128 && length != 192 && length != 256 {
			return nil, errors.New("AES raw key length must be 16, 24, or 32 bytes")
		}
		return &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   algorithm,
				"length": length,
			},
			Usages:    usages,
			SecretKey: cpy,
		}, nil
	case "DES-CBC":
		if len(cpy) != 8 {
			return nil, errors.New("DES-CBC raw key length must be 8 bytes")
		}
		return &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   "DES-CBC",
				"length": 64,
			},
			Usages:    usages,
			SecretKey: cpy,
		}, nil
	case "3DES-CBC":
		if len(cpy) != 16 && len(cpy) != 24 {
			return nil, errors.New("3DES-CBC raw key length must be 16 or 24 bytes")
		}
		return &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   "3DES-CBC",
				"length": len(cpy) * 8,
			},
			Usages:    usages,
			SecretKey: cpy,
		}, nil
	case "HMAC":
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return nil, err
		}
		length := len(cpy) * 8
		if algObj != nil {
			if v := algObj.Get("length"); isValuePresent(v) {
				length = int(v.ToInteger())
			}
		}
		return &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   "HMAC",
				"hash":   map[string]interface{}{"name": hashName},
				"length": length,
			},
			Usages:    usages,
			SecretKey: cpy,
			HMACHash:  hashName,
		}, nil
	case "PBKDF2":
		return &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name": "PBKDF2",
			},
			Usages:    usages,
			SecretKey: cpy,
		}, nil
	case "HKDF":
		return &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name": "HKDF",
			},
			Usages:    usages,
			SecretKey: cpy,
		}, nil
	case "ECDSA", "ECDH":
		if algObj == nil {
			return nil, errors.New("algorithm.namedCurve is required")
		}
		curveNameVal := algObj.Get("namedCurve")
		if !isValuePresent(curveNameVal) {
			return nil, errors.New("algorithm.namedCurve is required")
		}
		curve, normCurveName, err := namedCurveByName(curveNameVal.String())
		if err != nil {
			return nil, err
		}
		size := (curve.Params().BitSize + 7) / 8
		if len(cpy) != 1+2*size || cpy[0] != 0x04 {
			return nil, errors.New("EC raw key must be an uncompressed point")
		}
		ecdhCurve, err := ecdhCurveByElliptic(curve)
		if err != nil {
			return nil, err
		}
		if _, err = ecdhCurve.NewPublicKey(cpy); err != nil {
			return nil, errors.New("invalid EC raw public key")
		}
		x := new(big.Int).SetBytes(cpy[1 : 1+size])
		y := new(big.Int).SetBytes(cpy[1+size:])
		pub := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		handle := &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm: map[string]interface{}{
				"name":       algorithm,
				"namedCurve": normCurveName,
			},
			Usages: usages,
		}
		if algorithm == "ECDSA" {
			handle.ECDSAPublic = pub
		} else {
			handle.ECDHPublic = pub
		}
		return handle, nil
	case "Ed25519":
		keyType, keyTypeErr := rawKeyTypeHint(algObj)
		if keyTypeErr != nil {
			return nil, keyTypeErr
		}
		isPrivate := keyType == "private"
		if keyType == "" && len(cpy) == ed25519.PrivateKeySize {
			isPrivate = true
		}
		if keyType == "" && len(cpy) == ed25519.SeedSize && usageContains(usages, "sign") && !usageContains(usages, "verify") {
			isPrivate = true
		}
		if isPrivate {
			switch len(cpy) {
			case ed25519.SeedSize:
				priv := ed25519.NewKeyFromSeed(cpy)
				return &cryptoKeyHandle{
					Type:           "private",
					Extractable:    extractable,
					Algorithm:      map[string]interface{}{"name": "Ed25519"},
					Usages:         usages,
					Ed25519Private: priv,
				}, nil
			case ed25519.PrivateKeySize:
				priv := make([]byte, ed25519.PrivateKeySize)
				copy(priv, cpy)
				return &cryptoKeyHandle{
					Type:           "private",
					Extractable:    extractable,
					Algorithm:      map[string]interface{}{"name": "Ed25519"},
					Usages:         usages,
					Ed25519Private: ed25519.PrivateKey(priv),
				}, nil
			default:
				return nil, errors.New("Ed25519 raw private key must be 32-byte seed or 64-byte private key")
			}
		}
		if len(cpy) != ed25519.PublicKeySize {
			return nil, errors.New("Ed25519 raw public key length must be 32 bytes")
		}
		pub := make([]byte, ed25519.PublicKeySize)
		copy(pub, cpy)
		return &cryptoKeyHandle{
			Type:          "public",
			Extractable:   true,
			Algorithm:     map[string]interface{}{"name": "Ed25519"},
			Usages:        usages,
			Ed25519Public: ed25519.PublicKey(pub),
		}, nil
	case "X25519":
		keyType, keyTypeErr := rawKeyTypeHint(algObj)
		if keyTypeErr != nil {
			return nil, keyTypeErr
		}
		if len(cpy) != 32 {
			if keyType == "private" {
				return nil, errors.New("X25519 raw private key length must be 32 bytes")
			}
			return nil, errors.New("X25519 raw public key length must be 32 bytes")
		}
		if keyType == "private" {
			priv, privErr := ecdh.X25519().NewPrivateKey(cpy)
			if privErr != nil {
				return nil, privErr
			}
			return &cryptoKeyHandle{
				Type:          "private",
				Extractable:   extractable,
				Algorithm:     map[string]interface{}{"name": "X25519"},
				Usages:        usages,
				X25519Private: priv,
			}, nil
		}
		pub, err := ecdh.X25519().NewPublicKey(cpy)
		if err != nil {
			return nil, err
		}
		return &cryptoKeyHandle{
			Type:         "public",
			Extractable:  true,
			Algorithm:    map[string]interface{}{"name": "X25519"},
			Usages:       usages,
			X25519Public: pub,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported raw key algorithm: %s", algorithm)
	}
}
