package lifecycle

import (
	"time"

	"github.com/akira-io/billing-sdk-go/license"
)

type State string

const (
	StateNone     State = "none"
	StateInvalid  State = "invalid"
	StateActive   State = "active"
	StateTrialing State = "trialing"
	StateGrace    State = "grace"
	StateExpired  State = "expired"
)

func ComputeState(payload *license.SnapshotPayload, graceWindow time.Duration, now time.Time) State {
	if payload == nil {
		return StateNone
	}
	if payload.ValidUntil == "" {
		return StateInvalid
	}
	expiry, err := time.Parse(time.RFC3339, payload.ValidUntil)
	if err != nil {
		return StateInvalid
	}

	if now.Before(expiry) || now.Equal(expiry) {
		if isTrialPayload(payload) {
			return StateTrialing
		}
		return StateActive
	}

	cutoff := expiry.Add(graceWindow)
	if !now.After(cutoff) {
		return StateGrace
	}
	return StateExpired
}

func TrialDaysLeft(payload *license.SnapshotPayload, now time.Time) int {
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

func isTrialPayload(payload *license.SnapshotPayload) bool {
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
