package billing

import (
	"testing"
	"time"
)

func TestComputeState(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	grace := 7 * 24 * time.Hour

	cases := []struct {
		name    string
		payload *LicenseSnapshotPayload
		want    LicenseState
	}{
		{"nil", nil, LicenseStateNone},
		{"empty-valid", &LicenseSnapshotPayload{}, LicenseStateInvalid},
		{
			"active",
			&LicenseSnapshotPayload{ValidUntil: now.Add(48 * time.Hour).Format(time.RFC3339), PlanKey: "pro_monthly"},
			LicenseStateActive,
		},
		{
			"trial-by-plan",
			&LicenseSnapshotPayload{ValidUntil: now.Add(48 * time.Hour).Format(time.RFC3339), PlanKey: "pro:trial"},
			LicenseStateTrialing,
		},
		{
			"trial-by-feature",
			&LicenseSnapshotPayload{
				ValidUntil: now.Add(48 * time.Hour).Format(time.RFC3339),
				PlanKey:    "pro_monthly",
				Features:   map[string]bool{"__trial": true},
			},
			LicenseStateTrialing,
		},
		{
			"grace",
			&LicenseSnapshotPayload{ValidUntil: now.Add(-24 * time.Hour).Format(time.RFC3339), PlanKey: "pro"},
			LicenseStateGrace,
		},
		{
			"expired",
			&LicenseSnapshotPayload{ValidUntil: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339), PlanKey: "pro"},
			LicenseStateExpired,
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
	p := &LicenseSnapshotPayload{
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
