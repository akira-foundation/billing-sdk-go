package desktop

import (
	"context"
	"errors"

	billing "github.com/akira-io/billing-sdk-go"
)

type AuthSnapshot struct {
	Authenticated bool              `json:"authenticated"`
	Licensed      bool              `json:"licensed"`
	Customer      *billing.Customer `json:"customer,omitempty"`
	Features      []string          `json:"features"`
}

func GuestSnapshot() AuthSnapshot {
	return AuthSnapshot{Features: []string{}}
}

func (s AuthSnapshot) HasFeature(key string) bool {
	for _, f := range s.Features {
		if f == key {
			return true
		}
	}
	return false
}

type RefreshOptions struct {
	Product          string
	FallbackFeature  string
}

func RefreshAuth(ctx context.Context, client *billing.Client, opts RefreshOptions) (AuthSnapshot, error) {
	if client.CustomerToken == "" {
		return GuestSnapshot(), nil
	}

	customer, err := client.CustomerMe(ctx)
	if err != nil {
		var apiErr *billing.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 401 {
			return GuestSnapshot(), nil
		}
		return GuestSnapshot(), err
	}

	features := []string{}
	if resp, err := client.CustomerFeatures(ctx, opts.Product); err == nil {
		features = resp.Features
	}

	licensed := len(features) > 0
	if !licensed {
		if resp, err := client.LicenseCheck(ctx, billing.LicenseCheckPayload{
			Product: opts.Product,
			Feature: opts.FallbackFeature,
		}); err == nil {
			licensed = resp.Allowed
		}
	}

	return AuthSnapshot{
		Authenticated: true,
		Licensed:      licensed,
		Customer:      customer,
		Features:      features,
	}, nil
}
