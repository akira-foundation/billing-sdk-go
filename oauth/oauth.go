// Package oauth provides PKCE primitives and the OAuth exchange / providers endpoints.
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

type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderGitHub    Provider = "github"
	ProviderApple     Provider = "apple"
	ProviderMicrosoft Provider = "microsoft"
	ProviderGitLab    Provider = "gitlab"
	ProviderBitbucket Provider = "bitbucket"
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

func GenerateState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type InitURLOptions struct {
	BaseURL             string
	Provider            string
	Product             string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
}

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

type ProviderInfo struct {
	Provider Provider `json:"provider"`
	Label    string   `json:"label"`
	Scopes   []string `json:"scopes"`
}

type ProvidersResponse struct {
	Product   string         `json:"product"`
	Providers []ProviderInfo `json:"providers"`
}

type ExchangePayload struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

type ExchangeCustomer struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	ProductID string  `json:"product_id"`
}

type ExchangeEntitlement struct {
	PlanKey *string `json:"plan_key"`
	Source  string  `json:"source"`
	EndsAt  *string `json:"ends_at"`
}

type ExchangeResponse struct {
	AccessToken           string               `json:"access_token"`
	TokenType             string               `json:"token_type"`
	Customer              ExchangeCustomer     `json:"customer"`
	Entitlement           *ExchangeEntitlement `json:"entitlement"`
	RequiresPlanSelection bool                 `json:"requires_plan_selection"`
}

func ListProviders(ctx context.Context, c *client.Client, product string) (*ProvidersResponse, error) {
	out := &ProvidersResponse{}
	path := "/api/v1/products/" + url.PathEscape(product) + "/auth/providers"
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Exchange completes the PKCE flow and stores the bearer on c.
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
