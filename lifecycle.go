package billing

import "time"

type LicenseState string

const (
	LicenseStateNone     LicenseState = "none"
	LicenseStateInvalid  LicenseState = "invalid"
	LicenseStateActive   LicenseState = "active"
	LicenseStateTrialing LicenseState = "trialing"
	LicenseStateGrace    LicenseState = "grace"
	LicenseStateExpired  LicenseState = "expired"
)

// ComputeState derives the lifecycle state from a snapshot at the given moment.
// graceWindow is the offline grace period applied past valid_until. The trial
// flag is consulted from the snapshot's plan_key suffix `:trial` or features
// map entry `__trial`. Callers wanting custom rules should compose their own.
func ComputeState(payload *LicenseSnapshotPayload, graceWindow time.Duration, now time.Time) LicenseState {
	if payload == nil {
		return LicenseStateNone
	}
	if payload.ValidUntil == "" {
		return LicenseStateInvalid
	}
	expiry, err := time.Parse(time.RFC3339, payload.ValidUntil)
	if err != nil {
		return LicenseStateInvalid
	}

	if now.Before(expiry) || now.Equal(expiry) {
		if isTrialPayload(payload) {
			return LicenseStateTrialing
		}
		return LicenseStateActive
	}

	cutoff := expiry.Add(graceWindow)
	if !now.After(cutoff) {
		return LicenseStateGrace
	}
	return LicenseStateExpired
}

// TrialDaysLeft returns the integer days remaining in a trialing license, or 0
// when not trialing / past expiry.
func TrialDaysLeft(payload *LicenseSnapshotPayload, now time.Time) int {
	if payload == nil || !isTrialPayload(payload) {
		return 0
	}
	expiry, err := time.Parse(time.RFC3339, payload.ValidUntil)
	if err != nil || !now.Before(expiry) {
		return 0
	}
	delta := expiry.Sub(now)
	days := int(delta.Hours() / 24)
	if delta%(24*time.Hour) > 0 {
		days++
	}
	return days
}

func isTrialPayload(payload *LicenseSnapshotPayload) bool {
	if payload == nil {
		return false
	}
	if v, ok := payload.Features["__trial"]; ok && v {
		return true
	}
	if len(payload.PlanKey) >= 6 && payload.PlanKey[len(payload.PlanKey)-6:] == ":trial" {
		return true
	}
	return false
}
