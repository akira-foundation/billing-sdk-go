# billing-sdk-go

Go client for the [Akira Billing API](https://github.com/akira-foundation/billing).

Handles request signing, OTP login, license activation, trial start, and
plans listing. Pass-through for any endpoint via `Client.Do()`.

## Install

```bash
go get github.com/akira-foundation/billing-sdk-go
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    billing "github.com/akira-foundation/billing-sdk-go"
)

// Injected at build time. See "Build-time secret injection" below.
var productSecret string

func main() {
    client := billing.NewClient(
        "https://billing.akira.foundation",
        "spectra",
        productSecret,
    )

    ctx := context.Background()

    // 1. Public plans
    plans, err := client.Plans(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Beta active: %v · %d plans\n", plans.BetaActive, len(plans.Plans))

    // 2. OTP login
    if err := client.RequestOTP(ctx, billing.OtpRequestPayload{
        Email:      "kid@example.com",
        DeviceFP:   "deadbeef",
        Platform:   "macos",
        AppVersion: "0.1.0",
    }); err != nil {
        log.Fatal(err)
    }

    resp, err := client.VerifyOTP(ctx, billing.OtpVerifyPayload{
        Email:    "kid@example.com",
        Code:     "123456",
        DeviceFP: "deadbeef",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Signed in as %s\n", resp.Customer.Email)
    // resp.AccessToken is now stored on the client; subsequent calls auto-sign + auth.

    // 3. Start trial
    trial, err := client.StartTrial(ctx, "") // empty = first eligible plan
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Trial ends %s\n", trial.EndsAt)
}
```

## Configuration

```go
client := billing.NewClient(baseURL, productSlug, productSecret)
client.HTTP.Timeout = 30 * time.Second        // override default 10s
client.SetCustomerToken("existing-bearer")     // restore from keychain
```

| Field            | Type           | Notes                                                 |
| ---------------- | -------------- | ----------------------------------------------------- |
| `BaseURL`        | string         | Billing endpoint root, no trailing slash              |
| `ProductSlug`    | string         | Matches `products.key` on the backend                 |
| `ProductSecret`  | string         | Per-product HMAC secret, set at build time            |
| `CustomerToken`  | string         | Sanctum bearer, populated after `VerifyOTP`           |
| `HTTP`           | `*http.Client` | Override timeout / transport / cookies as needed      |

## Endpoints

| Method                                 | Backend route                                  | Auth        |
| -------------------------------------- | ---------------------------------------------- | ----------- |
| `Plans(ctx)`                           | `GET  /api/v1/products/{key}/plans`            | HMAC only   |
| `RequestOTP(ctx, payload)`             | `POST /api/auth/customer/otp/request`          | HMAC only   |
| `VerifyOTP(ctx, payload)`              | `POST /api/auth/customer/otp/verify`           | HMAC only   |
| `StartTrial(ctx, planKey)`             | `POST /api/v1/me/products/{key}/trial`         | HMAC + bearer |
| `Do(ctx, method, path, body, out)`     | any                                            | HMAC (+ bearer if set) |

`Do()` is the escape hatch for endpoints the SDK hasn't typed yet (license
activate/refresh, checkout, etc). Build the payload yourself and unmarshal
into a struct you provide.

## Error handling

Non-2xx responses come back as `*billing.APIError`:

```go
plans, err := client.Plans(ctx)
if err != nil {
    var apiErr *billing.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "unknown_product":
            // wrong slug
        case "trial_already_used", "already_has_entitlement":
            // expected business rule
        case "bad_signature", "missing_signature_headers", "timestamp_skew":
            // wire-level: rotate secret or sync clock
        }
    }
    return err
}
```

## Build-time secret injection

```bash
go build -ldflags "-X main.productSecret=$SPECTRA_BILLING_SECRET" ./cmd/spectra
```

The `productSecret` symbol must be a package-level `var string` in the
binary's main package. Linker overrides it at build time; release pipelines
load the secret from a vault.

For local development, drop it in a `.env` and read at startup:

```go
secret := os.Getenv("AKIRA_BILLING_SECRET")
if secret == "" {
    log.Fatal("AKIRA_BILLING_SECRET unset")
}
```

## Wire protocol

Signing scheme: HMAC-SHA256 over a canonical string that includes product
slug, unix timestamp, nonce, HTTP method, request path, and a hash of the
body.

Full spec: [docs/protocol.md](docs/protocol.md).

The fixture vectors in `tests/fixtures/signature-vectors.json` are shared
with the backend and the Rust crate. Run the test suite to confirm parity:

```bash
go test ./...
```

## Sister crate

[`akira-foundation/billing-sdk-rust`](https://github.com/akira-foundation/billing-sdk-rust)
mirrors this API for Tauri and other Rust apps. Both crates pass the same
shared vectors.

## License

MIT.
