package devices

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/akira-io/billing-sdk-go/client"
)

type Device struct {
	ID          string  `json:"id"`
	DeviceType  string  `json:"device_type"`
	Platform    *string `json:"platform"`
	Name        *string `json:"name"`
	AppVersion  *string `json:"app_version"`
	ActivatedAt string  `json:"activated_at"`
	LastSeenAt  *string `json:"last_seen_at"`
	RevokedAt   *string `json:"revoked_at"`
}

type Page struct {
	SlotsUsed  int      `json:"slots_used"`
	SlotsLimit int      `json:"slots_limit"`
	Devices    []Device `json:"devices"`
}

type LimitInfo struct {
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	SlotsLimit int      `json:"slots_limit"`
	SlotsUsed  int      `json:"slots_used"`
	Devices    []Device `json:"devices"`
}

func List(ctx context.Context, c *client.Client, product string) (*Page, error) {
	out := &Page{}
	if err := c.Do(ctx, "GET", "/api/me/devices/"+url.PathEscape(product), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Revoke(ctx context.Context, c *client.Client, deviceID string) error {
	return c.Do(ctx, "DELETE", "/api/me/devices/"+url.PathEscape(deviceID), nil, nil)
}

// LimitFromError reports whether err is a 409 device_limit_reached response and,
// when so, returns the slot counts plus the registered devices carried in the body.
func LimitFromError(err error) (*LimitInfo, bool) {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "device_limit_reached" {
		return nil, false
	}
	info := &LimitInfo{}
	if len(apiErr.Body) > 0 {
		_ = json.Unmarshal(apiErr.Body, info)
	}
	info.Code = apiErr.Code
	return info, true
}
