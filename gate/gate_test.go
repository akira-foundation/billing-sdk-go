package gate

import (
	"context"
	"testing"
	"time"

	"github.com/akira-io/billing-sdk-go/license"
	"github.com/akira-io/billing-sdk-go/lifecycle"
)

func newTestPayload(validUntil time.Time) *license.SnapshotPayload {
	return &license.SnapshotPayload{
		PlanKey:    "pro_monthly",
		ValidUntil: validUntil.Format(time.RFC3339),
		Features: map[string]bool{
			"mock_server":      true,
			"requests_per_day": true,
			"locked_feature":   false,
		},
		Usage: map[string]license.UsageFeatureState{
			"mock_server":      {Type: "bool", Enabled: true},
			"locked_feature":   {Type: "bool", Enabled: false},
			"requests_per_day": {Type: "counter", Allowance: 200, ConsumedAtIssue: 50},
		},
	}
}

func TestGateChecks(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	payload := newTestPayload(now.Add(24 * time.Hour))

	gate := New(Options{
		Loader: func(context.Context) (*license.SignedLicense, *license.SnapshotPayload, error) {
			return &license.SignedLicense{}, payload, nil
		},
		LocalConsumption: func(_ context.Context, feature string) (uint64, error) {
			if feature == "requests_per_day" {
				return 25, nil
			}
			return 0, nil
		},
		GraceWindow: 7 * 24 * time.Hour,
		Now:         func() time.Time { return now },
	})

	t.Run("bool-allowed", func(t *testing.T) {
		acc, err := gate.Check(context.Background(), "mock_server")
		if err != nil || !acc.Allowed || !acc.Unlimited {
			t.Fatalf("expected allowed unlimited, got %+v err=%v", acc, err)
		}
	})

	t.Run("counter-remaining", func(t *testing.T) {
		acc, err := gate.Check(context.Background(), "requests_per_day")
		if err != nil {
			t.Fatal(err)
		}
		if !acc.Allowed || acc.Remaining != 125 {
			t.Fatalf("want allowed remaining=125 got %+v", acc)
		}
	})

	t.Run("disabled-feature", func(t *testing.T) {
		acc, err := gate.Check(context.Background(), "locked_feature")
		if err != nil {
			t.Fatal(err)
		}
		if acc.Allowed || acc.Reason != "feature_disabled" {
			t.Fatalf("want denied feature_disabled got %+v", acc)
		}
	})

	t.Run("require-denied", func(t *testing.T) {
		_, err := gate.Require(context.Background(), "locked_feature")
		d, ok := IsDenied(err)
		if !ok || d.Access.Reason != "feature_disabled" {
			t.Fatalf("want Denied got %v", err)
		}
	})
}

func TestGateExpired(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	payload := newTestPayload(now.Add(-30 * 24 * time.Hour))

	gate := New(Options{
		Loader: func(context.Context) (*license.SignedLicense, *license.SnapshotPayload, error) {
			return &license.SignedLicense{}, payload, nil
		},
		GraceWindow: 7 * 24 * time.Hour,
		Now:         func() time.Time { return now },
	})

	acc, err := gate.Check(context.Background(), "mock_server")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Allowed || acc.State != lifecycle.StateExpired {
		t.Fatalf("want expired denial got %+v", acc)
	}
}

func TestGateNoLicense(t *testing.T) {
	gate := New(Options{
		Loader: func(context.Context) (*license.SignedLicense, *license.SnapshotPayload, error) {
			return nil, nil, nil
		},
	})
	acc, err := gate.Check(context.Background(), "mock_server")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Allowed || acc.Reason != "no_license" {
		t.Fatalf("want no_license got %+v", acc)
	}
}
