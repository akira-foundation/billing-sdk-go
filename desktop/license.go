package desktop

import (
	"context"
	"errors"
	"fmt"

	"github.com/akira-io/billing-sdk-go/client"
	"github.com/akira-io/billing-sdk-go/license"
)

type ActivateOrRefreshOptions struct {
	Product     string
	Fingerprint string
	DeviceType  string
	Platform    *string
	DeviceName  *string
	AppVersion  *string
}

type VerifiedLicense struct {
	Signed   license.SignedLicense
	Payload  license.SnapshotPayload
	Plan     string
	Features map[string]bool
	DeviceID string
}

func ActivateOrRefresh(ctx context.Context, c *client.Client, opts ActivateOrRefreshOptions, publicKeys map[string]string) (*VerifiedLicense, error) {
	resp, err := license.Refresh(ctx, c, license.RefreshPayload{Product: opts.Product, Fingerprint: opts.Fingerprint})
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			resp, err = license.Activate(ctx, c, license.ActivatePayload{
				Product:     opts.Product,
				DeviceType:  opts.DeviceType,
				Platform:    opts.Platform,
				DeviceName:  opts.DeviceName,
				AppVersion:  opts.AppVersion,
				Fingerprint: opts.Fingerprint,
			})
		}
		if err != nil {
			return nil, err
		}
	}

	signed := resp.License
	pk, ok := publicKeys[signed.KeyID]
	if !ok {
		return nil, fmt.Errorf("unknown signing key_id: %s", signed.KeyID)
	}

	valid, err := license.Verify(signed, pk)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("license signature verification failed")
	}

	decoded, err := license.Decode(signed)
	if err != nil {
		return nil, err
	}

	return &VerifiedLicense{
		Signed:   signed,
		Payload:  decoded.Payload,
		Plan:     resp.Plan,
		Features: resp.Features,
		DeviceID: resp.Device.ID,
	}, nil
}
