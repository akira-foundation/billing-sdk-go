# Client

HMAC-signed client over `net/http`. Mirrors the JS SDK's `BillingClient` and the Rust crate's `Client`. Hosts both `Do` (signed) and `DoPublic` (unsigned) plus typed wrappers in `endpoints.go`.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

> Server / desktop only. The `ProductSecret` is the trust root — see [03-architecture](03-architecture.md).

## Struct

```go
type Client struct {
    BaseURL       string
    ProductSlug   string
    ProductSecret string
    CustomerToken string
    HTTP          *http.Client
}
```

All fields are exported so consumers can rotate the bearer (`CustomerToken`) or swap the `*http.Client` (custom timeouts, proxies, transport).

## Construction

```go
client := billing.NewClient(
    "https://billing.akira.foundation",
    "unified-dev",
    os.Getenv("AKIRA_BILLING_SECRET"),
)
```

`NewClient` wires a default `*http.Client` with `Timeout: 10 * time.Second`. Replace `client.HTTP` to lift / shorten the timeout, or to install a custom transport (mTLS, retry middleware, OpenTelemetry).

## Token management

```go
client.SetCustomerToken(bearer)
client.CustomerToken = ""        // logout (equivalent to clearing)
```

`SetCustomerToken` exists for parity with the Rust/JS SDKs — direct field assignment works too. Both mutate the value; clone the `Client` for per-request token isolation.

## Authentication

| Method | Endpoint |
|--------|----------|
| `RequestOTP(ctx, payload)` | `POST /api/auth/customer/otp/request` |
| `VerifyOTP(ctx, payload)` | `POST /api/auth/customer/otp/verify` *(stores bearer)* |
| `ExchangeOauthCode(ctx, payload)` | `POST /api/auth/oauth/exchange` *(stores bearer)* |
| `ListOauthProviders(ctx, product)` | `GET /api/v1/products/{product}/auth/providers` |

`VerifyOTP` and `ExchangeOauthCode` call `SetCustomerToken` on success.

## Customer profile

| Method | Endpoint |
|--------|----------|
| `CustomerMe(ctx)` | `GET /api/me` |
| `Entitlements(ctx)` | `GET /api/me/entitlements` |
| `CustomerFeatures(ctx, product)` | `GET /api/me/features/{product}` |
| `BillingPortal(ctx, returnURL)` | `GET /api/billing/portal?return_url=...` |

## Licensing

| Method | Endpoint |
|--------|----------|
| `LicenseCheck(ctx, payload)` | `POST /api/licenses/check` |
| `LicenseActivate(ctx, payload)` | `POST /api/licenses/activate` |
| `LicenseRefresh(ctx, payload)` | `POST /api/licenses/refresh` |
| `LicenseSyncUsage(ctx, payload)` | `POST /api/licenses/sync-usage` |
| `PublicLicenseKeys(ctx)` | `GET /api/v1/license-keys/public` *(unsigned)* |

## Usage tracking

| Method | Endpoint |
|--------|----------|
| `TrackUsage(ctx, payload)` | `POST /api/me/usage` |
| `TrackAnonymousUsage(ctx, payload)` | `POST /api/v1/usage/anonymous` |

`UsagePayload.Action` is `"check"` for pre-flight and `"increment"` after the work completes. `Count` is variable.

## Downloads + releases

| Method | Endpoint |
|--------|----------|
| `LatestRelease(ctx, channel)` | `GET /api/v1/downloads/{product}/releases/{channel}/latest` |
| `IssueDownload(ctx, channel, platform)` | `GET /api/v1/downloads/{product}/{channel}/{platform}` |
| `CompleteDownload(ctx, beaconURL)` | `POST {beaconURL}` *(unsigned)* |
| `Plans(ctx)` | `GET /api/v1/products/{product}/plans` *(unsigned)* |
| `StartTrial(ctx, planKey)` | `POST /api/v1/me/products/{product}/trial` |

`platform` is `"os-arch"` (`macos-arm64`, `linux-x86_64`, …) matching the cross-SDK `AssetPlatform` convention.

## GitHub integration

| Method | Endpoint |
|--------|----------|
| `GithubInstallationToken(ctx, payload)` | `POST /api/me/github/installation-token` |
| `MeGithubInstallations(ctx)` | `GET /api/me/github/installations` |
| `GithubAppInfo(ctx)` | `GET /api/v1/github/app` *(unsigned)* |

## Low-level `Do` / `DoPublic`

For endpoints the typed wrappers do not yet cover, drop down to `Do` (signed) or `DoPublic` (unsigned):

```go
var out MyResponse
body, _ := json.Marshal(MyRequest{...})
err := client.Do(ctx, http.MethodPost, "/api/v1/me/new-endpoint", body, &out)
```

- `body` is the raw bytes; pass `nil` or `[]byte{}` for empty bodies.
- `out` is the JSON decode target; pass `nil` to discard the response body.
- `Do` signs with `ProductSecret` + adds the bearer when `CustomerToken != ""`.
- `DoPublic` sets only `Accept` / `Content-Type`.

Both wrap non-2xx responses in `*APIError` (see [12-errors](12-errors.md)).

## Concurrency

The shared `*http.Client` is safe for concurrent use. The `Client` value itself is **not** safe to mutate concurrently — wrap with `sync.RWMutex` or clone per request:

```go
clone := *client                 // shallow copy
clone.CustomerToken = req.Token  // local mutation
res, _ := clone.CustomerMe(ctx)
```

The shallow copy still points at the same `*http.Client`, so connection pooling is preserved.

## Custom HTTP client

Replace `client.HTTP` with your own:

```go
client.HTTP = &http.Client{
    Timeout:   30 * time.Second,
    Transport: instrumentedTransport(http.DefaultTransport),
}
```

The package never reaches inside the `*http.Client` — every request flows through `client.HTTP.Do(req)`.

## Cross-SDK parity

Method names use Go casing (`PascalCase`); endpoints, payload fields, and signature semantics are identical to the JS `BillingClient` and the Rust `Client`.

---

Navigation: [← Architecture](03-architecture.md) · **Client** · [Signature →](11-signature.md)
