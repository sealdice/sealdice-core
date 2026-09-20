package sealcrypto

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/dop251/goja"
)

func parseAlgorithmIdentifier(rt *goja.Runtime, value goja.Value) (string, *goja.Object, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return "", nil, errors.New("algorithm is required")
	}

	if s, ok := value.Export().(string); ok {
		name, err := normalizeAlgorithmName(s)
		return name, nil, err
	}

	obj := value.ToObject(rt)
	if obj == nil {
		return "", nil, errors.New("algorithm must be string or object")
	}
	nameVal := obj.Get("name")
	if goja.IsUndefined(nameVal) || goja.IsNull(nameVal) {
		return "", nil, errors.New("algorithm.name is required")
	}
	name, err := normalizeAlgorithmName(nameVal.String())
	if err != nil {
		return "", nil, err
	}
	return name, obj, nil
}

func normalizeAlgorithmName(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("algorithm name is required")
	}
	n := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(raw, "_", "-"), " ", ""))
	switch n {
	case "MD5":
		return "MD5", nil
	case "SHA-1", "SHA1":
		return "SHA-1", nil
	case "SHA-224", "SHA224":
		return "SHA-224", nil
	case "SHA-256", "SHA256":
		return "SHA-256", nil
	case "SHA-384", "SHA384":
		return "SHA-384", nil
	case "SHA-512", "SHA512":
		return "SHA-512", nil
	case "AES-CBC", "AESCBC":
		return "AES-CBC", nil
	case "AES-GCM", "AESGCM":
		return "AES-GCM", nil
	case "AES-CTR", "AESCTR":
		return "AES-CTR", nil
	case "AES-KW", "AESKW":
		return "AES-KW", nil
	case "DES-CBC", "DESCBC":
		return "DES-CBC", nil
	case "3DES-CBC", "3DESCBC", "TRIPLEDES-CBC", "TRIPLEDESCBC", "DES-EDE3-CBC", "DES3-CBC":
		return "3DES-CBC", nil
	case "HMAC":
		return "HMAC", nil
	case "PBKDF2":
		return "PBKDF2", nil
	case "HKDF":
		return "HKDF", nil
	case "ECDSA":
		return "ECDSA", nil
	case "ECDH":
		return "ECDH", nil
	case "ED25519":
		return "Ed25519", nil
	case "X25519":
		return "X25519", nil
	case "RSA-PSS", "RSAPSS":
		return "RSA-PSS", nil
	case "RSASSA-PKCS1-V1_5", "RSASSA-PKCS1-V1.5", "RSASSA-PKCS1-V1-5", "RSASSAPKCS1V15", "RSASSA-PKCS1V1_5":
		return "RSASSA-PKCS1-V1_5", nil
	case "RSA-OAEP", "RSAOAEP":
		return "RSA-OAEP", nil
	case "RSAES-PKCS1-V1_5", "RSAES-PKCS1-V1.5", "RSAES-PKCS1-V1-5", "RSAESPKCS1V15":
		return "RSAES-PKCS1-V1_5", nil
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", raw)
	}
}

func intProperty(obj *goja.Object, key string) (int, error) {
	if obj == nil {
		return 0, fmt.Errorf("algorithm.%s is required", key)
	}
	v := obj.Get(key)
	if goja.IsUndefined(v) || goja.IsNull(v) {
		return 0, fmt.Errorf("algorithm.%s is required", key)
	}
	i := int(v.ToInteger())
	if i <= 0 {
		return 0, fmt.Errorf("algorithm.%s must be > 0", key)
	}
	return i, nil
}

func publicExponentFromAlg(rt *goja.Runtime, obj *goja.Object) (int, error) {
	if obj == nil {
		return 65537, nil
	}
	v := obj.Get("publicExponent")
	if goja.IsUndefined(v) || goja.IsNull(v) {
		return 65537, nil
	}
	b, err := bufferSourceBytes(rt, v, true, false)
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, errors.New("invalid publicExponent")
	}
	eBig := new(big.Int).SetBytes(b)
	if eBig.Sign() <= 0 || eBig.Bit(0) == 0 {
		return 0, errors.New("publicExponent must be an odd integer > 1")
	}
	maxInt := big.NewInt(int64(int(^uint(0) >> 1)))
	if eBig.Cmp(maxInt) > 0 {
		return 0, errors.New("publicExponent exceeds int range")
	}
	e := int(eBig.Int64())
	if e <= 1 || e%2 == 0 {
		return 0, errors.New("publicExponent must be an odd integer > 1")
	}
	return e, nil
}

func requiredBufferProperty(rt *goja.Runtime, obj *goja.Object, key string) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("algorithm.%s is required", key)
	}
	v := obj.Get(key)
	if goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, fmt.Errorf("algorithm.%s is required", key)
	}
	return bufferSourceBytes(rt, v, true, false)
}

func isValuePresent(v goja.Value) bool {
	return v != nil && !goja.IsUndefined(v) && !goja.IsNull(v)
}

func valueToStringSlice(value goja.Value) ([]string, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	exported := value.Export()
	switch vv := exported.(type) {
	case []interface{}:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			s, ok := item.(string)
			if !ok {
				return nil, errors.New("usage list must contain strings")
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		out := make([]string, len(vv))
		copy(out, vv)
		return out, nil
	default:
		return nil, errors.New("usage list must be an array")
	}
}

func usageContains(usages []string, target string) bool {
	for _, u := range usages {
		if u == target {
			return true
		}
	}
	return false
}

func rawKeyTypeHint(algObj *goja.Object) (string, error) {
	if algObj == nil {
		return "", nil
	}
	keys := []string{"keyType", "type"}
	for _, key := range keys {
		v := algObj.Get(key)
		if !isValuePresent(v) {
			continue
		}
		raw := strings.TrimSpace(strings.ToLower(v.String()))
		switch raw {
		case "public":
			return "public", nil
		case "private":
			return "private", nil
		default:
			return "", fmt.Errorf("algorithm.%s must be 'public' or 'private'", key)
		}
	}
	return "", nil
}

func bufferSourceBytes(rt *goja.Runtime, value goja.Value, allowArrayBuffer bool, requireIntegerTypedArray bool) ([]byte, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, errors.New("input is required")
	}

	if allowArrayBuffer {
		if arrayBuffer, ok := value.Export().(goja.ArrayBuffer); ok {
			return arrayBuffer.Bytes(), nil
		}
	}

	obj := value.ToObject(rt)
	if obj == nil {
		return nil, errors.New("input must be an ArrayBuffer or an ArrayBufferView")
	}

	if requireIntegerTypedArray {
		ctorName := constructorName(rt, obj)
		if _, ok := integerTypedArrayNames[ctorName]; !ok {
			return nil, errors.New("input must be an integer TypedArray")
		}
	}

	bufferVal := obj.Get("buffer")
	if goja.IsUndefined(bufferVal) || goja.IsNull(bufferVal) {
		return nil, errors.New("input must be an ArrayBufferView")
	}

	arrayBuffer, ok := bufferVal.Export().(goja.ArrayBuffer)
	if !ok {
		return nil, errors.New("input buffer must be an ArrayBuffer")
	}

	offset := int(obj.Get("byteOffset").ToInteger())
	length := int(obj.Get("byteLength").ToInteger())
	bufferBytes := arrayBuffer.Bytes()
	if offset < 0 || length < 0 || offset+length > len(bufferBytes) {
		return nil, errors.New("invalid buffer range")
	}

	return bufferBytes[offset : offset+length], nil
}

func constructorName(rt *goja.Runtime, obj *goja.Object) string {
	ctor := obj.Get("constructor")
	if goja.IsUndefined(ctor) || goja.IsNull(ctor) {
		return ""
	}
	ctorObj := ctor.ToObject(rt)
	name := ctorObj.Get("name")
	if goja.IsUndefined(name) || goja.IsNull(name) {
		return ""
	}
	return name.String()
}

func newCryptoKeyObject(rt *goja.Runtime, handle *cryptoKeyHandle) *goja.Object {
	obj := rt.NewObject()
	_ = obj.Set("type", handle.Type)
	_ = obj.Set("extractable", handle.Extractable)
	_ = obj.Set("algorithm", cloneMap(handle.Algorithm))
	_ = obj.Set("usages", append([]string{}, handle.Usages...))
	_ = obj.Set(keyHandleSlot, handle)
	return obj
}

func extractCryptoKeyHandle(rt *goja.Runtime, value goja.Value) (*cryptoKeyHandle, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, errors.New("CryptoKey is required")
	}
	if h, ok := value.Export().(*cryptoKeyHandle); ok {
		return h, nil
	}
	obj := value.ToObject(rt)
	if obj == nil {
		return nil, errors.New("invalid CryptoKey")
	}
	hVal := obj.Get(keyHandleSlot)
	if goja.IsUndefined(hVal) || goja.IsNull(hVal) {
		return nil, errors.New("invalid CryptoKey handle")
	}
	h, ok := hVal.Export().(*cryptoKeyHandle)
	if !ok || h == nil {
		return nil, errors.New("invalid CryptoKey handle")
	}
	return h, nil
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		switch vv := v.(type) {
		case map[string]interface{}:
			dst[k] = cloneMap(vv)
		case []byte:
			cp := make([]byte, len(vv))
			copy(cp, vv)
			dst[k] = cp
		default:
			dst[k] = vv
		}
	}
	return dst
}

func leftPad(in []byte, size int) []byte {
	if len(in) >= size {
		cp := make([]byte, len(in))
		copy(cp, in)
		return cp
	}
	out := make([]byte, size)
	copy(out[size-len(in):], in)
	return out
}

func resolvedPromise(rt *goja.Runtime, value interface{}) goja.Value {
	p, resolve, _ := rt.NewPromise()
	_ = resolve(value)
	return rt.ToValue(p)
}

func rejectedPromise(rt *goja.Runtime, err error) goja.Value {
	p, _, reject := rt.NewPromise()
	_ = reject(rt.NewTypeError(err.Error()))
	return rt.ToValue(p)
}
