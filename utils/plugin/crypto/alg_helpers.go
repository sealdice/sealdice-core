//nolint:gosec // WebCrypto legacy compatibility intentionally includes MD5/SHA-1.
package sealcrypto

import (
	"crypto"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"math/big"

	"github.com/dop251/goja"
)

func hashFromAlgorithmObject(rt *goja.Runtime, obj *goja.Object, defaultName string) (string, error) {
	if obj == nil {
		return normalizeAlgorithmName(defaultName)
	}
	v := obj.Get("hash")
	if !isValuePresent(v) {
		return normalizeAlgorithmName(defaultName)
	}
	return hashFromAlgorithmValue(rt, v)
}

func hashFromAlgorithmValue(rt *goja.Runtime, value goja.Value) (string, error) {
	if !isValuePresent(value) {
		return "", errors.New("invalid hash algorithm")
	}
	if s, ok := value.Export().(string); ok {
		return normalizeAlgorithmName(s)
	}
	obj := value.ToObject(rt)
	if obj == nil {
		return "", errors.New("invalid hash algorithm")
	}
	name := obj.Get("name")
	if goja.IsUndefined(name) || goja.IsNull(name) {
		return "", errors.New("hash.name is required")
	}
	return normalizeAlgorithmName(name.String())
}

func hashFactory(name string) (func() hash.Hash, error) {
	switch name {
	case "MD5":
		return md5.New, nil
	case "SHA-1":
		return sha1.New, nil
	case "SHA-224":
		return sha256.New224, nil
	case "SHA-256":
		return sha256.New, nil
	case "SHA-384":
		return sha512.New384, nil
	case "SHA-512":
		return sha512.New, nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", name)
	}
}

func digestBytes(algorithm string, data []byte) ([]byte, error) {
	switch algorithm {
	case "MD5":
		sum := md5.Sum(data)
		out := make([]byte, len(sum))
		copy(out, sum[:])
		return out, nil
	case "SHA-1":
		sum := sha1.Sum(data)
		out := make([]byte, len(sum))
		copy(out, sum[:])
		return out, nil
	case "SHA-224":
		sum := sha256.Sum224(data)
		out := make([]byte, len(sum))
		copy(out, sum[:])
		return out, nil
	case "SHA-256":
		sum := sha256.Sum256(data)
		out := make([]byte, len(sum))
		copy(out, sum[:])
		return out, nil
	case "SHA-384":
		sum := sha512.Sum384(data)
		out := make([]byte, len(sum))
		copy(out, sum[:])
		return out, nil
	case "SHA-512":
		sum := sha512.Sum512(data)
		out := make([]byte, len(sum))
		copy(out, sum[:])
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported digest algorithm: %s", algorithm)
	}
}

func digestForCryptoHash(hashName string, data []byte) (crypto.Hash, []byte, error) {
	switch hashName {
	case "MD5":
		sum := md5.Sum(data)
		return crypto.MD5, sum[:], nil
	case "SHA-1":
		sum := sha1.Sum(data)
		return crypto.SHA1, sum[:], nil
	case "SHA-224":
		sum := sha256.Sum224(data)
		return crypto.SHA224, sum[:], nil
	case "SHA-256":
		sum := sha256.Sum256(data)
		return crypto.SHA256, sum[:], nil
	case "SHA-384":
		sum := sha512.Sum384(data)
		return crypto.SHA384, sum[:], nil
	case "SHA-512":
		sum := sha512.Sum512(data)
		return crypto.SHA512, sum[:], nil
	default:
		return 0, nil, fmt.Errorf("unsupported hash for RSA: %s", hashName)
	}
}

func digestLengthBytes(hashName string) int {
	switch hashName {
	case "MD5":
		return md5.Size
	case "SHA-1":
		return sha1.Size
	case "SHA-224":
		return sha256.Size224
	case "SHA-256":
		return sha256.Size
	case "SHA-384":
		return sha512.Size384
	case "SHA-512":
		return sha512.Size
	default:
		return 32
	}
}

func validateAESGCMTagLength(tagLength int) error {
	switch tagLength {
	case 32, 64, 96, 104, 112, 120, 128:
		return nil
	default:
		return errors.New("invalid AES-GCM tagLength")
	}
}

func newAESGCMForParams(block cipher.Block, ivLen int, tagLengthBits int) (cipher.AEAD, error) {
	tagBytes := tagLengthBits / 8
	if ivLen == 12 {
		return cipher.NewGCMWithTagSize(block, tagBytes)
	}
	if tagBytes != 16 {
		return nil, errors.New("AES-GCM non-12-byte iv requires tagLength 128")
	}
	return cipher.NewGCMWithNonceSize(block, ivLen)
}

func generateRSAKeyWithExponent(modulusLength int, exp int) (*rsa.PrivateKey, error) {
	if exp <= 1 || exp%2 == 0 {
		return nil, errors.New("publicExponent must be an odd integer > 1")
	}
	if exp == 65537 {
		return rsa.GenerateKey(rand.Reader, modulusLength)
	}
	return generateRSAKeyCustomExponent(modulusLength, exp)
}

func generateRSAKeyCustomExponent(modulusLength int, exp int) (*rsa.PrivateKey, error) {
	one := big.NewInt(1)
	eBig := big.NewInt(int64(exp))
	bitsP := modulusLength / 2
	bitsQ := modulusLength - bitsP

	for range 256 {
		p, err := rand.Prime(rand.Reader, bitsP)
		if err != nil {
			return nil, err
		}
		q, err := rand.Prime(rand.Reader, bitsQ)
		if err != nil {
			return nil, err
		}
		if p.Cmp(q) == 0 {
			continue
		}

		n := new(big.Int).Mul(p, q)
		if n.BitLen() != modulusLength {
			continue
		}

		pm1 := new(big.Int).Sub(p, one)
		qm1 := new(big.Int).Sub(q, one)
		if new(big.Int).GCD(nil, nil, eBig, pm1).Cmp(one) != 0 {
			continue
		}
		if new(big.Int).GCD(nil, nil, eBig, qm1).Cmp(one) != 0 {
			continue
		}

		phi := new(big.Int).Mul(pm1, qm1)
		d := new(big.Int).ModInverse(eBig, phi)
		if d == nil {
			continue
		}

		priv := &rsa.PrivateKey{
			PublicKey: rsa.PublicKey{N: n, E: exp},
			D:         d,
			Primes:    []*big.Int{p, q},
		}
		if err := priv.Validate(); err != nil {
			continue
		}
		priv.Precompute()
		return priv, nil
	}
	return nil, errors.New("failed to generate RSA key with requested publicExponent")
}

func hashNameForRSA(rt *goja.Runtime, algObj *goja.Object, key *cryptoKeyHandle, defaultName string) (string, error) {
	hashName := defaultName
	if key != nil && key.Algorithm != nil {
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
				return "", err
			}
		}
	}
	return normalizeAlgorithmName(hashName)
}
