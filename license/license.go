package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type DecodedLicense struct {
	Raw     SignedLicense
	Payload SnapshotPayload
}

func Decode(signed SignedLicense) (*DecodedLicense, error) {
	payloadBytes, err := base64.StdEncoding.DecodeString(signed.Payload)
	if err != nil {
		return nil, fmt.Errorf("billing: decode payload b64: %w", err)
	}

	var payload SnapshotPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("billing: parse payload: %w", err)
	}

	return &DecodedLicense{Raw: signed, Payload: payload}, nil
}

func Verify(signed SignedLicense, publicKeyB64 string) (bool, error) {
	if signed.Algorithm != "ed25519" {
		return false, nil
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(signed.Payload)
	if err != nil {
		return false, fmt.Errorf("billing: decode payload b64: %w", err)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil {
		return false, fmt.Errorf("billing: decode signature b64: %w", err)
	}
	pkBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return false, fmt.Errorf("billing: decode public key b64: %w", err)
	}

	if len(pkBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("billing: public key must be %d bytes", ed25519.PublicKeySize)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return false, fmt.Errorf("billing: signature must be %d bytes", ed25519.SignatureSize)
	}

	return ed25519.Verify(ed25519.PublicKey(pkBytes), payloadBytes, sigBytes), nil
}

func ComputeRemaining(payload SnapshotPayload, feature string, consumedLocal uint64) (uint64, bool, bool) {
	state, exists := payload.Usage[feature]
	if !exists {
		return 0, false, false
	}

	switch state.Type {
	case "bool":
		if state.Enabled {
			return math.MaxUint64, true, true
		}
		return 0, false, true
	case "counter":
		total := state.ConsumedAtIssue + consumedLocal
		if total >= state.Allowance {
			return 0, false, true
		}
		return state.Allowance - total, false, true
	default:
		return 0, false, false
	}
}

func IsExpired(payload SnapshotPayload, now time.Time) bool {
	expiry, err := time.Parse(time.RFC3339, payload.ValidUntil)
	if err != nil {
		return true
	}
	return now.After(expiry)
}

func IsInGrace(payload SnapshotPayload, graceSeconds int64, now time.Time) bool {
	expiry, err := time.Parse(time.RFC3339, payload.ValidUntil)
	if err != nil {
		return false
	}
	cutoff := expiry.Add(time.Duration(graceSeconds) * time.Second)
	return !now.After(cutoff)
}

func CanUseUpdate(payload SnapshotPayload, releaseDate time.Time) bool {
	var paidUp, fallback time.Time
	var paidUpOk, fallbackOk bool

	if payload.PaidUpUntil != nil {
		if t, err := time.Parse(time.RFC3339, *payload.PaidUpUntil); err == nil {
			paidUp = t
			paidUpOk = true
		}
	}
	if payload.FallbackReleaseDate != nil {
		if t, err := time.Parse(time.RFC3339, *payload.FallbackReleaseDate); err == nil {
			fallback = t
			fallbackOk = true
		}
	}

	if !paidUpOk && !fallbackOk {
		return true
	}

	effective := paidUp
	if fallbackOk && (!paidUpOk || fallback.After(paidUp)) {
		effective = fallback
	}

	var windowDays int
	if payload.UpdatesWindowDays != nil {
		windowDays = int(*payload.UpdatesWindowDays)
	}
	cutoff := effective.Add(time.Duration(windowDays) * 24 * time.Hour)

	return !releaseDate.After(cutoff)
}

func PeriodResetAt(payload SnapshotPayload, feature string) time.Time {
	state, exists := payload.Usage[feature]
	if !exists || state.Type != "counter" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, state.PeriodEnd)
	if err != nil {
		return time.Time{}
	}
	return t
}
