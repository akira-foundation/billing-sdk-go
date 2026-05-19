// Package signature implements the Akira Billing HMAC-SHA256 request signing
// protocol. The Go, Rust, and TypeScript SDKs share fixture vectors at
// tests/fixtures/signature-vectors.json.
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

// Canonical builds the canonical string that gets HMAC'd. Layout:
//
//	{product}\n{timestamp}\n{nonce}\n{METHOD}\n{path}\n{sha256(body)}
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

// Sign returns the lowercase-hex HMAC-SHA256 of the canonical string under the given secret.
func Sign(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))

	return hex.EncodeToString(mac.Sum(nil))
}

// NewNonce returns a 16-byte random nonce encoded as lowercase hex (32 chars).
func NewNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("billing: nonce: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
