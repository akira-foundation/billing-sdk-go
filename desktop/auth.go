package desktop

import (
	"context"
	"errors"

	"github.com/akira-io/billing-sdk-go/client"
	"github.com/akira-io/billing-sdk-go/customer"
	"github.com/akira-io/billing-sdk-go/license"
)

type AuthSnapshot struct {
	Authenticated bool               `json:"authenticated"`
	Licensed      bool               `json:"licensed"`
	Customer      *customer.Customer `json:"customer,omitempty"`
	Features      []string           `json:"features"`
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
	Product         string
	FallbackFeature string
}

func RefreshAuth(ctx context.Context, c *client.Client, opts RefreshOptions) (AuthSnapshot, error) {
	if c.CustomerToken == "" {
		return GuestSnapshot(), nil
	}

	me, err := customer.Me(ctx, c)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 401 {
			return GuestSnapshot(), nil
		}
		return GuestSnapshot(), err
	}

	features := []string{}
	if resp, err := customer.Features(ctx, c, opts.Product); err == nil {
		features = resp.Features
	}

	licensed := len(features) > 0
	if !licensed {
		if resp, err := license.Check(ctx, c, license.CheckPayload{
			Product: opts.Product,
			Feature: opts.FallbackFeature,
		}); err == nil {
			licensed = resp.Allowed
		}
	}

	return AuthSnapshot{
		Authenticated: true,
		Licensed:      licensed,
		Customer:      me,
		Features:      features,
	}, nil
}
