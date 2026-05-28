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

type LocalConsumptionFunc func(ctx context.Context, feature string) (uint64, error)

type LicenseLoader func(ctx context.Context) (*license.SignedLicense, *license.SnapshotPayload, error)

type Options struct {
	Loader           LicenseLoader
	LocalConsumption LocalConsumptionFunc
	GraceWindow      time.Duration
	Now              func() time.Time
}

type DenyReason string

const (
	DenyNoLoader               DenyReason = "no_loader"
	DenyNoLicense              DenyReason = "no_license"
	DenyVerifyFailed           DenyReason = "verify_failed"
	DenyLicenseExpired         DenyReason = "license_expired"
	DenyLicenseInvalid         DenyReason = "license_invalid"
	DenyFeatureDisabled        DenyReason = "feature_disabled"
	DenyFeatureMissing         DenyReason = "feature_missing"
	DenyLimitReached           DenyReason = "limit_reached"
	DenyLocalConsumptionFailed DenyReason = "local_consumption_failed"
)

type FeatureAccess struct {
	Feature    string
	Allowed    bool
	HasFeature bool
	Unlimited  bool
	Remaining  uint64
	Reason     string
	ReasonKind DenyReason
	Plan       string
	State      lifecycle.State
}

func (a *FeatureAccess) deny(reason DenyReason) {
	a.ReasonKind = reason
	a.Reason = string(reason)
}

type Denied struct {
	Access FeatureAccess
}

func (e *Denied) Error() string {
	return fmt.Sprintf("billing: feature %q denied (%s)", e.Access.Feature, e.Access.Reason)
}

func IsDenied(err error) (*Denied, bool) {
	var d *Denied
	if errors.As(err, &d) {
		return d, true
	}
	return nil, false
}

type Gate struct {
	opts Options
	mu   sync.Mutex
}

func New(opts Options) *Gate {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.LocalConsumption == nil {
		opts.LocalConsumption = func(context.Context, string) (uint64, error) { return 0, nil }
	}
	return &Gate{opts: opts}
}

func (g *Gate) Check(ctx context.Context, feature string) (FeatureAccess, error) {
	access := FeatureAccess{Feature: feature, State: lifecycle.StateNone}

	if g.opts.Loader == nil {
		access.deny(DenyNoLoader)
		return access, nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	_, payload, err := g.opts.Loader(ctx)
	if err != nil {
		access.deny(DenyVerifyFailed)
		return access, err
	}
	if payload == nil {
		access.deny(DenyNoLicense)
		return access, nil
	}

	access.Plan = payload.PlanKey
	now := g.opts.Now().UTC()
	access.State = lifecycle.ComputeState(payload, g.opts.GraceWindow, now)

	switch access.State {
	case lifecycle.StateExpired:
		access.deny(DenyLicenseExpired)
		return access, nil
	case lifecycle.StateInvalid:
		access.deny(DenyLicenseInvalid)
		return access, nil
	}

	if enabled, ok := payload.Features[feature]; ok {
		access.HasFeature = enabled
		if !enabled {
			access.deny(DenyFeatureDisabled)
			return access, nil
		}
	}

	consumed, err := g.opts.LocalConsumption(ctx, feature)
	if err != nil {
		access.deny(DenyLocalConsumptionFailed)
		return access, err
	}

	remaining, unlimited, known := license.ComputeRemaining(*payload, feature, consumed)
	if !known {
		if access.HasFeature {
			access.Allowed = true
			access.Unlimited = true
			return access, nil
		}
		access.deny(DenyFeatureMissing)
		return access, nil
	}

	access.Unlimited = unlimited
	access.Remaining = remaining
	if unlimited || remaining > 0 {
		access.Allowed = true
		access.HasFeature = true
		return access, nil
	}

	access.deny(DenyLimitReached)
	return access, nil
}

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
