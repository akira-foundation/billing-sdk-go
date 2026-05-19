# billing-sdk-go

Go client for the [Akira Billing API](https://github.com/akira-foundation/billing). Sub-package layout mirroring the [`onyx`](https://github.com/akira-io/onyx) toolkit and the sister Rust crate.

> Full reference: [`docs/00-index.md`](docs/00-index.md) - one file per package, mirrored across the JS, Rust, and Go SDKs.

> Full reference: [`docs/00-index.md`](docs/00-index.md) - one file per module, with the same numbered structure mirrored in the JS and Rust SDKs.

## Install

```bash
go get github.com/akira-io/billing-sdk-go@latest
```

## Packages

| Path | Topic |
|------|-------|
| `client` | `Client`, `APIError`, `Do`/`DoPublic`, `New`, `SetCustomerToken` |
| `signature` | HMAC primitives + headers (`Canonical`, `Sign`, `NewNonce`) |
| `customer` | OTP login, `Me`, `Entitlements`, `Features`, `Portal` |
| `license` | `Decode`, `Verify`, `ComputeRemaining`, `Check`, `Activate`, `Refresh`, `SyncUsage`, `PublicKeys` |
| `gate` | Runtime feature gate — `Check` / `Require` with offline + grace |
| `usage` | `Tracker`, `MemoryBuffer`, `Track`, `TrackAnonymous` |
| `lifecycle` | `ComputeState`, `TrialDaysLeft`, `State` |
| `oauth` | PKCE primitives, `BuildInitURL`, `Exchange`, `ListProviders` |
| `github` | `GetAppInfo`, `Installations`, `IssueInstallationToken` |
| `downloads` | `Plans`, `StartTrial`, `LatestRelease`, `IssueDownload`, `CompleteDownload` |
| `loopback` | Desktop loopback PKCE OAuth flow |
| `platform` | `Detect`, `Platform`, `DownloadURL`, `PickDownloadURL` |
| `desktop` | OS keychain, AES-256-GCM cipher, fingerprint, session, refresh helper |

## Quick start — backend service

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/akira-io/billing-sdk-go/client"
    "github.com/akira-io/billing-sdk-go/customer"
    "github.com/akira-io/billing-sdk-go/downloads"
)

func main() {
    c := client.New(
        "https://billing.akira.foundation",
        "spectra",
        os.Getenv("AKIRA_BILLING_SECRET"),
    )
    ctx := context.Background()

    plans, err := downloads.Plans(ctx, c)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Beta active: %v · %d plans\n", plans.BetaActive, len(plans.Plans))

    if err := customer.RequestOTP(ctx, c, customer.OtpRequestPayload{
        Email:      "kid@example.com",
        DeviceFP:   "deadbeef",
        Platform:   "macos",
        AppVersion: "0.1.0",
    }); err != nil {
        panic(err)
    }

    resp, err := customer.VerifyOTP(ctx, c, customer.OtpVerifyPayload{
        Email:    "kid@example.com",
        Code:     "123456",
        DeviceFP: "deadbeef",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Signed in as %s\n", resp.Customer.Email)
    // c now carries the bearer; subsequent calls auto-sign + auth.
}
```

## Configuration

```go
c := client.New(baseURL, productSlug, productSecret)
c.HTTP.Timeout = 30 * time.Second        // override default 10s
c.SetCustomerToken("existing-bearer")     // restore from keychain
```

`client.Client` exposes `BaseURL`, `ProductSlug`, `ProductSecret`, `CustomerToken`, and `HTTP` as plain fields so consumers can rotate any of them in place.

## Error handling

Non-2xx responses come back as `*client.APIError`:

```go
import (
    "errors"
    "github.com/akira-io/billing-sdk-go/client"
    "github.com/akira-io/billing-sdk-go/downloads"
)

plans, err := downloads.Plans(ctx, c)
if err != nil {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "unknown_product":
        case "trial_already_used", "already_has_entitlement":
        case "bad_signature", "missing_signature_headers", "timestamp_skew":
        }
    }
    return err
}
```

## Licensing modes

Products are tagged server-side with a `licensing_mode`:

| Mode | When to use | Client flow |
|------|-------------|-------------|
| `offline_snapshot` | Desktop apps. Long-lived entitlement, infrequent sync. | Refresh signed snapshot, decrement local counter, sync deltas periodically. |
| `online_realtime` | Pay-per-unit (AI tokens, API calls). | Pre-check budget + post-commit actual `Count`. |

### Offline snapshot helpers

```go
import (
    "github.com/akira-io/billing-sdk-go/license"
)

resp, err := license.Refresh(ctx, c, license.RefreshPayload{
    Product:     "maintainer",
    Fingerprint: fp,
})
if err != nil { return err }

decoded, err := license.Decode(resp.License)
if err != nil { return err }

keys, _ := license.PublicKeys(ctx, c)
ok, _ := license.Verify(resp.License, keys.Keys[0].PublicKeyBase64)
if !ok { return errors.New("forged license") }

remaining, unlim, present := license.ComputeRemaining(decoded.Payload, "agent_run", localConsumed)
if !present { return errors.New("unknown feature") }
if !unlim && remaining == 0 { return errors.New("limit reached") }

next, err := license.SyncUsage(ctx, c, license.SyncUsagePayload{
    Product:     "maintainer",
    Fingerprint: fp,
    Serial:      decoded.Payload.Serial,
    Deltas:      map[string]uint64{"agent_run": 3},
})
```

### Online realtime

```go
import "github.com/akira-io/billing-sdk-go/usage"

pre, err := usage.Track(ctx, c, usage.Payload{
    Product:  "aisite",
    Feature:  "llm_tokens",
    Date:     "2026-05-15",
    DeviceFP: fp,
    Action:   "check",
    Count:    4000,
})
if err != nil { return err }
if !pre.Allowed { return errors.New("budget exhausted") }

// call LLM, get actuals
actual := response.Usage.TotalTokens

_, err = usage.Track(ctx, c, usage.Payload{
    Product:  "aisite",
    Feature:  "llm_tokens",
    Date:     "2026-05-15",
    DeviceFP: fp,
    Action:   "increment",
    Count:    actual,
})
```

## Loopback OAuth

```go
import (
    "github.com/akira-io/billing-sdk-go/loopback"
)

outcome, err := loopback.Login(ctx, c, loopback.Options{
    Provider: "github",
    Product:  "spectra",
    Timeout:  5 * time.Minute,
}, openBrowser)
```

## Migration from the flat `billing` package

Pre-0.4 the package was a single `billing` root. The 0.4 release split the surface into the table above. The mapping is purely structural:

| Old | New |
|-----|-----|
| `billing.NewClient` | `client.New` |
| `billing.Client` | `client.Client` |
| `billing.APIError` | `client.APIError` |
| `billing.Canonical` / `Sign` / `NewNonce` | `signature.Canonical` / `Sign` / `NewNonce` |
| `billing.RequestOTP` / `VerifyOTP` / `CustomerMe` / ... | `customer.RequestOTP` / `VerifyOTP` / `Me` / ... |
| `billing.LicenseCheck` / `LicenseActivate` / ... | `license.Check` / `Activate` / ... |
| `billing.DecodeLicense` / `VerifyLicense` / ... | `license.Decode` / `Verify` / ... |
| `billing.Gate` / `NewGate` / `IsGateDenied` | `gate.Gate` / `gate.New` / `gate.IsDenied` |
| `billing.UsageTracker` / `NewUsageTracker` | `usage.Tracker` / `usage.NewTracker` |
| `billing.ComputeState` / `TrialDaysLeft` / `LicenseState*` | `lifecycle.ComputeState` / `TrialDaysLeft` / `State*` |
| `billing.GeneratePkceChallenge` / `BuildOauthInitURL` | `oauth.GeneratePkceChallenge` / `BuildInitURL` |
| `billing.ExchangeOauthCode` / `ListOauthProviders` | `oauth.Exchange` / `ListProviders` |
| `billing.GithubAppInfo` / `MeGithubInstallations` / `GithubInstallationToken` | `github.GetAppInfo` / `Installations` / `IssueInstallationToken` |
| `billing.Plans` / `StartTrial` / `LatestRelease` / `IssueDownload` / `CompleteDownload` | `downloads.Plans` / ... |
| `*Client.LoopbackLogin` | `loopback.Login` (takes `*client.Client`) |

Endpoint methods are now free functions taking `*client.Client`:

```go
// before
me, err := c.CustomerMe(ctx)

// after
me, err := customer.Me(ctx, c)
```
