package billing

import (
	"time"

	"github.com/akira-io/billing-sdk-go/license"
	"github.com/akira-io/billing-sdk-go/lifecycle"
)

type LicenseState = lifecycle.State

const (
	LicenseStateNone     LicenseState = lifecycle.StateNone
	LicenseStateInvalid  LicenseState = lifecycle.StateInvalid
	LicenseStateActive   LicenseState = lifecycle.StateActive
	LicenseStateTrialing LicenseState = lifecycle.StateTrialing
	LicenseStateGrace    LicenseState = lifecycle.StateGrace
	LicenseStateExpired  LicenseState = lifecycle.StateExpired
)

func ComputeState(payload *license.SnapshotPayload, graceWindow time.Duration, now time.Time) LicenseState {
	return lifecycle.ComputeState(payload, graceWindow, now)
}

func TrialDaysLeft(payload *license.SnapshotPayload, now time.Time) int {
	return lifecycle.TrialDaysLeft(payload, now)
}

func GraceDaysLeft(payload *license.SnapshotPayload, now time.Time) int {
	return lifecycle.GraceDaysLeft(payload, now)
}
