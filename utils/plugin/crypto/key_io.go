package sealcrypto

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/dop251/goja"
)

func importJWK(rt *goja.Runtime, jwk map[string]interface{}, algorithm string, algObj *goja.Object, extractable bool, usages []string) (*cryptoKeyHandle, error) {
	kty := strings.ToUpper(stringFromMap(jwk, "kty"))
	switch kty {
	case "OCT":
		k := stringFromMap(jwk, "k")
		if k == "" {
			return nil, errors.New("invalid oct JWK: missing k")
		}
		keyBytes, err := base64.RawURLEncoding.DecodeString(k)
		if err != nil {
			return nil, errors.New("invalid oct JWK key material")
		}
		return importRawKey(rt, algorithm, algObj, keyBytes, extractable, usages)
	case "RSA":
		n, err := parseJWKBigInt(jwk, "n")
		if err != nil {
			return nil, err
		}
		eInt, err := parseJWKBigInt(jwk, "e")
		if err != nil {
			return nil, err
		}
		e := int(eInt.Int64())
		if e <= 0 {
			return nil, errors.New("invalid RSA JWK exponent")
		}
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return nil, err
		}

		if hasMapKey(jwk, "d") {
			d, err := parseJWKBigInt(jwk, "d")
			if err != nil {
				return nil, err
			}
			if hasMapKey(jwk, "oth") {
				return nil, errors.New("RSA JWK multi-prime oth is unsupported")
			}
			hasP := hasMapKey(jwk, "p")
			hasQ := hasMapKey(jwk, "q")
			if hasP != hasQ {
				return nil, errors.New("RSA JWK p/q must both be present")
			}
			var p, q *big.Int
			if hasP {
				p, err = parseJWKBigInt(jwk, "p")
				if err != nil {
					return nil, err
				}
				q, err = parseJWKBigInt(jwk, "q")
				if err != nil {
					return nil, err
				}
			} else {
				p, q, err = recoverRSAFactorsFromNED(n, e, d)
				if err != nil {
					return nil, err
				}
			}
			priv := &rsa.PrivateKey{
				PublicKey: rsa.PublicKey{N: n, E: e},
				D:         d,
				Primes:    []*big.Int{p, q},
			}
			if err := priv.Validate(); err != nil {
				return nil, err
			}
			priv.Precompute()
			return &cryptoKeyHandle{
				Type:        "private",
				Extractable: extractable,
				Algorithm: map[string]interface{}{
					"name": algorithm,
					"hash": map[string]interface{}{"name": hashName},
				},
				Usages:     usages,
				RSAPrivate: priv,
			}, nil
		}

		pub := &rsa.PublicKey{N: n, E: e}
		return &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm: map[string]interface{}{
				"name": algorithm,
				"hash": map[string]interface{}{"name": hashName},
			},
			Usages:    usages,
			RSAPublic: pub,
		}, nil
	case "EC":
		if algorithm != "ECDSA" && algorithm != "ECDH" {
			return nil, errors.New("EC JWK currently supports only ECDSA/ECDH algorithm")
		}
		crv := strings.TrimSpace(stringFromMap(jwk, "crv"))
		curve, normCurveName, err := namedCurveByName(crv)
		if err != nil {
			return nil, err
		}
		x, err := parseJWKBigInt(jwk, "x")
		if err != nil {
			return nil, err
		}
		y, err := parseJWKBigInt(jwk, "y")
		if err != nil {
			return nil, err
		}
		size := (curve.Params().BitSize + 7) / 8
		rawPub := make([]byte, 1+2*size)
		rawPub[0] = 0x04
		copy(rawPub[1:1+size], leftPad(x.Bytes(), size))
		copy(rawPub[1+size:], leftPad(y.Bytes(), size))
		ecdhCurve, err := ecdhCurveByElliptic(curve)
		if err != nil {
			return nil, err
		}
		if _, err = ecdhCurve.NewPublicKey(rawPub); err != nil {
			return nil, errors.New("invalid EC JWK point")
		}
		pub := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		if hasMapKey(jwk, "d") {
			d, err := parseJWKBigInt(jwk, "d")
			if err != nil {
				return nil, err
			}
			priv := &ecdsa.PrivateKey{PublicKey: *pub, D: d}
			handle := &cryptoKeyHandle{
				Type:        "private",
				Extractable: extractable,
				Algorithm: map[string]interface{}{
					"name":       algorithm,
					"namedCurve": normCurveName,
				},
				Usages: usages,
			}
			if algorithm == "ECDSA" {
				handle.ECDSAPrivate = priv
			} else {
				handle.ECDHPrivate = priv
			}
			return handle, nil
		}
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
	case "OKP":
		crv := strings.TrimSpace(stringFromMap(jwk, "crv"))
		xEnc := stringFromMap(jwk, "x")
		if xEnc == "" {
			return nil, errors.New("invalid OKP JWK: missing x")
		}
		x, err := base64.RawURLEncoding.DecodeString(xEnc)
		if err != nil {
			return nil, errors.New("invalid OKP JWK x")
		}
		dEnc := stringFromMap(jwk, "d")
		switch strings.ToUpper(crv) {
		case "ED25519":
			if algorithm != "Ed25519" {
				return nil, errors.New("OKP Ed25519 key requires Ed25519 algorithm")
			}
			if len(x) != ed25519.PublicKeySize {
				return nil, errors.New("invalid Ed25519 public key length")
			}
			if dEnc != "" {
				d, err := base64.RawURLEncoding.DecodeString(dEnc)
				if err != nil {
					return nil, errors.New("invalid OKP JWK d")
				}
				if len(d) != ed25519.SeedSize {
					return nil, errors.New("invalid Ed25519 private seed length")
				}
				priv := ed25519.NewKeyFromSeed(d)
				return &cryptoKeyHandle{
					Type:           "private",
					Extractable:    extractable,
					Algorithm:      map[string]interface{}{"name": "Ed25519"},
					Usages:         usages,
					Ed25519Private: priv,
				}, nil
			}
			return &cryptoKeyHandle{
				Type:          "public",
				Extractable:   true,
				Algorithm:     map[string]interface{}{"name": "Ed25519"},
				Usages:        usages,
				Ed25519Public: ed25519.PublicKey(x),
			}, nil
		case "X25519":
			if algorithm != "X25519" {
				return nil, errors.New("OKP X25519 key requires X25519 algorithm")
			}
			if len(x) != 32 {
				return nil, errors.New("invalid X25519 public key length")
			}
			if dEnc != "" {
				d, err := base64.RawURLEncoding.DecodeString(dEnc)
				if err != nil {
					return nil, errors.New("invalid OKP JWK d")
				}
				priv, err := ecdh.X25519().NewPrivateKey(d)
				if err != nil {
					return nil, err
				}
				return &cryptoKeyHandle{
					Type:          "private",
					Extractable:   extractable,
					Algorithm:     map[string]interface{}{"name": "X25519"},
					Usages:        usages,
					X25519Private: priv,
				}, nil
			}
			pub, err := ecdh.X25519().NewPublicKey(x)
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
			return nil, errors.New("unsupported OKP crv")
		}
	default:
		return nil, errors.New("unsupported JWK kty")
	}
}

func exportJWK(handle *cryptoKeyHandle) (map[string]interface{}, error) {
	jwk := map[string]interface{}{
		"key_ops": handle.Usages,
		"ext":     handle.Extractable,
	}
	if alg := jwkAlgForHandle(handle); alg != "" {
		jwk["alg"] = alg
	}

	if len(handle.SecretKey) > 0 {
		jwk["kty"] = "oct"
		jwk["k"] = base64.RawURLEncoding.EncodeToString(handle.SecretKey)
		return jwk, nil
	}

	if handle.RSAPublic != nil || handle.RSAPrivate != nil {
		jwk["kty"] = "RSA"
		pub := handle.RSAPublic
		if pub == nil {
			pub = &handle.RSAPrivate.PublicKey
		}
		jwk["n"] = base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
		jwk["e"] = base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
		if handle.RSAPrivate != nil {
			priv := handle.RSAPrivate
			jwk["d"] = base64.RawURLEncoding.EncodeToString(priv.D.Bytes())
			if len(priv.Primes) >= 2 {
				jwk["p"] = base64.RawURLEncoding.EncodeToString(priv.Primes[0].Bytes())
				jwk["q"] = base64.RawURLEncoding.EncodeToString(priv.Primes[1].Bytes())
				dp := new(big.Int).Mod(priv.D, new(big.Int).Sub(priv.Primes[0], big.NewInt(1)))
				dq := new(big.Int).Mod(priv.D, new(big.Int).Sub(priv.Primes[1], big.NewInt(1)))
				qi := new(big.Int).ModInverse(priv.Primes[1], priv.Primes[0])
				if qi != nil {
					jwk["dp"] = base64.RawURLEncoding.EncodeToString(dp.Bytes())
					jwk["dq"] = base64.RawURLEncoding.EncodeToString(dq.Bytes())
					jwk["qi"] = base64.RawURLEncoding.EncodeToString(qi.Bytes())
				}
			}
		}
		return jwk, nil
	}

	if handle.ECDSAPublic != nil || handle.ECDSAPrivate != nil || handle.ECDHPublic != nil || handle.ECDHPrivate != nil {
		jwk["kty"] = "EC"
		pub := handle.ECDSAPublic
		if pub == nil && handle.ECDSAPrivate != nil {
			pub = &handle.ECDSAPrivate.PublicKey
		}
		if pub == nil {
			pub = handle.ECDHPublic
		}
		if pub == nil && handle.ECDHPrivate != nil {
			pub = &handle.ECDHPrivate.PublicKey
		}
		size := (pub.Curve.Params().BitSize + 7) / 8
		jwk["crv"] = namedCurveFromElliptic(pub.Curve)
		jwk["x"] = base64.RawURLEncoding.EncodeToString(leftPad(pub.X.Bytes(), size))
		jwk["y"] = base64.RawURLEncoding.EncodeToString(leftPad(pub.Y.Bytes(), size))
		if handle.ECDSAPrivate != nil {
			jwk["d"] = base64.RawURLEncoding.EncodeToString(leftPad(handle.ECDSAPrivate.D.Bytes(), size))
		} else if handle.ECDHPrivate != nil {
			jwk["d"] = base64.RawURLEncoding.EncodeToString(leftPad(handle.ECDHPrivate.D.Bytes(), size))
		}
		return jwk, nil
	}

	if len(handle.Ed25519Public) != 0 || len(handle.Ed25519Private) != 0 {
		jwk["kty"] = "OKP"
		jwk["crv"] = "Ed25519"
		pub := handle.Ed25519Public
		if len(pub) == 0 {
			pub = handle.Ed25519Private.Public().(ed25519.PublicKey)
		}
		jwk["x"] = base64.RawURLEncoding.EncodeToString(pub)
		if len(handle.Ed25519Private) != 0 {
			jwk["d"] = base64.RawURLEncoding.EncodeToString(handle.Ed25519Private.Seed())
		}
		return jwk, nil
	}

	if handle.X25519Public != nil || handle.X25519Private != nil {
		jwk["kty"] = "OKP"
		jwk["crv"] = "X25519"
		pub := handle.X25519Public
		if pub == nil && handle.X25519Private != nil {
			pub = handle.X25519Private.PublicKey()
		}
		jwk["x"] = base64.RawURLEncoding.EncodeToString(pub.Bytes())
		if handle.X25519Private != nil {
			jwk["d"] = base64.RawURLEncoding.EncodeToString(handle.X25519Private.Bytes())
		}
		return jwk, nil
	}

	return nil, errors.New("unsupported key type for JWK export")
}

func jwkAlgForHandle(handle *cryptoKeyHandle) string {
	algName := strings.TrimSpace(algorithmNameFromHandle(handle))
	if algName == "" {
		return ""
	}
	switch algName {
	case "AES-GCM":
		switch algorithmLengthFromHandle(handle) {
		case 128:
			return "A128GCM"
		case 192:
			return "A192GCM"
		case 256:
			return "A256GCM"
		}
	case "AES-KW":
		switch algorithmLengthFromHandle(handle) {
		case 128:
			return "A128KW"
		case 192:
			return "A192KW"
		case 256:
			return "A256KW"
		}
	case "HMAC":
		switch algorithmHashNameFromHandle(handle) {
		case "SHA-256":
			return "HS256"
		case "SHA-384":
			return "HS384"
		case "SHA-512":
			return "HS512"
		case "SHA-1":
			return "HS1"
		case "MD5":
			return "HMD5"
		}
	case "RSASSA-PKCS1-V1_5":
		switch algorithmHashNameFromHandle(handle) {
		case "SHA-256":
			return "RS256"
		case "SHA-384":
			return "RS384"
		case "SHA-512":
			return "RS512"
		case "SHA-1":
			return "RS1"
		}
	case "RSA-PSS":
		switch algorithmHashNameFromHandle(handle) {
		case "SHA-256":
			return "PS256"
		case "SHA-384":
			return "PS384"
		case "SHA-512":
			return "PS512"
		case "SHA-1":
			return "PS1"
		}
	case "RSA-OAEP":
		switch algorithmHashNameFromHandle(handle) {
		case "SHA-1":
			return "RSA-OAEP"
		case "SHA-256":
			return "RSA-OAEP-256"
		case "SHA-384":
			return "RSA-OAEP-384"
		case "SHA-512":
			return "RSA-OAEP-512"
		}
	case "RSAES-PKCS1-V1_5":
		return "RSA1_5"
	case "ECDSA":
		switch strings.ToUpper(strings.TrimSpace(algorithmNamedCurveFromHandle(handle))) {
		case "P-256":
			return "ES256"
		case "P-384":
			return "ES384"
		case "P-521":
			return "ES512"
		}
	case "Ed25519":
		return "EdDSA"
	case "ECDH", "X25519":
		return "ECDH-ES"
	}
	return algName
}

func algorithmNameFromHandle(handle *cryptoKeyHandle) string {
	if handle == nil || handle.Algorithm == nil {
		return ""
	}
	if v, ok := handle.Algorithm["name"].(string); ok {
		return v
	}
	return ""
}

func algorithmHashNameFromHandle(handle *cryptoKeyHandle) string {
	if handle == nil || handle.Algorithm == nil {
		return ""
	}
	hashValue, ok := handle.Algorithm["hash"]
	if !ok || hashValue == nil {
		if handle.HMACHash != "" {
			return handle.HMACHash
		}
		return ""
	}
	switch hv := hashValue.(type) {
	case string:
		return hv
	case map[string]interface{}:
		if name, ok := hv["name"].(string); ok {
			return name
		}
	}
	return ""
}

func algorithmLengthFromHandle(handle *cryptoKeyHandle) int {
	if handle == nil || handle.Algorithm == nil {
		return 0
	}
	if v, ok := handle.Algorithm["length"]; ok {
		switch vv := v.(type) {
		case int:
			return vv
		case int8:
			return int(vv)
		case int16:
			return int(vv)
		case int32:
			return int(vv)
		case int64:
			return int(vv)
		case float32:
			return int(vv)
		case float64:
			return int(vv)
		}
	}
	if len(handle.SecretKey) > 0 {
		return len(handle.SecretKey) * 8
	}
	return 0
}

func algorithmNamedCurveFromHandle(handle *cryptoKeyHandle) string {
	if handle == nil || handle.Algorithm == nil {
		return ""
	}
	if v, ok := handle.Algorithm["namedCurve"].(string); ok {
		return v
	}
	return ""
}

func parsePrivateKey(der []byte) (interface{}, error) {
	if v, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := v.(type) {
		case *rsa.PrivateKey:
			return key, nil
		case *ecdsa.PrivateKey:
			return key, nil
		case ed25519.PrivateKey:
			return key, nil
		case *ecdh.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("pkcs8 key is unsupported")
		}
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	return nil, errors.New("failed to parse private key")
}

func privateKeyToHandle(rt *goja.Runtime, privAny interface{}, algorithm string, algObj *goja.Object, extractable bool, usages []string) (*cryptoKeyHandle, error) {
	switch key := privAny.(type) {
	case *rsa.PrivateKey:
		switch algorithm {
		case "RSASSA-PKCS1-V1_5", "RSA-PSS", "RSA-OAEP", "RSAES-PKCS1-V1_5":
		default:
			return nil, errors.New("algorithm is not compatible with RSA private key")
		}
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return nil, err
		}
		return &cryptoKeyHandle{
			Type:        "private",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name": algorithm,
				"hash": map[string]interface{}{"name": hashName},
			},
			Usages:     usages,
			RSAPrivate: key,
		}, nil
	case *ecdsa.PrivateKey:
		if algorithm != "ECDSA" && algorithm != "ECDH" {
			return nil, errors.New("algorithm is not compatible with EC private key")
		}
		handle := &cryptoKeyHandle{
			Type:        "private",
			Extractable: extractable,
			Algorithm: map[string]interface{}{
				"name":       algorithm,
				"namedCurve": namedCurveFromElliptic(key.Curve),
			},
			Usages: usages,
		}
		if algorithm == "ECDSA" {
			handle.ECDSAPrivate = key
		} else {
			handle.ECDHPrivate = key
		}
		return handle, nil
	case ed25519.PrivateKey:
		if algorithm != "Ed25519" {
			return nil, errors.New("algorithm is not compatible with Ed25519 private key")
		}
		return &cryptoKeyHandle{
			Type:           "private",
			Extractable:    extractable,
			Algorithm:      map[string]interface{}{"name": "Ed25519"},
			Usages:         usages,
			Ed25519Private: key,
		}, nil
	case *ecdh.PrivateKey:
		if algorithm != "X25519" {
			return nil, errors.New("algorithm is not compatible with X25519 private key")
		}
		return &cryptoKeyHandle{
			Type:          "private",
			Extractable:   extractable,
			Algorithm:     map[string]interface{}{"name": "X25519"},
			Usages:        usages,
			X25519Private: key,
		}, nil
	default:
		return nil, errors.New("unsupported private key type")
	}
}

func publicKeyToHandle(rt *goja.Runtime, pubAny interface{}, algorithm string, algObj *goja.Object, usages []string) (*cryptoKeyHandle, error) {
	switch key := pubAny.(type) {
	case *rsa.PublicKey:
		switch algorithm {
		case "RSASSA-PKCS1-V1_5", "RSA-PSS", "RSA-OAEP", "RSAES-PKCS1-V1_5":
		default:
			return nil, errors.New("algorithm is not compatible with RSA public key")
		}
		hashName, err := hashFromAlgorithmObject(rt, algObj, "SHA-256")
		if err != nil {
			return nil, err
		}
		return &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm: map[string]interface{}{
				"name": algorithm,
				"hash": map[string]interface{}{"name": hashName},
			},
			Usages:    usages,
			RSAPublic: key,
		}, nil
	case *ecdsa.PublicKey:
		if algorithm != "ECDSA" && algorithm != "ECDH" {
			return nil, errors.New("algorithm is not compatible with EC public key")
		}
		handle := &cryptoKeyHandle{
			Type:        "public",
			Extractable: true,
			Algorithm: map[string]interface{}{
				"name":       algorithm,
				"namedCurve": namedCurveFromElliptic(key.Curve),
			},
			Usages: usages,
		}
		if algorithm == "ECDSA" {
			handle.ECDSAPublic = key
		} else {
			handle.ECDHPublic = key
		}
		return handle, nil
	case ed25519.PublicKey:
		if algorithm != "Ed25519" {
			return nil, errors.New("algorithm is not compatible with Ed25519 public key")
		}
		return &cryptoKeyHandle{
			Type:          "public",
			Extractable:   true,
			Algorithm:     map[string]interface{}{"name": "Ed25519"},
			Usages:        usages,
			Ed25519Public: key,
		}, nil
	case *ecdh.PublicKey:
		if algorithm != "X25519" {
			return nil, errors.New("algorithm is not compatible with X25519 public key")
		}
		return &cryptoKeyHandle{
			Type:         "public",
			Extractable:  true,
			Algorithm:    map[string]interface{}{"name": "X25519"},
			Usages:       usages,
			X25519Public: key,
		}, nil
	default:
		return nil, errors.New("unsupported public key type")
	}
}

func parseJWK(rt *goja.Runtime, value goja.Value) (map[string]interface{}, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, errors.New("JWK value is required")
	}
	exported := value.Export()
	if m, ok := exported.(map[string]interface{}); ok {
		return m, nil
	}
	obj := value.ToObject(rt)
	if obj == nil {
		return nil, errors.New("invalid JWK object")
	}
	exported = obj.Export()
	m, ok := exported.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid JWK object")
	}
	return m, nil
}

func parseJWKBigInt(jwk map[string]interface{}, key string) (*big.Int, error) {
	v := stringFromMap(jwk, key)
	if v == "" {
		return nil, fmt.Errorf("invalid RSA JWK: missing %s", key)
	}
	buf, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA JWK field %s", key)
	}
	n := new(big.Int).SetBytes(buf)
	if n.Sign() <= 0 {
		return nil, fmt.Errorf("invalid RSA JWK field %s", key)
	}
	return n, nil
}

func recoverRSAFactorsFromNED(n *big.Int, e int, d *big.Int) (*big.Int, *big.Int, error) {
	if n == nil || d == nil || e <= 1 {
		return nil, nil, errors.New("invalid RSA key parameters")
	}
	one := big.NewInt(1)
	two := big.NewInt(2)
	nMinus1 := new(big.Int).Sub(n, one)

	k := new(big.Int).Mul(d, big.NewInt(int64(e)))
	k.Sub(k, one)
	if k.Sign() <= 0 || k.Bit(0) != 0 {
		return nil, nil, errors.New("failed to recover RSA factors from n/e/d")
	}

	r := new(big.Int).Set(k)
	t := 0
	for r.Bit(0) == 0 {
		r.Rsh(r, 1)
		t++
	}

	maxCandidate := new(big.Int).Sub(n, two)
	if maxCandidate.Sign() <= 0 {
		return nil, nil, errors.New("failed to recover RSA factors from n/e/d")
	}

	tryBase := func(g *big.Int) (*big.Int, *big.Int, bool) {
		gcd := new(big.Int).GCD(nil, nil, g, n)
		if gcd.Cmp(one) > 0 && gcd.Cmp(n) < 0 {
			p := gcd
			q := new(big.Int).Div(new(big.Int).Set(n), p)
			if p.Cmp(q) > 0 {
				p, q = q, p
			}
			return p, q, true
		}

		y := new(big.Int).Exp(g, r, n)
		if y.Cmp(one) == 0 || y.Cmp(nMinus1) == 0 {
			return nil, nil, false
		}

		for range t {
			x := new(big.Int).Mul(y, y)
			x.Mod(x, n)

			if x.Cmp(one) == 0 {
				p := new(big.Int).Sub(y, one)
				p.GCD(nil, nil, p, n)
				if p.Cmp(one) <= 0 || p.Cmp(n) >= 0 {
					return nil, nil, false
				}
				q := new(big.Int).Div(new(big.Int).Set(n), p)
				if new(big.Int).Mul(new(big.Int).Set(p), new(big.Int).Set(q)).Cmp(n) != 0 {
					return nil, nil, false
				}
				if p.Cmp(q) > 0 {
					p, q = q, p
				}
				return p, q, true
			}

			if x.Cmp(nMinus1) == 0 {
				return nil, nil, false
			}
			y = x
		}
		return nil, nil, false
	}

	for base := int64(2); base < 8192; base++ {
		g := big.NewInt(base)
		if g.Cmp(nMinus1) >= 0 {
			break
		}
		if p, q, ok := tryBase(g); ok {
			return p, q, nil
		}
	}

	// 随机兜底，理论上不应该走到这里
	for range 8192 {
		g, err := rand.Int(rand.Reader, maxCandidate)
		if err != nil {
			return nil, nil, err
		}
		g.Add(g, two)
		if g.Cmp(nMinus1) >= 0 {
			g.Sub(g, one)
		}
		if p, q, ok := tryBase(g); ok {
			return p, q, nil
		}
	}
	return nil, nil, errors.New("failed to recover RSA factors from n/e/d")
}

func hasMapKey(m map[string]interface{}, k string) bool {
	_, ok := m[k]
	return ok
}

func stringFromMap(m map[string]interface{}, k string) string {
	v, ok := m[k]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func namedCurveByName(name string) (elliptic.Curve, string, error) {
	n := strings.ToUpper(strings.TrimSpace(name))
	switch n {
	case "P-256", "SECP256R1":
		return elliptic.P256(), "P-256", nil
	case "P-384", "SECP384R1":
		return elliptic.P384(), "P-384", nil
	case "P-521", "SECP521R1":
		return elliptic.P521(), "P-521", nil
	default:
		return nil, "", fmt.Errorf("unsupported namedCurve: %s", name)
	}
}

func ecdhCurveByElliptic(curve elliptic.Curve) (ecdh.Curve, error) {
	switch curve {
	case elliptic.P256():
		return ecdh.P256(), nil
	case elliptic.P384():
		return ecdh.P384(), nil
	case elliptic.P521():
		return ecdh.P521(), nil
	default:
		return nil, errors.New("unsupported ECDH curve")
	}
}

func namedCurveFromElliptic(curve elliptic.Curve) string {
	switch curve {
	case elliptic.P256():
		return "P-256"
	case elliptic.P384():
		return "P-384"
	case elliptic.P521():
		return "P-521"
	default:
		return ""
	}
}

func exportKeyBytesForWrap(rt *goja.Runtime, format string, handle *cryptoKeyHandle) ([]byte, error) {
	switch format {
	case "raw":
		return exportRawKeyMaterial(handle)
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
			return nil, errors.New("pkcs8 export requires a private key")
		}
		return x509.MarshalPKCS8PrivateKey(privAny)
	case "pkcs1":
		if handle.RSAPrivate != nil {
			return x509.MarshalPKCS1PrivateKey(handle.RSAPrivate), nil
		}
		pub := handle.RSAPublic
		if pub == nil && handle.RSAPrivate != nil {
			pub = &handle.RSAPrivate.PublicKey
		}
		if pub == nil {
			return nil, errors.New("pkcs1 export requires an RSA key")
		}
		return x509.MarshalPKCS1PublicKey(pub), nil
	case "sec1":
		var priv *ecdsa.PrivateKey
		if handle.ECDSAPrivate != nil {
			priv = handle.ECDSAPrivate
		} else if handle.ECDHPrivate != nil {
			priv = handle.ECDHPrivate
		} else {
			return nil, errors.New("sec1 export requires an EC private key")
		}
		return x509.MarshalECPrivateKey(priv)
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
			return nil, errors.New("spki export requires a public key")
		}
		return x509.MarshalPKIXPublicKey(pubAny)
	case "jwk":
		jwk, err := exportJWK(handle)
		if err != nil {
			return nil, err
		}
		return json.Marshal(jwk)
	default:
		return nil, fmt.Errorf("unsupported wrap key format: %s", format)
	}
}

func importUnwrappedKey(rt *goja.Runtime, format string, rawKey []byte, algorithm string, algObj *goja.Object, extractable bool, usages []string) (*cryptoKeyHandle, error) {
	switch format {
	case "raw":
		return importRawKey(rt, algorithm, algObj, rawKey, extractable, usages)
	case "jwk":
		var jwk map[string]interface{}
		if err := json.Unmarshal(rawKey, &jwk); err != nil {
			return nil, errors.New("invalid wrapped JWK data")
		}
		return importJWK(rt, jwk, algorithm, algObj, extractable, usages)
	case "pkcs8":
		privAny, err := parsePrivateKey(rawKey)
		if err != nil {
			return nil, err
		}
		return privateKeyToHandle(rt, privAny, algorithm, algObj, extractable, usages)
	case "pkcs1":
		if priv, err := x509.ParsePKCS1PrivateKey(rawKey); err == nil {
			return privateKeyToHandle(rt, priv, algorithm, algObj, extractable, usages)
		}
		pub, err := x509.ParsePKCS1PublicKey(rawKey)
		if err != nil {
			return nil, errors.New("failed to parse pkcs1 key")
		}
		return publicKeyToHandle(rt, pub, algorithm, algObj, usages)
	case "sec1":
		priv, err := x509.ParseECPrivateKey(rawKey)
		if err != nil {
			return nil, errors.New("failed to parse sec1 key")
		}
		return privateKeyToHandle(rt, priv, algorithm, algObj, extractable, usages)
	case "spki":
		pubAny, err := x509.ParsePKIXPublicKey(rawKey)
		if err != nil {
			return nil, err
		}
		return publicKeyToHandle(rt, pubAny, algorithm, algObj, usages)
	default:
		return nil, fmt.Errorf("unsupported unwrap key format: %s", format)
	}
}
