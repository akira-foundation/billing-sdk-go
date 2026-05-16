package desktop

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// KeyStore persists a long-lived AES-256 key. Primary backend is the OS
// keychain; an optional debug-mode file fallback keeps the key stable across
// unsigned rebuilds (macOS ad-hoc signatures lose keychain ACL on each build).
type KeyStore struct {
	keyring       TokenKeyring
	DebugFilePath string
}

func NewKeyStore(keyring TokenKeyring) KeyStore {
	return KeyStore{keyring: keyring}
}

func (k KeyStore) LoadOrCreate() ([32]byte, error) {
	if k.DebugFilePath != "" {
		if encoded, err := os.ReadFile(k.DebugFilePath); err == nil {
			if key, err := decodeKey(strings.TrimSpace(string(encoded))); err == nil {
				return key, nil
			}
		}
	}

	if encoded, ok, err := k.keyring.Get(); err == nil && ok {
		key, err := decodeKey(encoded)
		if err == nil {
			_ = k.writeDebugFile(encoded)
		}
		return key, err
	}

	key, err := GenerateKey()
	if err != nil {
		return key, err
	}
	encoded := base64.StdEncoding.EncodeToString(key[:])
	if err := k.keyring.Set(encoded); err != nil {
		_ = k.writeDebugFile(encoded)
	}
	_ = k.writeDebugFile(encoded)
	return key, nil
}

func (k KeyStore) writeDebugFile(encoded string) error {
	if k.DebugFilePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(k.DebugFilePath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(k.DebugFilePath, []byte(encoded), 0o600)
}

func decodeKey(encoded string) ([32]byte, error) {
	var key [32]byte
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return key, err
	}
	if len(bytes) != 32 {
		return key, errors.New("key has unexpected length")
	}
	copy(key[:], bytes)
	return key, nil
}
