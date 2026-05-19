package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestGeneratePkceChallengeMatchesVerifier(t *testing.T) {
	pkce, err := GeneratePkceChallenge()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if pkce.Method != "S256" {
		t.Fatalf("method: got %s", pkce.Method)
	}

	sum := sha256.Sum256([]byte(pkce.Verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	if pkce.Challenge != expected {
		t.Fatalf("challenge mismatch")
	}
}

func TestGenerateStateIsRandomAndUrlSafe(t *testing.T) {
	a, err := GenerateState()
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, _ := GenerateState()
	if a == b {
		t.Fatalf("expected different values")
	}
	for _, r := range a {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			t.Fatalf("non url-safe char: %c", r)
		}
	}
}

func TestBuildInitURL(t *testing.T) {
	raw := BuildInitURL(InitURLOptions{
		BaseURL:       "https://billing.akira.io/",
		Provider:      "google",
		Product:       "maintainer",
		RedirectURI:   "http://127.0.0.1:53000/cb",
		CodeChallenge: "abc",
		State:         "csrf-1",
	})

	if !strings.HasPrefix(raw, "https://billing.akira.io/auth/google?") {
		t.Fatalf("prefix wrong: %s", raw)
	}

	u, _ := url.Parse(raw)
	q := u.Query()
	if q.Get("product") != "maintainer" {
		t.Fatalf("product: %s", q.Get("product"))
	}
	if q.Get("redirect_uri") != "http://127.0.0.1:53000/cb" {
		t.Fatalf("redirect_uri: %s", q.Get("redirect_uri"))
	}
	if q.Get("code_challenge") != "abc" {
		t.Fatalf("code_challenge: %s", q.Get("code_challenge"))
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("method: %s", q.Get("code_challenge_method"))
	}
	if q.Get("state") != "csrf-1" {
		t.Fatalf("state: %s", q.Get("state"))
	}
}

func TestBuildInitURLOmitsState(t *testing.T) {
	raw := BuildInitURL(InitURLOptions{
		BaseURL:       "https://billing.akira.io",
		Provider:      "github",
		Product:       "m",
		RedirectURI:   "http://127.0.0.1:1/cb",
		CodeChallenge: "xyz",
	})

	u, _ := url.Parse(raw)
	if u.Query().Has("state") {
		t.Fatalf("state should be absent")
	}
}
