package signature

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	HeaderProduct   = "X-Akira-Product"
	HeaderTimestamp = "X-Akira-Timestamp"
	HeaderNonce     = "X-Akira-Nonce"
	HeaderSignature = "X-Akira-Signature"
)

func Canonical(product string, timestamp int64, nonce, method, path string, body []byte) string {
	sum := sha256.Sum256(body)

	return strings.Join([]string{
		product,
		fmt.Sprintf("%d", timestamp),
		nonce,
		strings.ToUpper(method),
		path,
		hex.EncodeToString(sum[:]),
	}, "\n")
}

func Sign(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))

	return hex.EncodeToString(mac.Sum(nil))
}

func NewNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("billing: nonce: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
