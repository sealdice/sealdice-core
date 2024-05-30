package crypto

import "crypto"

func CalculateSHA1(data []byte) []byte {
	hashInstance := crypto.SHA1.New()
	hashInstance.Write(data)
	hashed := hashInstance.Sum(nil)
	return hashed
}
