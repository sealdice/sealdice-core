package crypto

import (
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func Sha256Checksum(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}

func CalculateSHA256Str(data []byte) string {
	return hex.EncodeToString(CalculateSHA256(data))
}

func CalculateSHA256(data []byte) []byte {
	hashInstance := crypto.SHA256.New()
	hashInstance.Write(data)
	hashed := hashInstance.Sum(nil)
	return hashed
}
