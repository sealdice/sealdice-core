package sealcrypto

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

func getRandomValues(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	target := call.Argument(0)
	bytesData, err := bufferSourceBytes(rt, target, false, true)
	if err != nil {
		panic(rt.NewTypeError("crypto.getRandomValues: " + err.Error()))
	}
	if len(bytesData) > maxGetRandomValuesBytes {
		panic(rt.NewTypeError("crypto.getRandomValues: byteLength exceeds 65536"))
	}
	if _, err = rand.Read(bytesData); err != nil {
		panic(rt.NewGoError(err))
	}
	return target
}

func subtleDigest(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, _, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	data, err := bufferSourceBytes(rt, call.Argument(1), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	digest, err := digestBytes(algorithm, data)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, rt.NewArrayBuffer(digest))
}

func subtleGenerateKey(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	extractable := call.Argument(1).ToBoolean()
	usages, err := valueToStringSlice(call.Argument(2))
	if err != nil {
		return rejectedPromise(rt, err)
	}

	switch algorithm {
	case "AES-CBC", "AES-GCM", "AES-CTR", "AES-KW":
		length, err := intProperty(algObj, "length")
		if err != nil {
			return rejectedPromise(rt, err)
		}
		if length != 128 && length != 192 && length != 256 {
			return rejectedPromise(rt, errors.New("AES key length must be 128, 192, or 256"))
		}
		key := make([]byte, length/8)
		if _, err := rand.Read(key); err != nil {
			return rejectedPromise(rt, err)
		}
		handle := &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   algorithm,
				"length": length,
			},
			Usages:    usages,
			SecretKey: key,
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "DES-CBC":
		length := 64
		if algObj != nil {
			if v := algObj.Get("length"); isValuePresent(v) {
				length = int(v.ToInteger())
			}
		}
		if length != 64 {
			return rejectedPromise(rt, errors.New("DES-CBC key length must be 64"))
		}
		key := make([]byte, 8)
		if _, err := rand.Read(key); err != nil {
			return rejectedPromise(rt, err)
		}
		handle := &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   "DES-CBC",
				"length": 64,
			},
			Usages:    usages,
			SecretKey: key,
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "3DES-CBC":
		length := 192
		if algObj != nil {
			if v := algObj.Get("length"); isValuePresent(v) {
				length = int(v.ToInteger())
			}
		}
		if length != 128 && length != 192 {
			return rejectedPromise(rt, errors.New("3DES-CBC key length must be 128 or 192"))
		}
		key := make([]byte, length/8)
		if _, err := rand.Read(key); err != nil {
			return rejectedPromise(rt, err)
		}
		handle := &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   "3DES-CBC",
				"length": length,
			},
			Usages:    usages,
			SecretKey: key,
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "HMAC":
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return rejectedPromise(rt, err)
		}
		length := 0
		if algObj != nil {
			if v := algObj.Get("length"); isValuePresent(v) {
				length = int(v.ToInteger())
			}
		}
		if length <= 0 {
			length = digestLengthBytes(hashName) * 8
		}
		key := make([]byte, length/8)
		if _, err := rand.Read(key); err != nil {
			return rejectedPromise(rt, err)
		}
		handle := &cryptoKeyHandle{
			Type:        "secret",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":   "HMAC",
				"hash":   map[string]interface{}{"name": hashName},
				"length": length,
			},
			Usages:    usages,
			SecretKey: key,
			HMACHash:  hashName,
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "ECDSA":
		if algObj == nil {
			return rejectedPromise(rt, errors.New("algorithm.namedCurve is required"))
		}
		curveNameVal := algObj.Get("namedCurve")
		if goja.IsUndefined(curveNameVal) || goja.IsNull(curveNameVal) {
			return rejectedPromise(rt, errors.New("algorithm.namedCurve is required"))
		}
		curveName := strings.TrimSpace(curveNameVal.String())
		curve, normCurveName, err := namedCurveByName(curveName)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		priv, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return rejectedPromise(rt, err)
		}

		commonAlg := map[string]interface{}{
			"name":       "ECDSA",
			"namedCurve": normCurveName,
		}
		pubUsages, priUsages := splitECUsages("ECDSA", usages)
		pubHandle := &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm:   cloneMap(commonAlg),
			Usages:      pubUsages,
			ECDSAPublic: &priv.PublicKey,
		}
		priHandle := &cryptoKeyHandle{
			Type:         "private",
			Extractable:  extractable,
			Algorithm:    cloneMap(commonAlg),
			Usages:       priUsages,
			ECDSAPrivate: priv,
		}

		pair := rt.NewObject()
		_ = pair.Set("publicKey", newCryptoKeyObject(rt, pubHandle))
		_ = pair.Set("privateKey", newCryptoKeyObject(rt, priHandle))
		return resolvedPromise(rt, pair)
	case "ECDH":
		if algObj == nil {
			return rejectedPromise(rt, errors.New("algorithm.namedCurve is required"))
		}
		curveNameVal := algObj.Get("namedCurve")
		if goja.IsUndefined(curveNameVal) || goja.IsNull(curveNameVal) {
			return rejectedPromise(rt, errors.New("algorithm.namedCurve is required"))
		}
		curveName := strings.TrimSpace(curveNameVal.String())
		curve, normCurveName, err := namedCurveByName(curveName)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		priv, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return rejectedPromise(rt, err)
		}

		commonAlg := map[string]interface{}{
			"name":       "ECDH",
			"namedCurve": normCurveName,
		}
		pubUsages, priUsages := splitECUsages("ECDH", usages)
		pubHandle := &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm:   cloneMap(commonAlg),
			Usages:      pubUsages,
			ECDHPublic:  &priv.PublicKey,
		}
		priHandle := &cryptoKeyHandle{
			Type:        "private",
			Extractable: extractable,
			Algorithm:   cloneMap(commonAlg),
			Usages:      priUsages,
			ECDHPrivate: priv,
		}

		pair := rt.NewObject()
		_ = pair.Set("publicKey", newCryptoKeyObject(rt, pubHandle))
		_ = pair.Set("privateKey", newCryptoKeyObject(rt, priHandle))
		return resolvedPromise(rt, pair)
	case "Ed25519":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		commonAlg := map[string]interface{}{"name": "Ed25519"}
		pubUsages, priUsages := splitOKPUsages("Ed25519", usages)
		pubHandle := &cryptoKeyHandle{
			Type:          "public",
			Extractable:   true,
			Algorithm:     cloneMap(commonAlg),
			Usages:        pubUsages,
			Ed25519Public: pub,
		}
		priHandle := &cryptoKeyHandle{
			Type:           "private",
			Extractable:    extractable,
			Algorithm:      cloneMap(commonAlg),
			Usages:         priUsages,
			Ed25519Private: priv,
		}
		pair := rt.NewObject()
		_ = pair.Set("publicKey", newCryptoKeyObject(rt, pubHandle))
		_ = pair.Set("privateKey", newCryptoKeyObject(rt, priHandle))
		return resolvedPromise(rt, pair)
	case "X25519":
		priv, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		commonAlg := map[string]interface{}{"name": "X25519"}
		pubUsages, priUsages := splitOKPUsages("X25519", usages)
		pubHandle := &cryptoKeyHandle{
			Type:         "public",
			Extractable:  true,
			Algorithm:    cloneMap(commonAlg),
			Usages:       pubUsages,
			X25519Public: priv.PublicKey(),
		}
		priHandle := &cryptoKeyHandle{
			Type:          "private",
			Extractable:   extractable,
			Algorithm:     cloneMap(commonAlg),
			Usages:        priUsages,
			X25519Private: priv,
		}
		pair := rt.NewObject()
		_ = pair.Set("publicKey", newCryptoKeyObject(rt, pubHandle))
		_ = pair.Set("privateKey", newCryptoKeyObject(rt, priHandle))
		return resolvedPromise(rt, pair)
	case "RSASSA-PKCS1-V1_5", "RSA-PSS", "RSA-OAEP", "RSAES-PKCS1-V1_5":
		modulusLength, err := intProperty(algObj, "modulusLength")
		if err != nil {
			return rejectedPromise(rt, err)
		}
		if modulusLength < 512 {
			return rejectedPromise(rt, errors.New("modulusLength must be >= 512"))
		}
		exp, err := publicExponentFromAlg(rt, algObj)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return rejectedPromise(rt, err)
		}

		priv, err := generateRSAKeyWithExponent(modulusLength, exp)
		if err != nil {
			return rejectedPromise(rt, err)
		}

		commonAlg := map[string]interface{}{
			"name":          algorithm,
			"modulusLength": modulusLength,
			"publicExponent": []byte{
				byte((exp >> 16) & 0xff),
				byte((exp >> 8) & 0xff),
				byte(exp & 0xff),
			},
			"hash": map[string]interface{}{"name": hashName},
		}

		pubUsages, priUsages := splitRSAUsages(algorithm, usages)
		pubHandle := &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm:   cloneMap(commonAlg),
			Usages:      pubUsages,
			RSAPublic:   &priv.PublicKey,
		}
		priHandle := &cryptoKeyHandle{
			Type:        "private",
			Extractable: extractable,
			Algorithm:   cloneMap(commonAlg),
			Usages:      priUsages,
			RSAPrivate:  priv,
		}

		pair := rt.NewObject()
		_ = pair.Set("publicKey", newCryptoKeyObject(rt, pubHandle))
		_ = pair.Set("privateKey", newCryptoKeyObject(rt, priHandle))
		return resolvedPromise(rt, pair)
	default:
		return rejectedPromise(rt, fmt.Errorf("unsupported generateKey algorithm: %s", algorithm))
	}
}

func subtleImportKey(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	format := strings.ToLower(strings.TrimSpace(call.Argument(0).String()))
	if format == "" {
		return rejectedPromise(rt, errors.New("format is required"))
	}
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(2))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	extractable := call.Argument(3).ToBoolean()
	usages, err := valueToStringSlice(call.Argument(4))
	if err != nil {
		return rejectedPromise(rt, err)
	}

	switch format {
	case "raw":
		raw, err := bufferSourceBytes(rt, call.Argument(1), true, false)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		handle, err := importRawKey(rt, algorithm, algObj, raw, extractable, usages)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "jwk":
		jwk, err := parseJWK(rt, call.Argument(1))
		if err != nil {
			return rejectedPromise(rt, err)
		}
		handle, err := importJWK(rt, jwk, algorithm, algObj, extractable, usages)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "pkcs1":
		raw, err := bufferSourceBytes(rt, call.Argument(1), true, false)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		if priv, parseErr := x509.ParsePKCS1PrivateKey(raw); parseErr == nil {
			handle, handleErr := privateKeyToHandle(rt, priv, algorithm, algObj, extractable, usages)
			if handleErr != nil {
				return rejectedPromise(rt, handleErr)
			}
			return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
		}
		pub, err := x509.ParsePKCS1PublicKey(raw)
		if err != nil {
			return rejectedPromise(rt, errors.New("failed to parse pkcs1 key"))
		}
		handle, err := publicKeyToHandle(rt, pub, algorithm, algObj, usages)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "sec1":
		raw, err := bufferSourceBytes(rt, call.Argument(1), true, false)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		priv, err := x509.ParseECPrivateKey(raw)
		if err != nil {
			return rejectedPromise(rt, errors.New("failed to parse sec1 key"))
		}
		handle, err := privateKeyToHandle(rt, priv, algorithm, algObj, extractable, usages)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "pkcs8":
		raw, err := bufferSourceBytes(rt, call.Argument(1), true, false)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		privAny, err := parsePrivateKey(raw)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		handle, err := privateKeyToHandle(rt, privAny, algorithm, algObj, extractable, usages)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	case "spki":
		raw, err := bufferSourceBytes(rt, call.Argument(1), true, false)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		pubAny, err := x509.ParsePKIXPublicKey(raw)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		handle, err := publicKeyToHandle(rt, pubAny, algorithm, algObj, usages)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
	default:
		return rejectedPromise(rt, fmt.Errorf("unsupported key format: %s", format))
	}
}

func subtleExportKey(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	format := strings.ToLower(strings.TrimSpace(call.Argument(0).String()))
	handle, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	if !handle.Extractable {
		return rejectedPromise(rt, errors.New("key is not extractable"))
	}

	switch format {
	case "raw":
		out, err := exportRawKeyMaterial(handle)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, rt.NewArrayBuffer(out))
	case "jwk":
		jwk, err := exportJWK(handle)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, jwk)
	case "pkcs8":
		var privAny interface{}
		if handle.RSAPrivate != nil {
			privAny = handle.RSAPrivate
		} else if handle.ECDSAPrivate != nil {
			privAny = handle.ECDSAPrivate
		} else if handle.ECDHPrivate != nil {
			privAny = handle.ECDHPrivate
		} else if len(handle.Ed25519Private) != 0 {
			privAny = handle.Ed25519Private
		} else if handle.X25519Private != nil {
			privAny = handle.X25519Private
		} else {
			return rejectedPromise(rt, errors.New("pkcs8 export requires a private key"))
		}
		der, err := x509.MarshalPKCS8PrivateKey(privAny)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, rt.NewArrayBuffer(der))
	case "pkcs1":
		if handle.RSAPrivate != nil {
			return resolvedPromise(rt, rt.NewArrayBuffer(x509.MarshalPKCS1PrivateKey(handle.RSAPrivate)))
		}
		pub := handle.RSAPublic
		if pub == nil && handle.RSAPrivate != nil {
			pub = &handle.RSAPrivate.PublicKey
		}
		if pub == nil {
			return rejectedPromise(rt, errors.New("pkcs1 export requires an RSA key"))
		}
		return resolvedPromise(rt, rt.NewArrayBuffer(x509.MarshalPKCS1PublicKey(pub)))
	case "sec1":
		var priv *ecdsa.PrivateKey
		if handle.ECDSAPrivate != nil {
			priv = handle.ECDSAPrivate
		} else if handle.ECDHPrivate != nil {
			priv = handle.ECDHPrivate
		} else {
			return rejectedPromise(rt, errors.New("sec1 export requires an EC private key"))
		}
		der, err := x509.MarshalECPrivateKey(priv)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, rt.NewArrayBuffer(der))
	case "spki":
		var pubAny interface{}
		if handle.RSAPublic != nil {
			pubAny = handle.RSAPublic
		} else if handle.ECDSAPublic != nil {
			pubAny = handle.ECDSAPublic
		} else if handle.ECDHPublic != nil {
			pubAny = handle.ECDHPublic
		} else if len(handle.Ed25519Public) != 0 {
			pubAny = handle.Ed25519Public
		} else if handle.X25519Public != nil {
			pubAny = handle.X25519Public
		} else if handle.RSAPrivate != nil {
			pubAny = &handle.RSAPrivate.PublicKey
		} else if handle.ECDSAPrivate != nil {
			pubAny = &handle.ECDSAPrivate.PublicKey
		} else if handle.ECDHPrivate != nil {
			pubAny = &handle.ECDHPrivate.PublicKey
		} else if len(handle.Ed25519Private) != 0 {
			pubAny = handle.Ed25519Private.Public().(ed25519.PublicKey)
		} else if handle.X25519Private != nil {
			pubAny = handle.X25519Private.PublicKey()
		} else {
			return rejectedPromise(rt, errors.New("spki export requires a public key"))
		}
		der, err := x509.MarshalPKIXPublicKey(pubAny)
		if err != nil {
			return rejectedPromise(rt, err)
		}
		return resolvedPromise(rt, rt.NewArrayBuffer(der))
	default:
		return rejectedPromise(rt, fmt.Errorf("unsupported export format: %s", format))
	}
}

func subtleSign(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	handle, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	data, err := bufferSourceBytes(rt, call.Argument(2), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	sig, err := signData(rt, algorithm, algObj, handle, data)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, rt.NewArrayBuffer(sig))
}

func subtleVerify(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	handle, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	signature, err := bufferSourceBytes(rt, call.Argument(2), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	data, err := bufferSourceBytes(rt, call.Argument(3), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	ok, err := verifyData(rt, algorithm, algObj, handle, signature, data)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, ok)
}

func subtleEncrypt(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	handle, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	data, err := bufferSourceBytes(rt, call.Argument(2), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	cipherText, err := encryptData(rt, algorithm, algObj, handle, data)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, rt.NewArrayBuffer(cipherText))
}

func subtleDecrypt(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	handle, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	data, err := bufferSourceBytes(rt, call.Argument(2), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	plainText, err := decryptData(rt, algorithm, algObj, handle, data)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, rt.NewArrayBuffer(plainText))
}

func subtleDeriveBits(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	algorithm, algObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	baseKey, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	lengthBits := int(call.Argument(2).ToInteger())
	if lengthBits <= 0 || lengthBits%8 != 0 {
		return rejectedPromise(rt, errors.New("length must be a positive multiple of 8"))
	}

	var bits []byte
	switch algorithm {
	case "PBKDF2":
		bits, err = deriveBitsPBKDF2(rt, algObj, baseKey, lengthBits)
	case "HKDF":
		bits, err = deriveBitsHKDF(rt, algObj, baseKey, lengthBits)
	case "ECDH":
		bits, err = deriveBitsECDH(rt, algObj, baseKey, lengthBits)
	case "X25519":
		bits, err = deriveBitsX25519(rt, algObj, baseKey, lengthBits)
	default:
		return rejectedPromise(rt, fmt.Errorf("unsupported deriveBits algorithm: %s", algorithm))
	}
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, rt.NewArrayBuffer(bits))
}

func subtleDeriveKey(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	baseAlgorithm, baseAlgObj, err := parseAlgorithmIdentifier(rt, call.Argument(0))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	baseKey, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	derivedAlgorithm, derivedAlgObj, err := parseAlgorithmIdentifier(rt, call.Argument(2))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	extractable := call.Argument(3).ToBoolean()
	usages, err := valueToStringSlice(call.Argument(4))
	if err != nil {
		return rejectedPromise(rt, err)
	}

	lengthBits, err := deriveKeyLengthBits(rt, derivedAlgorithm, derivedAlgObj)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	var bits []byte
	switch baseAlgorithm {
	case "PBKDF2":
		bits, err = deriveBitsPBKDF2(rt, baseAlgObj, baseKey, lengthBits)
	case "HKDF":
		bits, err = deriveBitsHKDF(rt, baseAlgObj, baseKey, lengthBits)
	case "ECDH":
		bits, err = deriveBitsECDH(rt, baseAlgObj, baseKey, lengthBits)
	case "X25519":
		bits, err = deriveBitsX25519(rt, baseAlgObj, baseKey, lengthBits)
	default:
		return rejectedPromise(rt, fmt.Errorf("unsupported deriveKey base algorithm: %s", baseAlgorithm))
	}
	if err != nil {
		return rejectedPromise(rt, err)
	}

	handle, err := importRawKey(rt, derivedAlgorithm, derivedAlgObj, bits, extractable, usages)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
}

func subtleWrapKey(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	format := strings.ToLower(strings.TrimSpace(call.Argument(0).String()))
	keyToWrap, err := extractCryptoKeyHandle(rt, call.Argument(1))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	if !keyToWrap.Extractable {
		return rejectedPromise(rt, errors.New("key is not extractable"))
	}
	wrappingKey, err := extractCryptoKeyHandle(rt, call.Argument(2))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	wrapAlgorithm, wrapAlgObj, err := parseAlgorithmIdentifier(rt, call.Argument(3))
	if err != nil {
		return rejectedPromise(rt, err)
	}

	rawKey, err := exportKeyBytesForWrap(rt, format, keyToWrap)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	wrapped, err := encryptData(rt, wrapAlgorithm, wrapAlgObj, wrappingKey, rawKey)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, rt.NewArrayBuffer(wrapped))
}

func subtleUnwrapKey(rt *goja.Runtime, call goja.FunctionCall) goja.Value {
	format := strings.ToLower(strings.TrimSpace(call.Argument(0).String()))
	wrappedData, err := bufferSourceBytes(rt, call.Argument(1), true, false)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	unwrappingKey, err := extractCryptoKeyHandle(rt, call.Argument(2))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	unwrapAlgorithm, unwrapAlgObj, err := parseAlgorithmIdentifier(rt, call.Argument(3))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	unwrappedKeyAlgorithm, unwrappedKeyAlgObj, err := parseAlgorithmIdentifier(rt, call.Argument(4))
	if err != nil {
		return rejectedPromise(rt, err)
	}
	extractable := call.Argument(5).ToBoolean()
	usages, err := valueToStringSlice(call.Argument(6))
	if err != nil {
		return rejectedPromise(rt, err)
	}

	rawKey, err := decryptData(rt, unwrapAlgorithm, unwrapAlgObj, unwrappingKey, wrappedData)
	if err != nil {
		return rejectedPromise(rt, err)
	}

	handle, err := importUnwrappedKey(rt, format, rawKey, unwrappedKeyAlgorithm, unwrappedKeyAlgObj, extractable, usages)
	if err != nil {
		return rejectedPromise(rt, err)
	}
	return resolvedPromise(rt, newCryptoKeyObject(rt, handle))
}
