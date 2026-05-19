package github

import (
	"context"
	"encoding/json"

	"github.com/akira-io/billing-sdk-go/client"
)

type AppInfo struct {
	Slug       string `json:"slug"`
	InstallURL string `json:"install_url"`
}

type UserSummary struct {
	ID    uint64 `json:"id"`
	Login string `json:"login"`
}

type Installation struct {
	ID           uint64 `json:"id"`
	HTMLURL      string `json:"html_url"`
	AccountID    uint64 `json:"account_id"`
	AccountLogin string `json:"account_login"`
	AccountType  string `json:"account_type"`
	TargetType   string `json:"target_type"`
}

type InstallationsResponse struct {
	User          UserSummary    `json:"user"`
	Installations []Installation `json:"installations"`
}

type InstallationTokenPayload struct {
	InstallationID *uint64 `json:"installation_id,omitempty"`
}

type InstallationTokenResponse struct {
	Token          string `json:"token"`
	ExpiresAt      string `json:"expires_at"`
	InstallationID int64  `json:"installation_id"`
	AccountLogin   string `json:"account_login"`
	AccountType    string `json:"account_type"`
}

type UserTokenResponse struct {
	Token     string  `json:"token"`
	ExpiresAt *string `json:"expires_at"`
}

func GetAppInfo(ctx context.Context, c *client.Client) (*AppInfo, error) {
	out := &AppInfo{}
	if err := c.DoPublic(ctx, "GET", "/api/v1/github/app", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Installations(ctx context.Context, c *client.Client) (*InstallationsResponse, error) {
	out := &InstallationsResponse{}
	if err := c.Do(ctx, "GET", "/api/me/github/installations", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

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

func UserToken(ctx context.Context, c *client.Client) (*UserTokenResponse, error) {
	out := &UserTokenResponse{}
	if err := c.Do(ctx, "GET", "/api/me/github/user-token", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
