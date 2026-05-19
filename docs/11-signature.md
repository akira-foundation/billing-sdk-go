# Signature

HMAC-SHA256 over a deterministic canonical string. Cross-SDK identical — the JS, Rust, and Go SDKs all share `tests/fixtures/signature-vectors.json`.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

`Client.Do` wraps the whole flow. Use these primitives when signing requests from outside the package (custom HTTP middleware, service mesh, message bus).

## Headers

| Constant | Value |
|----------|-------|
| `billing.HeaderProduct` | `"X-Akira-Product"` |
| `billing.HeaderTimestamp` | `"X-Akira-Timestamp"` |
| `billing.HeaderNonce` | `"X-Akira-Nonce"` |
| `billing.HeaderSignature` | `"X-Akira-Signature"` |

## Canonical

```go
func Canonical(product string, timestamp int64, nonce, method, path string, body []byte) string
```

Returns:

```
{product}\n{timestamp}\n{nonce}\n{METHOD}\n{path}\n{sha256_hex_of_body}
```

Rules:

- `method` is uppercased (`http.MethodGet`, etc. already are).
- `path` includes the query string when present.
- `body` is the request body bytes; pass `nil` or `[]byte{}` for `GET`/`DELETE`.
- `sha256_hex_of_body` is lowercase hex. Empty body → `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

## Sign

```go
func Sign(secret, canonical string) string
```

Returns lowercase-hex HMAC-SHA256. Uses `crypto/hmac` and `crypto/sha256` from the stdlib. The secret is consumed as raw bytes.

## NewNonce

```go
func NewNonce() (string, error)
```

16 bytes from `crypto/rand.Read` → 32 lowercase hex chars. Returns an error only when `crypto/rand` fails (effectively never in normal operation).

## Replay window

The server checks `X-Akira-Timestamp` against its wall clock. Outside the 300-second window the request is rejected with `timestamp_skew`. Nonces are cached for 600 seconds; a replayed `(timestamp, nonce)` pair is rejected with `nonce_replay`.

## Worked example — sign from a custom middleware

```go
package middleware

import (
    "bytes"
    "io"
    "net/http"
    "strconv"
    "time"

    billing "github.com/akira-io/billing-sdk-go"
)

type Signer struct {
    Product string
    Secret  string
}

func (s *Signer) RoundTrip(req *http.Request) (*http.Response, error) {
    var body []byte
    if req.Body != nil {
        b, err := io.ReadAll(req.Body)
        if err != nil {
            return nil, err
        }
        body = b
        req.Body = io.NopCloser(bytes.NewReader(b))
    }

    nonce, err := billing.NewNonce()
    if err != nil {
        return nil, err
    }
    ts := time.Now().Unix()

    canonical := billing.Canonical(s.Product, ts, nonce, req.Method, req.URL.RequestURI(), body)
    sig := billing.Sign(s.Secret, canonical)

    req.Header.Set(billing.HeaderProduct, s.Product)
    req.Header.Set(billing.HeaderTimestamp, strconv.FormatInt(ts, 10))
    req.Header.Set(billing.HeaderNonce, nonce)
    req.Header.Set(billing.HeaderSignature, sig)

    return http.DefaultTransport.RoundTrip(req)
}
```

## Cross-SDK fixtures

`tests/fixtures/signature-vectors.json` defines the cross-SDK contract:

```json
[
  {
    "name": "get-no-body",
    "secret": "7d1c…",
    "product": "spectra",
    "timestamp": 1714532400,
    "nonce": "0123456789abcdef0123456789abcdef",
    "method": "GET",
    "path": "/api/v1/me/products/spectra/license",
    "body": "",
    "expected_signature": "999833bb…"
  },
  …
]
```

The Go signature test (`signature_test.go`) loads this fixture and asserts byte-equal hex output against `Sign(secret, Canonical(...))`. The Rust crate runs the same assertion against the same JSON.

---

Navigation: [← Client](10-client.md) · **Signature** · [Errors →](12-errors.md)
