# Lifecycle

Two pure functions that classify a license snapshot's lifecycle state. Consumed by `Gate` and useful directly in UI / banner logic.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

## LicenseState

```go
type LicenseState string

const (
    LicenseStateNone     LicenseState = "none"
    LicenseStateInvalid  LicenseState = "invalid"
    LicenseStateActive   LicenseState = "active"
    LicenseStateTrialing LicenseState = "trialing"
    LicenseStateGrace    LicenseState = "grace"
    LicenseStateExpired  LicenseState = "expired"
)
```

| Value | When |
|-------|------|
| `none` | `payload` was `nil` |
| `invalid` | `ValidUntil` empty or unparseable |
| `active` | `now <= ValidUntil`, not a trial |
| `trialing` | `now <= ValidUntil`, payload flagged as trial |
| `grace` | `ValidUntil < now <= ValidUntil + graceWindow` |
| `expired` | `now > ValidUntil + graceWindow` |

Lowercase string matches the JS/Rust SDK output.

## ComputeState

```go
func ComputeState(
    payload *LicenseSnapshotPayload,
    graceWindow time.Duration,
    now time.Time,
) LicenseState
```

`graceWindow` is a `time.Duration`. Convert from `payload.OfflineGraceDays` via `time.Duration(*p.OfflineGraceDays) * 24 * time.Hour`. `Gate` performs this conversion when `GraceWindow` is provided to `GateOptions`.

```go
state := billing.ComputeState(
    &decoded.Payload,
    time.Duration(*decoded.Payload.OfflineGraceDays) * 24 * time.Hour,
    time.Now(),
)

switch state {
case billing.LicenseStateTrialing:
    banner("trial")
case billing.LicenseStateGrace:
    banner("reconnect to keep offline mode")
case billing.LicenseStateExpired:
    banner("license expired")
}
```

### Trial detection

A payload is classified as a trial when either is true:

- `payload.Features["__trial"] == true` (server flag).
- `payload.PlanKey` ends with `":trial"` (naming convention).

Identical to the JS/Rust SDKs.

## TrialDaysLeft

```go
func TrialDaysLeft(payload *LicenseSnapshotPayload, now time.Time) int
```

Returns `ceil((ValidUntil - now) / 24h)` for trialing payloads. `0` when:

- `payload` is `nil`.
- Not a trial.
- `ValidUntil` unparseable.
- Past expiry (`now >= expiry`).

```go
left := billing.TrialDaysLeft(&decoded.Payload, time.Now())
if left > 0 {
    fmt.Printf("%d days left in trial\n", left)
}
```

Use `ComputeState(...) == LicenseStateTrialing` for branching; `TrialDaysLeft` is for the badge.

## Testing

Inject `now` directly:

```go
payload := billing.LicenseSnapshotPayload{
    ValidUntil: "2026-05-19T00:00:00Z",
    /* … */
}
grace := 24 * time.Hour

assertEqual(t, billing.ComputeState(&payload, grace, time.Date(2026, 5, 18, 23, 59, 0, 0, time.UTC)), billing.LicenseStateActive)
assertEqual(t, billing.ComputeState(&payload, grace, time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)), billing.LicenseStateGrace)
assertEqual(t, billing.ComputeState(&payload, grace, time.Date(2026, 5, 20, 1, 0, 0, 0, time.UTC)), billing.LicenseStateExpired)
```

---

Navigation: [← Usage](32-usage.md) · **Lifecycle** · [Loopback →](40-loopback.md)
