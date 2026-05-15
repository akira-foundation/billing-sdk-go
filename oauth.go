package billing

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
)

type PkceChallenge struct {
	Verifier  string
	Challenge string
	Method    string
}

func GeneratePkceChallenge() (PkceChallenge, error) {
	buf := make([]byte, 48)
	if _, err := rand.Read(buf); err != nil {
		return PkceChallenge{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)

	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	return PkceChallenge{Verifier: verifier, Challenge: challenge, Method: "S256"}, nil
}

func GenerateOauthState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type BuildOauthInitUrlOptions struct {
	BaseURL             string
	Provider            string
	Product             string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
}

func BuildOauthInitURL(opts BuildOauthInitUrlOptions) string {
	method := opts.CodeChallengeMethod
	if method == "" {
		method = "S256"
	}
	q := url.Values{}
	q.Set("product", opts.Product)
	q.Set("redirect_uri", opts.RedirectURI)
	q.Set("code_challenge", opts.CodeChallenge)
	q.Set("code_challenge_method", method)
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	base := strings.TrimRight(opts.BaseURL, "/")
	return base + "/auth/" + url.PathEscape(opts.Provider) + "?" + q.Encode()
}
