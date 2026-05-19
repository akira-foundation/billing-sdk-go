// Package github owns the GitHub App + installation endpoint helpers.
package github

import (
	"context"
	"encoding/json"

	"github.com/akira-io/billing-sdk-go/client"
)

// AppInfo mirrors GET /api/v1/github/app (unsigned).
type AppInfo struct {
	Slug       string `json:"slug"`
	InstallURL string `json:"install_url"`
}

// UserSummary describes the linked GitHub user.
type UserSummary struct {
	ID    uint64 `json:"id"`
	Login string `json:"login"`
}

// Installation describes one of the customer's GitHub installations.
type Installation struct {
	ID           uint64 `json:"id"`
	HTMLURL      string `json:"html_url"`
	AccountID    uint64 `json:"account_id"`
	AccountLogin string `json:"account_login"`
	AccountType  string `json:"account_type"`
	TargetType   string `json:"target_type"`
}

// InstallationsResponse mirrors GET /api/me/github/installations
type InstallationsResponse struct {
	User          UserSummary    `json:"user"`
	Installations []Installation `json:"installations"`
}

// InstallationTokenPayload mirrors POST /api/me/github/installation-token
type InstallationTokenPayload struct {
	InstallationID *uint64 `json:"installation_id,omitempty"`
}

// InstallationTokenResponse carries the short-lived installation token.
type InstallationTokenResponse struct {
	Token          string `json:"token"`
	ExpiresAt      string `json:"expires_at"`
	InstallationID int64  `json:"installation_id"`
	AccountLogin   string `json:"account_login"`
	AccountType    string `json:"account_type"`
}

// GetAppInfo returns public GitHub App metadata (slug + install URL). Unsigned.
func GetAppInfo(ctx context.Context, c *client.Client) (*AppInfo, error) {
	out := &AppInfo{}
	if err := c.DoPublic(ctx, "GET", "/api/v1/github/app", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Installations returns the authenticated customer's GitHub user info and installations.
func Installations(ctx context.Context, c *client.Client) (*InstallationsResponse, error) {
	out := &InstallationsResponse{}
	if err := c.Do(ctx, "GET", "/api/me/github/installations", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IssueInstallationToken mints a short-lived install token for the given installation.
func IssueInstallationToken(ctx context.Context, c *client.Client, payload InstallationTokenPayload) (*InstallationTokenResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &InstallationTokenResponse{}
	if err := c.Do(ctx, "POST", "/api/me/github/installation-token", body, out); err != nil {
		return nil, err
	}
	return out, nil
}
