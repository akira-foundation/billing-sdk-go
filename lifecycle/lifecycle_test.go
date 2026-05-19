package lifecycle

import (
	"testing"
	"time"

	"github.com/akira-io/billing-sdk-go/license"
)

func TestComputeState(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	grace := 7 * 24 * time.Hour

	cases := []struct {
		name    string
		payload *license.SnapshotPayload
		want    State
	}{
		{"nil", nil, StateNone},
		{"empty-valid", &license.SnapshotPayload{}, StateInvalid},
		{
			"active",
			&license.SnapshotPayload{ValidUntil: now.Add(48 * time.Hour).Format(time.RFC3339), PlanKey: "pro_monthly"},
			StateActive,
		},
		{
			"trial-by-plan",
			&license.SnapshotPayload{ValidUntil: now.Add(48 * time.Hour).Format(time.RFC3339), PlanKey: "pro:trial"},
			StateTrialing,
		},
		{
			"trial-by-feature",
			&license.SnapshotPayload{
				ValidUntil: now.Add(48 * time.Hour).Format(time.RFC3339),
				PlanKey:    "pro_monthly",
				Features:   map[string]bool{"__trial": true},
			},
			StateTrialing,
		},
		{
			"grace",
			&license.SnapshotPayload{ValidUntil: now.Add(-24 * time.Hour).Format(time.RFC3339), PlanKey: "pro"},
			StateGrace,
		},
		{
			"expired",
			&license.SnapshotPayload{ValidUntil: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339), PlanKey: "pro"},
			StateExpired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeState(tc.payload, grace, now)
			if got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestTrialDaysLeft(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	p := &license.SnapshotPayload{
		ValidUntil: now.Add(72 * time.Hour).Format(time.RFC3339),
		PlanKey:    "pro:trial",
	}
	if got := TrialDaysLeft(p, now); got != 3 {
		t.Fatalf("want 3 got %d", got)
	}
	if got := TrialDaysLeft(nil, now); got != 0 {
		t.Fatalf("nil should be 0 got %d", got)
	}
}
