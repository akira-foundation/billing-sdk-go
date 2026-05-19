// Package oauth provides PKCE primitives, the OAuth init-URL builder, and the
// authenticated exchange / providers endpoint helpers.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/akira-io/billing-sdk-go/client"
)

// Provider is the canonical lowercase identifier of an OAuth provider.
type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderGitHub    Provider = "github"
	ProviderApple     Provider = "apple"
	ProviderMicrosoft Provider = "microsoft"
	ProviderGitLab    Provider = "gitlab"
	ProviderBitbucket Provider = "bitbucket"
)

// PkceChallenge carries the PKCE verifier, the SHA-256 challenge derived from
// it, and the fixed S256 method identifier.
type PkceChallenge struct {
	Verifier  string
	Challenge string
	Method    string
}

// GeneratePkceChallenge returns a fresh PKCE pair using crypto/rand.
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

// GenerateState returns 24 bytes of CSPRNG encoded as URL-safe base64.
func GenerateState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// InitURLOptions configures BuildInitURL.
type InitURLOptions struct {
	BaseURL             string
	Provider            string
	Product             string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
}

// BuildInitURL builds the provider authorization URL.
func BuildInitURL(opts InitURLOptions) string {
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

// ProviderInfo describes one available provider for a product.
type ProviderInfo struct {
	Provider Provider `json:"provider"`
	Label    string   `json:"label"`
	Scopes   []string `json:"scopes"`
}

// ProvidersResponse mirrors GET /api/v1/products/{product}/auth/providers
type ProvidersResponse struct {
	Product   string         `json:"product"`
	Providers []ProviderInfo `json:"providers"`
}

// ExchangePayload mirrors POST /api/auth/oauth/exchange
type ExchangePayload struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

// ExchangeCustomer is the customer descriptor returned by the exchange.
type ExchangeCustomer struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	ProductID string  `json:"product_id"`
}

// ExchangeEntitlement is the optional entitlement returned by the exchange.
type ExchangeEntitlement struct {
	PlanKey *string `json:"plan_key"`
	Source  string  `json:"source"`
	EndsAt  *string `json:"ends_at"`
}

// ExchangeResponse carries the access token and the customer/entitlement state.
type ExchangeResponse struct {
	AccessToken           string               `json:"access_token"`
	TokenType             string               `json:"token_type"`
	Customer              ExchangeCustomer     `json:"customer"`
	Entitlement           *ExchangeEntitlement `json:"entitlement"`
	RequiresPlanSelection bool                 `json:"requires_plan_selection"`
}

// ListProviders returns the OAuth providers available for the given product.
func ListProviders(ctx context.Context, c *client.Client, product string) (*ProvidersResponse, error) {
	out := &ProvidersResponse{}
	path := "/api/v1/products/" + url.PathEscape(product) + "/auth/providers"
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Exchange completes the PKCE OAuth flow and stores the bearer on c.
func Exchange(ctx context.Context, c *client.Client, payload ExchangePayload) (*ExchangeResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &ExchangeResponse{}
	if err := c.Do(ctx, "POST", "/api/auth/oauth/exchange", body, out); err != nil {
		return nil, err
	}
	c.SetCustomerToken(out.AccessToken)
	return out, nil
}
