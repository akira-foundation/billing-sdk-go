# Gate

Runtime feature gate. Composes [30-license](30-license.md), [33-lifecycle](33-lifecycle.md), and a caller-supplied local consumption hook into a single `Check(ctx, feature)` / `Require(ctx, feature)` API.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

## Function types

```go
type LicenseLoader func(ctx context.Context) (
    *SignedLicense, *LicenseSnapshotPayload, error,
)

type LocalConsumptionFunc func(ctx context.Context, feature string) (uint64, error)
```

`LicenseLoader` returns `(nil, nil, nil)` when no license is present. Returning an error propagates to the caller with `Reason = "verify_failed"`.

## GateOptions

```go
type GateOptions struct {
    Loader           LicenseLoader
    LocalConsumption LocalConsumptionFunc
    GraceWindow      time.Duration
    Now              func() time.Time
}
```

| Field | Default | Notes |
|-------|---------|-------|
| `Loader` | nil | When nil, every `Check` returns `Reason = "no_loader"`. |
| `LocalConsumption` | `func(_,_) (0,nil)` | Returns the in-memory tally since the snapshot was issued. |
| `GraceWindow` | `0` | Convert from `payload.OfflineGraceDays` via `time.Duration(*p.OfflineGraceDays) * 24 * time.Hour`. |
| `Now` | `time.Now` | Inject for tests. |

## Construction

```go
gate := billing.NewGate(billing.GateOptions{
    Loader: func(ctx context.Context) (*billing.SignedLicense, *billing.LicenseSnapshotPayload, error) {
        raw, err := store.Read(ctx, "license.json")
        if err != nil || raw == nil {
            return nil, nil, err
        }
        var signed billing.SignedLicense
        if err := json.Unmarshal(raw, &signed); err != nil {
            return nil, nil, err
        }
        decoded, err := billing.DecodeLicense(signed)
        if err != nil {
            return nil, nil, err
        }
        return &signed, &decoded.Payload, nil
    },
    LocalConsumption: func(ctx context.Context, feature string) (uint64, error) {
        return counter.Peek(ctx, feature)
    },
    GraceWindow: 7 * 24 * time.Hour,
})
```

## FeatureAccess

```go
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
```

Reason codes (when `Allowed == false`):

| Code | Cause |
|------|-------|
| `no_loader` | `GateOptions.Loader` was nil |
| `no_license` | Loader returned `(nil, nil, nil)` |
| `license_invalid` | `ComputeState` returned `LicenseStateInvalid` |
| `license_expired` | `ComputeState` returned `LicenseStateExpired` |
| `feature_disabled` | `payload.Features[feature] == false` |
| `feature_missing` | Feature absent from both `Features` and `Usage` |
| `limit_reached` | Counter quota exhausted |
| `verify_failed` | Loader returned an error (propagated) |
| `local_consumption_failed` | `LocalConsumption` returned an error (propagated) |

## Check

```go
func (g *Gate) Check(ctx context.Context, feature string) (FeatureAccess, error)
```

Returns `(access, nil)` for every billing decision. Returns `(access, err)` only when `Loader` or `LocalConsumption` propagate a real error.

```go
access, err := gate.Check(ctx, "agent_run")
if err != nil {
    return err
}
if !access.Allowed {
    toast(access.Reason)
    return nil
}
runAgent()
counter.Bump(ctx, "agent_run", 1)
```

## Require

```go
func (g *Gate) Require(ctx context.Context, feature string) (FeatureAccess, error)
```

Returns `(access, *GateDenied)` when the gate denies. The error wraps the same `FeatureAccess` for downstream UI.

```go
access, err := gate.Require(ctx, "agent_run")
if d, ok := billing.IsGateDenied(err); ok {
    showPaywall(d.Access.Reason, d.Access.Remaining)
    return
}
if err != nil {
    return err
}
runAgent()
```

`IsGateDenied(err)` uses `errors.As` under the hood — survives error wrapping.

## GateDenied

```go
type GateDenied struct {
    Access FeatureAccess
}

func (e *GateDenied) Error() string         // "billing: feature %q denied (%s)"
func IsGateDenied(err error) (*GateDenied, bool)
```

## Concurrency

`Gate` holds an internal `sync.Mutex` serializing concurrent `Check` calls — safe to share across goroutines via pointer (`*Gate`). Loader caching avoids per-call I/O.

## Feature presence rules

Mirrors the JS/Rust SDKs:

1. `payload.Features[feature]`:
   - `false` → `feature_disabled`.
   - `true` → mark `HasFeature: true`, continue.
   - absent → leave `HasFeature: false`, continue.
2. `ComputeRemaining(payload, feature, consumed)`:
   - `(_, _, false)` + `HasFeature: true` → unlimited (allowed).
   - `(_, _, false)` + `HasFeature: false` → `feature_missing`.
   - `(_, true, true)` → unlimited (`Allowed: true`, `Unlimited: true`).
   - `(n, false, true)` with `n > 0` → `Allowed: true`, `Remaining = n`.
   - `(0, false, true)` → `limit_reached`.

## Testing pattern

Inject `Now` and a synthetic loader — no network mocks needed.

```go
payload := billing.LicenseSnapshotPayload{ /* … */ }
signed := billing.SignedLicense{ /* … */ }

gate := billing.NewGate(billing.GateOptions{
    Loader: func(_ context.Context) (*billing.SignedLicense, *billing.LicenseSnapshotPayload, error) {
        return &signed, &payload, nil
    },
    LocalConsumption: func(_ context.Context, _ string) (uint64, error) { return 5, nil },
    GraceWindow:      7 * 24 * time.Hour,
    Now:              func() time.Time { return time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC) },
})

access, _ := gate.Check(ctx, "agent_run")
if !access.Allowed || access.Remaining != 2 {
    t.Fatalf("unexpected access: %+v", access)
}
```

---

Navigation: [← License](30-license.md) · **Gate** · [Usage →](32-usage.md)
