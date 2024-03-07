package crypto

import (
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
