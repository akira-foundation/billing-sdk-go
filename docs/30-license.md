# License

Offline-snapshot helpers. Decode the base64-wrapped JSON payload, verify the Ed25519 signature, compute remaining quota, and reason about expiry / grace / updates-window.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

## SignedLicense (wire shape)

```go
type SignedLicense struct {
    KeyID       string `json:"key_id"`
    Algorithm   string `json:"algorithm"`       // currently "ed25519"
    Payload     string `json:"payload"`         // base64(LicenseSnapshotPayload JSON)
    Signature   string `json:"signature"`       // base64(Ed25519 signature over payload bytes)
    ValidUntil  string `json:"valid_until"`     // ISO-8601, mirror of payload.valid_until
}
```

Store verbatim. Never re-encode `Payload` — the signature covers the exact bytes the server signed.

## DecodeLicense

```go
func DecodeLicense(signed SignedLicense) (*DecodedLicense, error)

type DecodedLicense struct {
    Raw     SignedLicense
    Payload LicenseSnapshotPayload
}
```

Base64-decodes `Payload` and JSON-unmarshals it. Wraps errors with `billing: decode payload b64:` / `billing: parse payload:`.

## VerifyLicense

```go
func VerifyLicense(signed SignedLicense, publicKeyB64 string) (bool, error)
```

- Returns `(false, nil)` when `Algorithm != "ed25519"`.
- Base64-decodes payload + signature + public key.
- Validates lengths (`ed25519.PublicKeySize`, `ed25519.SignatureSize`).
- Calls `ed25519.Verify` from `crypto/ed25519`.

Returns `(true, nil)` on success, `(false, nil)` on signature mismatch, `(false, err)` on decode failures.

```go
keys, _ := client.PublicLicenseKeys(ctx)
active := keys.Keys[0]
for _, k := range keys.Keys {
    if k.KeyID == res.License.KeyID {
        active = k
        break
    }
}

ok, err := billing.VerifyLicense(res.License, active.PublicKeyBase64)
if err != nil || !ok {
    return fmt.Errorf("forged license")
}
```

## LicenseSnapshotPayload

```go
type LicenseSnapshotPayload struct {
    V                   *uint32                       `json:"v"`
    KeyID               string                        `json:"key_id"`
    CustomerID          string                        `json:"customer_id"`
    ProductKey          string                        `json:"product_key"`
    PlanKey             string                        `json:"plan_key"`
    LicensingMode       *string                       `json:"licensing_mode"`
    Features            map[string]bool               `json:"features"`
    Usage               map[string]UsageFeatureState  `json:"usage"`
    FingerprintHash     string                        `json:"fingerprint_hash"`
    Serial              uint64                        `json:"serial"`
    IssuedAt            string                        `json:"issued_at"`
    ValidUntil          string                        `json:"valid_until"`
    PaidUpUntil         *string                       `json:"paid_up_until"`
    FallbackReleaseDate *string                       `json:"fallback_release_date"`
    UpdatesWindowDays   *uint32                       `json:"updates_window_days"`
    OfflineGraceDays    *uint32                       `json:"offline_grace_days"`
}

type UsageFeatureState struct {
    Type            string  `json:"type"`         // "bool" | "counter"
    Enabled         bool    `json:"enabled"`
    Allowance       uint64  `json:"allowance"`
    Period          string  `json:"period"`       // "daily" | "weekly" | "monthly" | "yearly"
    PeriodStart     string  `json:"period_start"`
    PeriodEnd       string  `json:"period_end"`
    ConsumedAtIssue uint64  `json:"consumed_at_issue"`
}
```

`UsageFeatureState` is a flat struct (not a tagged union) — branch on `Type`.

## ComputeRemaining

```go
func ComputeRemaining(
    payload LicenseSnapshotPayload,
    feature string,
    consumedLocal uint64,
) (remaining uint64, unlimited bool, ok bool)
```

Returns:

- `(_, _, false)` — feature absent from `payload.Usage`.
- `(math.MaxUint64, true, true)` — boolean feature with `Enabled: true`.
- `(0, false, true)` — boolean feature with `Enabled: false`.
- `(allowance - consumed_at_issue - consumedLocal, false, true)` — counter feature; saturates at 0.

`consumedLocal` is the in-memory tally since the snapshot was issued.

```go
remaining, unlimited, ok := billing.ComputeRemaining(decoded.Payload, "agent_run", local)
switch {
case !ok:           denyMissingFeature()
case unlimited:     allow()
case remaining > 0: allow(); record(1)
default:            showPaywall()
}
```

## IsExpired / IsInGrace

```go
func IsExpired(payload LicenseSnapshotPayload, now time.Time) bool
func IsInGrace(payload LicenseSnapshotPayload, graceSeconds int64, now time.Time) bool
```

`graceSeconds` is **seconds**, not a `time.Duration`. Convert from `payload.OfflineGraceDays` as `int64(*payload.OfflineGraceDays) * 86_400`.

```go
if billing.IsExpired(payload, time.Now()) && !billing.IsInGrace(payload, 7*86400, time.Now()) {
    promptRefresh()
}
```

## CanUseUpdate

```go
func CanUseUpdate(payload LicenseSnapshotPayload, releaseDate time.Time) bool
```

Enforces the "perpetual fallback + updates window" model:

- If neither `PaidUpUntil` nor `FallbackReleaseDate` is set → returns `true` (no restriction).
- Effective cutoff = `max(PaidUpUntil, FallbackReleaseDate) + UpdatesWindowDays days`.
- Returns `!releaseDate.After(cutoff)`.

```go
release, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00Z")
if !billing.CanUseUpdate(decoded.Payload, release) {
    showRenewPrompt()
}
```

## PeriodResetAt

```go
func PeriodResetAt(payload LicenseSnapshotPayload, feature string) time.Time
```

Returns the counter's `PeriodEnd` parsed as RFC-3339, or `time.Time{}` when the feature is missing, not a counter, or the timestamp is unparseable. Use `t.IsZero()` to detect absence.

## Worked example — offline gate

```go
res, err := client.LicenseRefresh(ctx, billing.LicenseRefreshPayload{
    Product: "unified-dev", Fingerprint: fp,
})
if err != nil {
    return err
}

decoded, err := billing.DecodeLicense(res.License)
if err != nil {
    return err
}

keys, err := client.PublicLicenseKeys(ctx)
if err != nil {
    return err
}
active := keys.Keys[0]
for _, k := range keys.Keys {
    if k.KeyID == res.License.KeyID {
        active = k
        break
    }
}

ok, err := billing.VerifyLicense(res.License, active.PublicKeyBase64)
if err != nil || !ok {
    return fmt.Errorf("forged license")
}

graceSec := int64(0)
if decoded.Payload.OfflineGraceDays != nil {
    graceSec = int64(*decoded.Payload.OfflineGraceDays) * 86_400
}
if !billing.IsInGrace(decoded.Payload, graceSec, time.Now()) {
    return fmt.Errorf("license expired")
}

remaining, unlimited, ok := billing.ComputeRemaining(decoded.Payload, "agent_run", localConsumed)
if !ok || (!unlimited && remaining == 0) {
    return fmt.Errorf("quota reached")
}
```

Wrap the whole sequence behind `Gate.Require(ctx, "agent_run")` — see [31-gate](31-gate.md).

---

Navigation: [← OAuth](24-oauth.md) · **License** · [Gate →](31-gate.md)
