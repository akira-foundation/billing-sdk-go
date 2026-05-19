# Architecture

## Package layout

```
billing-sdk-go
├── client.go           // Client value type, NewClient, Do/DoPublic
├── endpoints.go        // typed wrappers for every API call
├── signature.go        // Canonical / Sign / NewNonce / HEADER_* constants
├── oauth.go            // PKCE primitives + BuildOauthInitURL
├── license.go          // DecodeLicense / VerifyLicense / ComputeRemaining
├── lifecycle.go        // ComputeState / TrialDaysLeft / LicenseState
├── gate.go             // Gate runtime — feature checks
├── usage.go            // UsageTracker / MemoryBuffer / UsageBuffer interface
├── loopback.go         // desktop loopback PKCE OAuth flow
└── desktop/
    ├── browser.go      // OpenBrowser shim
    ├── cipher.go       // AES-256-GCM TokenCipher
    ├── config.go       // env helpers, checkout URL builder
    ├── fingerprint.go  // device fingerprint
    ├── keyring.go      // OS keychain wrapper
    ├── keystore.go     // 32-byte AES key persistence
    ├── session.go      // SessionStore — keychain ↔ client glue
    └── auth.go         // RefreshAuth — combined sign-in refresh
```

## Trust model

The package is server / desktop only. The `productSecret` passed to `NewClient` signs every authenticated request. Treat it as the trust root:

- Public CLIs: ship a thin command that calls a backend you control; do not embed the secret in the binary.
- Desktop apps: the binary is signed and shipped to end users. The secret in the binary is recoverable via `strings` or any disassembler. Use the `desktop/` keychain + cipher for **per-customer** tokens, but accept that the **product** secret is effectively public.

For desktop apps you have two practical options:

1. **Per-product secret + accept exposure.** The billing app rate-limits per-product traffic.
2. **Per-build rotation.** Rotate `hmac_secret` on every release. Old binaries lose access after the 24-hour grace window (see [60-protocol](60-protocol.md)).

## Request signing

Every signed request carries four headers and an HMAC-SHA256 signature computed over a deterministic canonical string. See [11-signature](11-signature.md).

```
X-Akira-Product:   <productSlug>
X-Akira-Timestamp: <unix seconds>
X-Akira-Nonce:     <random 32 hex chars>
X-Akira-Signature: <hex-encoded HMAC-SHA256>
```

The Go and Rust SDKs share `tests/fixtures/signature-vectors.json`. Any change to canonical layout requires both modules to roll together.

## Contexts and cancellation

Every endpoint method takes `context.Context` as the first argument:

```go
res, err := client.LicenseCheck(ctx, payload)
```

The context flows into the underlying `http.Request.WithContext` — cancelling it aborts the in-flight call. Set timeouts via `context.WithTimeout` rather than mutating `http.Client.Timeout` (the client is shared).

## Connection pooling

`NewClient` constructs the `*http.Client` once. The default `http.Transport` keeps idle connections per host, so cloning the value is cheap and shares the pool:

```go
base := billing.NewClient(baseURL, slug, secret)

// per-request clone (avoids token rotation race)
client := *base
client.CustomerToken = req.Cookie("akira_token").Value
res, _ := client.CustomerMe(ctx)
```

The `Client` exposes `CustomerToken` as a public field so the swap is a plain assignment. `SetCustomerToken` is provided for symmetry with the Rust/JS SDKs.

## Concurrency

- The shared `*http.Client` is safe for concurrent use.
- Mutating `CustomerToken` from multiple goroutines is **not** safe — clone the `Client` per request (cheap value copy) or guard with a `sync.RWMutex`.

## Error model

Two error shapes flow through the package:

1. Network / transport — surfaced as the underlying `error` from `net/http` (`*url.Error`, etc.).
2. Billing API — surfaced as `*billing.APIError` with `Status int` and `Code string`.

See [12-errors](12-errors.md) for the branching pattern.

## Module boundaries

```
                  storefront        signed runtime
                  ─────────         ──────────────
                  oauth.go          client.go + endpoints.go
                                    license.go
                                    gate.go
                                    usage.go
                                    lifecycle.go
                                    loopback.go (uses Client)
                                    desktop/* (uses Client + keyring)
```

`oauth.go` is browser-friendly (no secret) but is also imported by `loopback.go` to drive the desktop flow.

---

Navigation: [← Quickstart](02-quickstart.md) · **Architecture** · [Client →](10-client.md)
