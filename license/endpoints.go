package license

import (
	"context"
	"encoding/json"

	"github.com/akira-io/billing-sdk-go/client"
)

func Check(ctx context.Context, c *client.Client, payload CheckPayload) (*CheckResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &CheckResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/check", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Activate(ctx context.Context, c *client.Client, payload ActivatePayload) (*ActivateResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &ActivateResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/activate", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Refresh(ctx context.Context, c *client.Client, payload RefreshPayload) (*ActivateResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &ActivateResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/refresh", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

func SyncUsage(ctx context.Context, c *client.Client, payload SyncUsagePayload) (*SyncUsageResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &SyncUsageResponse{}
	if err := c.Do(ctx, "POST", "/api/licenses/sync-usage", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

func PublicKeys(ctx context.Context, c *client.Client) (*PublicKeysResponse, error) {
	out := &PublicKeysResponse{}
	if err := c.DoPublic(ctx, "GET", "/api/v1/license-keys/public", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
