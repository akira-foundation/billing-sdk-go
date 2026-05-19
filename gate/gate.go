// Package gate is the runtime feature gate. It composes the license decoder,
// lifecycle state, and a caller-supplied local consumption hook into a single
// Check / Require API.
package gate

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/akira-io/billing-sdk-go/license"
	"github.com/akira-io/billing-sdk-go/lifecycle"
)

// LocalConsumptionFunc returns the locally-buffered consumption for a counter
// feature that has not yet been flushed to the server.
type LocalConsumptionFunc func(ctx context.Context, feature string) (uint64, error)

// LicenseLoader fetches and verifies the cached license. Implementations should
// return (nil, nil, nil) when no license is present.
type LicenseLoader func(ctx context.Context) (*license.SignedLicense, *license.SnapshotPayload, error)

// Options configures a Gate.
type Options struct {
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
	State      lifecycle.State
}

// Denied is returned by Require when access is not granted.
type Denied struct {
	Access FeatureAccess
}

func (e *Denied) Error() string {
	return fmt.Sprintf("billing: feature %q denied (%s)", e.Access.Feature, e.Access.Reason)
}

// IsDenied reports whether err is a *Denied and exposes its access info.
func IsDenied(err error) (*Denied, bool) {
	var d *Denied
	if errors.As(err, &d) {
		return d, true
	}
	return nil, false
}

// Gate combines verify + state + ComputeRemaining in one call.
type Gate struct {
	opts Options
	mu   sync.Mutex
}

// New returns a Gate with sensible defaults for nil callbacks.
func New(opts Options) *Gate {
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
	access := FeatureAccess{Feature: feature, State: lifecycle.StateNone}

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
	access.State = lifecycle.ComputeState(payload, g.opts.GraceWindow, now)

	switch access.State {
	case lifecycle.StateExpired, lifecycle.StateInvalid:
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

	remaining, unlimited, known := license.ComputeRemaining(*payload, feature, consumed)
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

// Require denies access with a typed *Denied error.
func (g *Gate) Require(ctx context.Context, feature string) (FeatureAccess, error) {
	access, err := g.Check(ctx, feature)
	if err != nil {
		return access, err
	}
	if !access.Allowed {
		return access, &Denied{Access: access}
	}
	return access, nil
}
