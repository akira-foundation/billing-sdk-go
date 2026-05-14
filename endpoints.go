package billing

import (
	"context"
	"encoding/json"
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
	StripePriceID   *string          `json:"stripe_price_id"`
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
