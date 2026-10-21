package sealcrypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"errors"
)

func exportRawKeyMaterial(handle *cryptoKeyHandle) ([]byte, error) {
	if len(handle.SecretKey) != 0 {
		out := make([]byte, len(handle.SecretKey))
		copy(out, handle.SecretKey)
		return out, nil
	}

	if pub := ecPublicFromHandle(handle); pub != nil {
		return marshalECRawPublicKey(pub), nil
	}

	pubEd := handle.Ed25519Public
	if len(pubEd) == 0 && len(handle.Ed25519Private) != 0 {
		pubEd = handle.Ed25519Private.Public().(ed25519.PublicKey)
	}
	if len(pubEd) != 0 {
		out := make([]byte, len(pubEd))
		copy(out, pubEd)
		return out, nil
	}

	pubX := handle.X25519Public
	if pubX == nil && handle.X25519Private != nil {
		pubX = handle.X25519Private.PublicKey()
	}
	if pubX != nil {
		out := pubX.Bytes()
		cp := make([]byte, len(out))
		copy(cp, out)
		return cp, nil
	}

	return nil, errors.New("raw export requires a secret key or supported public key")
}

func ecPublicFromHandle(handle *cryptoKeyHandle) *ecdsa.PublicKey {
	if handle.ECDSAPublic != nil {
		return handle.ECDSAPublic
	}
	if handle.ECDSAPrivate != nil {
		return &handle.ECDSAPrivate.PublicKey
	}
	if handle.ECDHPublic != nil {
		return handle.ECDHPublic
	}
	if handle.ECDHPrivate != nil {
		return &handle.ECDHPrivate.PublicKey
	}
	return nil
}

func marshalECRawPublicKey(pub *ecdsa.PublicKey) []byte {
	size := (pub.Curve.Params().BitSize + 7) / 8
	out := make([]byte, 1+2*size)
	out[0] = 0x04
	copy(out[1:1+size], leftPad(pub.X.Bytes(), size))
	copy(out[1+size:], leftPad(pub.Y.Bytes(), size))
	return out
}
