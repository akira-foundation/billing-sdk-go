// Package customer owns the authenticated customer endpoints.
package customer

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/akira-io/billing-sdk-go/client"
)

type Customer struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        *string `json:"name"`
	TrialEndsAt *string `json:"trial_ends_at"`
	Plan        *string `json:"plan"`
}

type OtpRequestPayload struct {
	Email      string `json:"email"`
	DeviceFP   string `json:"device_fp,omitempty"`
	Platform   string `json:"platform,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
}

type OtpVerifyPayload struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	DeviceFP string `json:"device_fp,omitempty"`
}

type OtpVerifyResponse struct {
	AccessToken string `json:"access_token"`
	Customer    struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"customer"`
}

type EntitlementCustomer struct {
	ID    string  `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
}

type EntitlementsResponse struct {
	Customer     EntitlementCustomer `json:"customer"`
	Entitlements json.RawMessage     `json:"entitlements"`
	Devices      json.RawMessage     `json:"devices"`
}

type PortalLink struct {
	URL string `json:"url"`
}

type FeaturesResponse struct {
	Product  string   `json:"product"`
	Features []string `json:"features"`
}

func RequestOTP(ctx context.Context, c *client.Client, payload OtpRequestPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.Do(ctx, "POST", "/api/auth/customer/otp/request", body, nil)
}

// VerifyOTP stores the bearer on c.
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

func Me(ctx context.Context, c *client.Client) (*Customer, error) {
	out := &Customer{}
	if err := c.Do(ctx, "GET", "/api/me", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Entitlements(ctx context.Context, c *client.Client) (*EntitlementsResponse, error) {
	out := &EntitlementsResponse{}
	if err := c.Do(ctx, "GET", "/api/me/entitlements", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Features(ctx context.Context, c *client.Client, product string) (*FeaturesResponse, error) {
	path := "/api/me/features/" + url.PathEscape(product)
	out := &FeaturesResponse{}
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Portal(ctx context.Context, c *client.Client, returnURL string) (*PortalLink, error) {
	path := "/api/billing/portal?return_url=" + url.QueryEscape(returnURL)
	out := &PortalLink{}
	if err := c.Do(ctx, "GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
