package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

type SymmetricCipher interface {
}

func Encrypt(text []byte, secret string) ([]byte, error) {
	c, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, text, nil), nil
}

func Decrypt(text []byte, secret string) (string, error) {
	c, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	t, err := gcm.Open(nil, text[:gcm.NonceSize()], text[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}

	return string(t), nil
}
