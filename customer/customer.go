// Package customer owns the authenticated customer endpoint helpers
// (OTP login, profile, entitlements, billing portal, customer features).
package customer

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/akira-io/billing-sdk-go/client"
)

// Customer mirrors GET /api/me
type Customer struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        *string `json:"name"`
	TrialEndsAt *string `json:"trial_ends_at"`
	Plan        *string `json:"plan"`
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

// FeaturesResponse mirrors GET /api/me/features/{product}
type FeaturesResponse struct {
	Product  string   `json:"product"`
	Features []string `json:"features"`
}

// RequestOTP triggers an OTP email for the supplied address.
func RequestOTP(ctx context.Context, c *client.Client, payload OtpRequestPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.Do(ctx, "POST", "/api/auth/customer/otp/request", body, nil)
}

// VerifyOTP exchanges the OTP code for a Sanctum token and stores it on c.
func VerifyOTP(ctx context.Context, c *client.Client, payload OtpVerifyPayload) (*OtpVerifyResponse, error) {
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

// Me fetches the authenticated customer.
func Me(ctx context.Context, c *client.Client) (*Customer, error) {
	out := &Customer{}
	if err := c.Do(ctx, "GET", "/api/me", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Entitlements returns the active entitlement and device snapshot for the customer.
func Entitlements(ctx context.Context, c *client.Client) (*EntitlementsResponse, error) {
	out := &EntitlementsResponse{}
	if err := c.Do(ctx, "GET", "/api/me/entitlements", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Features returns the feature keys granted to the customer for a product.
func Features(ctx context.Context, c *client.Client, product string) (*FeaturesResponse, error) {
	path := "/api/me/features/" + url.PathEscape(product)
	out := &FeaturesResponse{}
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Portal returns a short-lived Stripe customer portal URL.
func Portal(ctx context.Context, c *client.Client, returnURL string) (*PortalLink, error) {
	path := "/api/billing/portal?return_url=" + url.QueryEscape(returnURL)
	out := &PortalLink{}
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
