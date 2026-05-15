package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// PlansResponse mirrors GET /api/v1/products/{key}/plans
type PlansResponse struct {
	Product     string    `json:"product"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	LandingURL  *string   `json:"landing_url"`
	BetaEndsAt  *string   `json:"beta_ends_at"`
	BetaActive  bool      `json:"beta_active"`
	Plans       []APIPlan `json:"plans"`
}

type APIPlan struct {
	ID              string           `json:"id"`
	Key             string           `json:"key"`
	Name            string           `json:"name"`
	Description     *string          `json:"description"`
	Amount          *int             `json:"amount"`
	Currency        *string          `json:"currency"`
	BillingInterval *string          `json:"billing_interval"`
	TrialPeriodDays int              `json:"trial_period_days"`
	IsComingSoon    bool             `json:"is_coming_soon"`
	Features        []APIPlanFeature `json:"features"`
}

type APIPlanFeature struct {
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

// OtpRequestPayload mirrors POST /api/auth/customer/otp/request
type OtpRequestPayload struct {
	Email      string `json:"email"`
	DeviceFP   string `json:"device_fp,omitempty"`
	Platform   string `json:"platform,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
}

// OtpVerifyPayload mirrors POST /api/auth/customer/otp/verify
type OtpVerifyPayload struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	DeviceFP string `json:"device_fp,omitempty"`
}

// OtpVerifyResponse holds the Sanctum token plus a minimal customer descriptor.
type OtpVerifyResponse struct {
	AccessToken string `json:"access_token"`
	Customer    struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"customer"`
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

// LatestRelease fetches the current release manifest for a channel.
// Channel is one of "stable", "beta", "nightly".
func (c *Client) LatestRelease(ctx context.Context, channel string) (*ReleaseManifest, error) {
	out := &ReleaseManifest{}
	path := "/api/v1/downloads/" + c.ProductSlug + "/releases/" + channel + "/latest"
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IssueDownload requests a signed URL for the matching asset and records a
// DownloadEvent on the backend. Platform is "os-arch", e.g. "macos-arm64".
func (c *Client) IssueDownload(ctx context.Context, channel, platform string) (*IssuedDownload, error) {
	out := &IssuedDownload{}
	path := "/api/v1/downloads/" + c.ProductSlug + "/" + channel + "/" + platform
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompleteDownload posts the completion beacon for an issued event. The
// beacon URL is the absolute URL returned in IssuedDownload.BeaconURL, which
// already carries the signature query string. No HMAC signing.
func (c *Client) CompleteDownload(ctx context.Context, beaconURL string) error {
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
		apiErr := &APIError{Status: resp.StatusCode}
		_ = json.Unmarshal(raw, apiErr)
		if apiErr.Code == "" {
			apiErr.Code = string(raw)
		}
		return apiErr
	}
	return nil
}

// Plans fetches the public plans payload for the configured product.
func (c *Client) Plans(ctx context.Context) (*PlansResponse, error) {
	out := &PlansResponse{}
	if err := c.Do(ctx, "GET", "/api/v1/products/"+c.ProductSlug+"/plans", nil, out); err != nil {
		return nil, err
	}

	return out, nil
}

// StartTrial activates the optional trial plan for the configured product.
func (c *Client) StartTrial(ctx context.Context, planKey string) (*IssuedTrial, error) {
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

// RequestOTP triggers an OTP email for the supplied address.
func (c *Client) RequestOTP(ctx context.Context, payload OtpRequestPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.Do(ctx, "POST", "/api/auth/customer/otp/request", body, nil)
}

// VerifyOTP exchanges the OTP code for a Sanctum token and saves it on the Client.
func (c *Client) VerifyOTP(ctx context.Context, payload OtpVerifyPayload) (*OtpVerifyResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	out := &OtpVerifyResponse{}
	if err := c.Do(ctx, "POST", "/api/auth/customer/otp/verify", body, out); err != nil {
		return nil, err
	}

	c.SetCustomerToken(out.AccessToken)

	return out, nil
}

// Customer mirrors GET /api/me
type Customer struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	Name         *string `json:"name"`
	TrialEndsAt  *string `json:"trial_ends_at"`
	Plan         *string `json:"plan"`
}

// LicenseCheckPayload mirrors POST /api/licenses/check
type LicenseCheckPayload struct {
	Product string `json:"product"`
	Feature string `json:"feature"`
}

// LicenseCheckResponse describes a feature gate result.
type LicenseCheckResponse struct {
	Allowed bool    `json:"allowed"`
	Product string  `json:"product"`
	Plan    *string `json:"plan"`
	Feature string  `json:"feature"`
	Source  *string `json:"source"`
}

// LicenseActivatePayload mirrors POST /api/licenses/activate
type LicenseActivatePayload struct {
	Product     string  `json:"product"`
	DeviceType  string  `json:"device_type"`
	Platform    *string `json:"platform,omitempty"`
	DeviceName  *string `json:"device_name,omitempty"`
	AppVersion  *string `json:"app_version,omitempty"`
	Fingerprint string  `json:"fingerprint"`
}

// LicenseRefreshPayload mirrors POST /api/licenses/refresh
type LicenseRefreshPayload struct {
	Product     string `json:"product"`
	Fingerprint string `json:"fingerprint"`
}

// SignedLicense carries the signed envelope returned by activate/refresh.
type SignedLicense struct {
	KeyID      string `json:"key_id"`
	Algorithm  string `json:"algorithm"`
	Payload    string `json:"payload"`
	Signature  string `json:"signature"`
	ValidUntil string `json:"valid_until"`
}

// ActivatedDevice describes the device slot returned with a license envelope.
type ActivatedDevice struct {
	ID         string `json:"id"`
	DeviceType string `json:"type"`
	SlotsUsed  int    `json:"slots_used"`
	SlotsLimit *int   `json:"slots_limit"`
}

// LicenseActivateResponse mirrors the activate/refresh response.
type LicenseActivateResponse struct {
	Allowed  bool             `json:"allowed"`
	Product  string           `json:"product"`
	Plan     string           `json:"plan"`
	Features map[string]bool  `json:"features"`
	Device   ActivatedDevice  `json:"device"`
	License  SignedLicense    `json:"license"`
}

// EntitlementCustomer is the embedded customer descriptor on entitlements.
type EntitlementCustomer struct {
	ID    string  `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
}

// EntitlementsResponse mirrors GET /api/me/entitlements
type EntitlementsResponse struct {
	Customer     EntitlementCustomer `json:"customer"`
	Entitlements json.RawMessage     `json:"entitlements"`
	Devices      json.RawMessage     `json:"devices"`
}

// PortalLink wraps the Stripe customer portal short-lived URL.
type PortalLink struct {
	URL string `json:"url"`
}

// LicensePublicKey describes a registered Ed25519 verification key.
type LicensePublicKey struct {
	KeyID            string `json:"key_id"`
	Algorithm        string `json:"algorithm"`
	PublicKeyBase64  string `json:"public_key_base64"`
}

// LicensePublicKeysResponse mirrors GET /api/v1/license-keys/public
type LicensePublicKeysResponse struct {
	Keys         []LicensePublicKey `json:"keys"`
	ActiveKeyID  *string            `json:"active_key_id"`
}

// UsagePayload mirrors POST /api/me/usage
type UsagePayload struct {
	Product  string `json:"product"`
	Feature  string `json:"feature"`
	Date     string `json:"date"`
	DeviceFP string `json:"device_fp"`
	Action   string `json:"action"` // "check" | "increment"
}

// UsageResponse carries the current count and the plan limit for the feature.
type UsageResponse struct {
	Count   int  `json:"count"`
	Limit   *int `json:"limit"`
	Allowed bool `json:"allowed"`
}

// CustomerMe fetches the authenticated customer.
func (c *Client) CustomerMe(ctx context.Context) (*Customer, error) {
	out := &Customer{}
	if err := c.Do(ctx, "GET", "/api/me", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// LicenseCheck performs a feature gate query without mutating state.
func (c *Client) LicenseCheck(ctx context.Context, payload LicenseCheckPayload) (*LicenseCheckResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &LicenseCheckResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/check", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// LicenseActivate activates a device and returns the signed license envelope.
func (c *Client) LicenseActivate(ctx context.Context, payload LicenseActivatePayload) (*LicenseActivateResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &LicenseActivateResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/activate", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// LicenseRefresh refreshes the signed envelope for an existing device.
func (c *Client) LicenseRefresh(ctx context.Context, payload LicenseRefreshPayload) (*LicenseActivateResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &LicenseActivateResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/refresh", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Entitlements returns the active entitlement and device snapshot for the customer.
func (c *Client) Entitlements(ctx context.Context) (*EntitlementsResponse, error) {
	out := &EntitlementsResponse{}
	if err := c.Do(ctx, "GET", "/api/me/entitlements", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BillingPortal returns a short-lived Stripe customer portal URL.
func (c *Client) BillingPortal(ctx context.Context, returnURL string) (*PortalLink, error) {
	path := "/api/billing/portal?return_url=" + url.QueryEscape(returnURL)
	out := &PortalLink{}
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PublicLicenseKeys lists the registered verification public keys. Unauthenticated.
func (c *Client) PublicLicenseKeys(ctx context.Context) (*LicensePublicKeysResponse, error) {
	out := &LicensePublicKeysResponse{}
	if err := c.DoPublic(ctx, "GET", "/api/v1/license-keys/public", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// TrackUsage hits POST /api/me/usage to either check or increment the per-day
// counter for (customer, product, feature, device_fp).
func (c *Client) TrackUsage(ctx context.Context, payload UsagePayload) (*UsageResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &UsageResponse{}
	if err := c.Do(ctx, "POST", "/api/me/usage", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// TrackAnonymousUsage hits POST /api/v1/usage/anonymous (HMAC only, no
// bearer). Use when the customer is not yet authenticated; the server applies
// the limits defined on the product's anonymous_plan.
func (c *Client) TrackAnonymousUsage(ctx context.Context, payload UsagePayload) (*UsageResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &UsageResponse{}
	if err := c.Do(ctx, "POST", "/api/v1/usage/anonymous", body, out); err != nil {
		return nil, err
	}
	return out, nil
}
