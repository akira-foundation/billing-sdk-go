package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// LocalConsumptionFunc returns the locally-buffered consumption for a counter
// feature that has not yet been flushed to the server.
type LocalConsumptionFunc func(ctx context.Context, feature string) (uint64, error)

// LicenseLoader fetches and verifies the cached license. Implementations should
// return (nil, nil, nil) when no license is present.
type LicenseLoader func(ctx context.Context) (*SignedLicense, *LicenseSnapshotPayload, error)

// GateOptions configures a Gate.
type GateOptions struct {
	Loader           LicenseLoader
	LocalConsumption LocalConsumptionFunc
	GraceWindow      time.Duration
	Now              func() time.Time
}

// FeatureAccess describes the outcome of a single feature check.
type FeatureAccess struct {
	Feature    string
	Allowed    bool
	HasFeature bool
	Unlimited  bool
	Remaining  uint64
	Reason     string
	Plan       string
	State      LicenseState
}

// GateDenied is returned by Require when access is not granted.
type GateDenied struct {
	Access FeatureAccess
}

func (e *GateDenied) Error() string {
	return fmt.Sprintf("billing: feature %q denied (%s)", e.Access.Feature, e.Access.Reason)
}

// IsGateDenied reports whether err is a GateDenied and exposes its access info.
func IsGateDenied(err error) (*GateDenied, bool) {
	var d *GateDenied
	if errors.As(err, &d) {
		return d, true
	}
	return nil, false
}

// Gate combines verify + state + ComputeRemaining in one call.
type Gate struct {
	opts GateOptions
	mu   sync.Mutex
}

// NewGate returns a Gate with sensible defaults for nil callbacks.
func NewGate(opts GateOptions) *Gate {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.LocalConsumption == nil {
		opts.LocalConsumption = func(context.Context, string) (uint64, error) { return 0, nil }
	}
	return &Gate{opts: opts}
}

// Check evaluates a feature without raising an error on denial.
func (g *Gate) Check(ctx context.Context, feature string) (FeatureAccess, error) {
	access := FeatureAccess{Feature: feature, State: LicenseStateNone}

	if g.opts.Loader == nil {
		access.Reason = "no_loader"
		return access, nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	_, payload, err := g.opts.Loader(ctx)
	if err != nil {
		access.Reason = "verify_failed"
		return access, err
	}
	if payload == nil {
		access.Reason = "no_license"
		return access, nil
	}

	access.Plan = payload.PlanKey
	now := g.opts.Now().UTC()
	access.State = ComputeState(payload, g.opts.GraceWindow, now)

	switch access.State {
	case LicenseStateExpired, LicenseStateInvalid:
		access.Reason = "license_" + string(access.State)
		return access, nil
	}

	if enabled, ok := payload.Features[feature]; ok {
		access.HasFeature = enabled
		if !enabled {
			access.Reason = "feature_disabled"
			return access, nil
		}
	}

	consumed, err := g.opts.LocalConsumption(ctx, feature)
	if err != nil {
		access.Reason = "local_consumption_failed"
		return access, err
	}

	remaining, unlimited, known := ComputeRemaining(*payload, feature, consumed)
	if !known {
		if access.HasFeature {
			access.Allowed = true
			access.Unlimited = true
			return access, nil
		}
		access.Reason = "feature_missing"
		return access, nil
	}

	access.Unlimited = unlimited
	access.Remaining = remaining
	if unlimited || remaining > 0 {
		access.Allowed = true
		access.HasFeature = true
		return access, nil
	}

	access.Reason = "limit_reached"
	return access, nil
}

// Require denies access with a typed GateDenied error.
func (g *Gate) Require(ctx context.Context, feature string) (FeatureAccess, error) {
	access, err := g.Check(ctx, feature)
	if err != nil {
		return access, err
	}
	if !access.Allowed {
		return access, &GateDenied{Access: access}
	}
	return access, nil
}
