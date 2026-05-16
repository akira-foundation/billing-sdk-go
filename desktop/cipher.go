package desktop

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

const nonceLen = 12

// TokenCipher provides AES-256-GCM string encryption keyed by a 32-byte secret.
type TokenCipher struct {
	key [32]byte
}

func NewTokenCipher(key [32]byte) *TokenCipher {
	return &TokenCipher{key: key}
}

func (c *TokenCipher) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func (c *TokenCipher) Decrypt(encoded string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(bytes) <= nonceLen {
		return "", errors.New("ciphertext too short")
	}
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce, ct := bytes[:nonceLen], bytes[nonceLen:]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func GenerateKey() ([32]byte, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return key, err
	}
	return key, nil
}
