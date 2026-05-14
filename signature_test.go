package billing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type vector struct {
	Name      string `json:"name"`
	Product   string `json:"product"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Body      string `json:"body"`
	Secret    string `json:"secret"`
	Canonical string `json:"canonical"`
	Signature string `json:"signature"`
}

func loadVectors(t *testing.T) []vector {
	t.Helper()

	path := filepath.Join("tests", "fixtures", "signature-vectors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	var v []vector
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode fixtures: %v", err)
	}

	return v
}

func TestCanonicalMatchesFixtures(t *testing.T) {
	for _, tc := range loadVectors(t) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			got := Canonical(tc.Product, tc.Timestamp, tc.Nonce, tc.Method, tc.Path, []byte(tc.Body))
			if got != tc.Canonical {
				t.Fatalf("canonical mismatch\nwant: %q\n got: %q", tc.Canonical, got)
			}
		})
	}
}

func TestSignMatchesFixtures(t *testing.T) {
	for _, tc := range loadVectors(t) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			canonical := Canonical(tc.Product, tc.Timestamp, tc.Nonce, tc.Method, tc.Path, []byte(tc.Body))
			got := Sign(tc.Secret, canonical)
			if got != tc.Signature {
				t.Fatalf("signature mismatch\nwant: %s\n got: %s", tc.Signature, got)
			}
		})
	}
}

func TestNonceLengthAndHex(t *testing.T) {
	nonce, err := NewNonce()
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	if len(nonce) != 32 {
		t.Fatalf("expected 32-char nonce, got %d", len(nonce))
	}
	for _, c := range nonce {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("nonce contains non-hex char: %q", c)
		}
	}
}
