# Errors

Two error shapes flow through the package: transport errors (`net/http`) and billing-API errors (`*billing.APIError`).

```go
import billing "github.com/akira-io/billing-sdk-go"
```

## APIError

```go
type APIError struct {
    Status  int    `json:"-"`
    Code    string `json:"error"`
    Message string `json:"message"`
}

func (e *APIError) Error() string  // "billing api {status}: {code|message}"
```

`Code` is parsed from the response body's `"error"` field. If the server sent `"message"` instead, `Code` stays empty and `Message` is populated. `Status` is the HTTP status — never zero for an `APIError`.

When the response body fails to JSON-decode, `Code` is populated with the raw body string as a last-resort.

## Branching

```go
import (
    "errors"
    "net/http"
    billing "github.com/akira-io/billing-sdk-go"
)

err := client.LicenseActivate(ctx, payload)
var apiErr *billing.APIError
if errors.As(err, &apiErr) {
    switch {
    case apiErr.Status == http.StatusPaymentRequired && apiErr.Code == "no_active_plan":
        redirectToUpgrade()
    case apiErr.Status == http.StatusConflict && apiErr.Code == "device_limit_reached":
        showDeviceManager()
    default:
        return err
    }
}
```

Prefer `errors.As` over type assertions — `Client.Do` always returns a `*APIError` (pointer) for non-2xx, but other layers in your stack may wrap it.

## Transport errors

Network failures (DNS, TCP, TLS, timeout, context cancellation) surface as the underlying `error` returned by `*http.Client.Do`. Common types:

- `*url.Error` — wraps the raw cause, carries `URL` and `Op`.
- `*net.OpError` — TCP-level failures.
- `context.DeadlineExceeded` / `context.Canceled` — context termination.

```go
err := client.CustomerMe(ctx)
switch {
case errors.Is(err, context.DeadlineExceeded):
    // timeout
case errors.Is(err, context.Canceled):
    // caller cancelled
}
```

## Catalogue (non-exhaustive)

Stable codes the billing API emits — safe to branch on.

### Authentication

| Code | Status |
|------|--------|
| `missing_signature_headers` | 401 |
| `unknown_product` | 401 |
| `timestamp_skew` | 401 |
| `bad_signature` | 401 |
| `nonce_replay` | 401 |
| `unauthorized` | 401 |
| `forbidden` | 403 |

### Customer + licensing

| Code | Status |
|------|--------|
| `customer_not_found` | 404 |
| `no_active_plan` | 402 |
| `device_limit_reached` | 409 |
| `license_invalid` | 409 |
| `license_revoked` | 410 |
| `fingerprint_mismatch` | 409 |

### Usage

| Code | Status |
|------|--------|
| `feature_disabled` | 403 |
| `limit_reached` | 429 |
| `usage_window_closed` | 409 |

### OAuth

| Code | Status |
|------|--------|
| `oauth_provider_unsupported` | 400 |
| `oauth_state_mismatch` | 400 |
| `oauth_code_invalid` | 400 |

## Retry guidance

- Transport timeout / connect reset → safe to retry with exponential backoff. Rebuild the request — the timestamp and nonce should not be reused.
- `5xx` `*APIError` → retry; signature is still valid for the timestamp window.
- `Code == "timestamp_skew"` → resync the system clock; rebuild the request.
- `Code == "nonce_replay"` → almost always a logic bug. Investigate.
- `Code == "limit_reached"` → surface to the user; do not auto-retry.

The package does not retry on its own — wrap `client.HTTP` with a transport that handles retries, or wrap each call site with `backoff` / your own loop.

## Distinguishing classes

```go
func classify(err error) string {
    var apiErr *billing.APIError
    switch {
    case errors.As(err, &apiErr):
        switch {
        case apiErr.Status == 401:               return "auth"
        case apiErr.Status == 402:               return "billing"
        case apiErr.Status == 429:               return "rate_limit"
        case apiErr.Status >= 500:               return "server"
        default:                                 return "client"
        }
    case errors.Is(err, context.DeadlineExceeded): return "timeout"
    case errors.Is(err, context.Canceled):         return "cancel"
    default:                                       return "transport"
    }
}
```

Useful for tagging tracing spans or structured log entries.

---

Navigation: [← Signature](11-signature.md) · **Errors** · [OAuth →](24-oauth.md)
