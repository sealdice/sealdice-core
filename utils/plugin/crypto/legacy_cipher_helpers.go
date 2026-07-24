package sealcrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"encoding/hex"
	"errors"
)

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - len(data)%blockSize
	if padLen == 0 {
		padLen = blockSize
	}
	pad := bytes.Repeat([]byte{byte(padLen)}, padLen)
	out := make([]byte, 0, len(data)+padLen)
	out = append(out, data...)
	out = append(out, pad...)
	return out
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid PKCS7 padding")
	}
	padLen := int(data[len(data)-1])
	if padLen <= 0 || padLen > blockSize || padLen > len(data) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if int(data[i]) != padLen {
			return nil, errors.New("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-padLen], nil
}

func aesKeyWrap(kek []byte, plaintext []byte) ([]byte, error) {
	if len(plaintext) < 16 || len(plaintext)%8 != 0 {
		return nil, errors.New("AES-KW plaintext length must be multiple of 8 and at least 16")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	n := len(plaintext) / 8
	a := []byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}
	r := make([][]byte, n)
	for i := range n {
		r[i] = make([]byte, 8)
		copy(r[i], plaintext[i*8:(i+1)*8])
	}

	buf := make([]byte, 16)
	for j := range 6 {
		for i := range n {
			copy(buf[:8], a)
			copy(buf[8:], r[i])
			block.Encrypt(buf, buf)
			t := uint64(n*j + i + 1)
			xorT(buf[:8], t)
			copy(a, buf[:8])
			copy(r[i], buf[8:])
		}
	}

	out := make([]byte, 8+8*n)
	copy(out[:8], a)
	for i := range n {
		copy(out[8+i*8:8+(i+1)*8], r[i])
	}
	return out, nil
}

func aesKeyUnwrap(kek []byte, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 24 || len(ciphertext)%8 != 0 {
		return nil, errors.New("AES-KW ciphertext length must be multiple of 8 and at least 24")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	n := len(ciphertext)/8 - 1
	a := make([]byte, 8)
	copy(a, ciphertext[:8])
	r := make([][]byte, n)
	for i := range n {
		r[i] = make([]byte, 8)
		copy(r[i], ciphertext[8+i*8:8+(i+1)*8])
	}

	buf := make([]byte, 16)
	for j := 5; j >= 0; j-- {
		for i := n - 1; i >= 0; i-- {
			copy(buf[:8], a)
			t := uint64(n*j + i + 1)
			xorT(buf[:8], t)
			copy(buf[8:], r[i])
			block.Decrypt(buf, buf)
			copy(a, buf[:8])
			copy(r[i], buf[8:])
		}
	}
	iv := []byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}
	if !bytes.Equal(a, iv) {
		return nil, errors.New("AES-KW integrity check failed")
	}

	out := make([]byte, 8*n)
	for i := range n {
		copy(out[i*8:(i+1)*8], r[i])
	}
	return out, nil
}

func xorT(a []byte, t uint64) {
	for i := 7; i >= 0; i-- {
		a[i] ^= byte(t & 0xff)
		t >>= 8
	}
}

func tripleDESKey(raw []byte) ([]byte, error) {
	switch len(raw) {
	case 24:
		out := make([]byte, 24)
		copy(out, raw)
		return out, nil
	case 16:
		// 2-key 3DES: K1 || K2 || K1
		out := make([]byte, 24)
		copy(out[:16], raw)
		copy(out[16:], raw[:8])
		return out, nil
	default:
		return nil, errors.New("3DES-CBC requires a 16-byte or 24-byte secret key")
	}
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])
	return string(out[:])
}
