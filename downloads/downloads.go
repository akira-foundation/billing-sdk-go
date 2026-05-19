// Package downloads owns the plans, trial activation, and download endpoint
// helpers.
package downloads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/akira-io/billing-sdk-go/client"
)

// PlansResponse mirrors GET /api/v1/products/{key}/plans
type PlansResponse struct {
	Product     string  `json:"product"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	LandingURL  *string `json:"landing_url"`
	BetaEndsAt  *string `json:"beta_ends_at"`
	BetaActive  bool    `json:"beta_active"`
	Plans       []Plan  `json:"plans"`
}

// Plan describes one plan available on the product.
type Plan struct {
	ID              string        `json:"id"`
	Key             string        `json:"key"`
	Name            string        `json:"name"`
	Description     *string       `json:"description"`
	Amount          *int          `json:"amount"`
	Currency        *string       `json:"currency"`
	BillingInterval *string       `json:"billing_interval"`
	TrialPeriodDays int           `json:"trial_period_days"`
	IsComingSoon    bool          `json:"is_coming_soon"`
	Features        []PlanFeature `json:"features"`
}

// PlanFeature describes a feature attached to a plan.
type PlanFeature struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// IssuedTrial mirrors POST /api/v1/me/products/{key}/trial
type IssuedTrial struct {
	Product         string    `json:"product"`
	Plan            *string   `json:"plan"`
	Source          string    `json:"source"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	TrialPeriodDays *int      `json:"trial_period_days"`
}

// ReleaseAsset describes a single downloadable artifact in a release manifest.
type ReleaseAsset struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Format    string `json:"format"`
	ObjectKey string `json:"object_key"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// ReleaseManifest mirrors GET /api/v1/downloads/{product}/releases/{channel}/latest
type ReleaseManifest struct {
	Version    string         `json:"version"`
	Channel    string         `json:"channel"`
	ReleasedAt time.Time      `json:"released_at"`
	NotesURL   *string        `json:"notes_url"`
	Assets     []ReleaseAsset `json:"assets"`
}

// IssuedDownload mirrors GET /api/v1/downloads/{product}/{channel}/{platform} (Accept: application/json)
type IssuedDownload struct {
	EventID   string    `json:"eventId"`
	Product   string    `json:"product"`
	Version   string    `json:"version"`
	Channel   string    `json:"channel"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Format    string    `json:"format"`
	SizeBytes int64     `json:"sizeBytes"`
	SHA256    string    `json:"sha256"`
	SignedURL string    `json:"signedUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
	BeaconURL string    `json:"beaconUrl"`
}

// Plans fetches the public plans payload for the configured product.
func Plans(ctx context.Context, c *client.Client) (*PlansResponse, error) {
	out := &PlansResponse{}
	if err := c.Do(ctx, "GET", "/api/v1/products/"+c.ProductSlug+"/plans", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// StartTrial activates the optional trial plan for the configured product.
func StartTrial(ctx context.Context, c *client.Client, planKey string) (*IssuedTrial, error) {
	body := []byte(`{}`)
	if planKey != "" {
		body, _ = json.Marshal(map[string]string{"plan": planKey})
	}

	out := &IssuedTrial{}
	if err := c.Do(ctx, "POST", "/api/v1/me/products/"+c.ProductSlug+"/trial", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// LatestRelease fetches the current release manifest for a channel.
// Channel is one of "stable", "beta", "nightly".
func LatestRelease(ctx context.Context, c *client.Client, channel string) (*ReleaseManifest, error) {
	out := &ReleaseManifest{}
	path := "/api/v1/downloads/" + c.ProductSlug + "/releases/" + channel + "/latest"
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IssueDownload requests a signed URL for the matching asset and records a
// DownloadEvent on the backend. Platform is "os-arch", e.g. "macos-arm64".
func IssueDownload(ctx context.Context, c *client.Client, channel, platform string) (*IssuedDownload, error) {
	out := &IssuedDownload{}
	path := "/api/v1/downloads/" + c.ProductSlug + "/" + channel + "/" + platform
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompleteDownload posts the completion beacon for an issued event. The beacon
// URL is the absolute URL returned in IssuedDownload.BeaconURL, which already
// carries the signature query string. Unsigned.
func CompleteDownload(ctx context.Context, c *client.Client, beaconURL string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", beaconURL, nil)
	if err != nil {
		return fmt.Errorf("billing: build beacon request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("billing: complete download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		apiErr := &client.APIError{Status: resp.StatusCode}
		_ = json.Unmarshal(raw, apiErr)
		if apiErr.Code == "" && apiErr.Message == "" {
			apiErr.Code = string(raw)
		}
		return apiErr
	}
	return nil
}
