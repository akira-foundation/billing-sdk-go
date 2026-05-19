# billing-sdk-go — Reference

Go client for the [Akira Billing API](https://github.com/akira-foundation/billing). Sister SDK of the [Rust crate](https://github.com/akira-io/billing-sdk-rust); both consume the same wire protocol and pass the same fixture vectors.

This reference documents every public surface of the module. Each topic lives in its own file; the in-repo filenames match the topic.

## Meta

| File | Topic |
|------|-------|
| [01-installation](01-installation.md) | Add the module, Go version requirements |
| [02-quickstart](02-quickstart.md) | 60-second snippet for a server + a desktop app |
| [03-architecture](03-architecture.md) | Trust model, request signing, context, package layout |

## Core client

| File | Topic |
|------|-------|
| [10-client](10-client.md) | `Client` — full method table, payload shapes |
| [11-signature](11-signature.md) | HMAC canonical form, `Canonical`, `Sign`, `NewNonce` |
| [12-errors](12-errors.md) | `APIError`, `Error` interface, branching |

## Storefront helpers

| File | Topic |
|------|-------|
| [24-oauth](24-oauth.md) | PKCE primitives, `BuildOauthInitURL`, OAuth state |

The Go SDK stays server / desktop focused. No pricing / checkout / UI helpers (those live in the [TypeScript SDK](https://github.com/akira-io/billing-sdk-js)).

## License + usage runtime

| File | Topic |
|------|-------|
| [30-license](30-license.md) | `DecodeLicense`, `VerifyLicense`, `ComputeRemaining`, grace, updates-window |
| [31-gate](31-gate.md) | `Gate` — feature checks with offline + grace + local consumption |
| [32-usage](32-usage.md) | `UsageTracker`, `MemoryBuffer`, `UsageBuffer` interface |
| [33-lifecycle](33-lifecycle.md) | `ComputeState`, `TrialDaysLeft`, `LicenseState` |

## Integrations

| File | Topic |
|------|-------|
| [40-loopback](40-loopback.md) | Desktop loopback PKCE OAuth (`loopback.go`) |
| [41-desktop](41-desktop.md) | Keychain, cipher, fingerprint, session, auth refresh (`desktop/`) |

## Reference

| File | Topic |
|------|-------|
| [60-protocol](60-protocol.md) | HTTP protocol contract (headers, paths, error envelope) |

---

Navigation: [README](../README.md) · **Index** · [Installation →](01-installation.md)
