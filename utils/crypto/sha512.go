package crypto

import (
	"crypto"
	"encoding/hex"
)

func CalculateSHA512Str(data []byte) string {
	return hex.EncodeToString(CalculateSHA512(data))
}

func CalculateSHA512(data []byte) []byte {
	hashInstance := crypto.SHA512.New()
	hashInstance.Write(data)
	hashed := hashInstance.Sum(nil)
	return hashed
}
