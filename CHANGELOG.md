# Changelog

All notable changes to `billing-sdk-go` are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the module adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.7] — 2026-05-15

### Added

- New `oauth.go` with `GeneratePkceChallenge`, `GenerateOauthState`,
  and `BuildOauthInitURL` helpers for the Authorization Code + PKCE
  flow brokered by billing.
- `Client.ListOauthProviders(ctx, product)` returns the enabled
  providers + scopes for a product.
- `Client.ExchangeOauthCode(ctx, payload)` redeems a one-time code for
  a customer access token and stores it on the client.
- Types: `OauthProvider`, `OauthProviderInfo`, `OauthProvidersResponse`,
  `OauthExchangePayload`, `OauthExchangeResponse`, `PkceChallenge`,
  `BuildOauthInitUrlOptions`.

[0.1.7]: https://github.com/akira-io/billing-sdk-go/releases/tag/v0.1.7

## [0.1.6] — 2026-05-15

### Added

- `LicenseSnapshotPayload.UpdatesWindowDays` and
  `LicenseSnapshotPayload.OfflineGraceDays` carried through from the
  plan-level overrides on the billing server.

### Changed

- `CanUseUpdate` now uses the maximum of `PaidUpUntil` and
  `FallbackReleaseDate`, extended by `UpdatesWindowDays`, before
  comparing against the release date. Snapshots without any of the
  three fields still allow all releases.

[0.1.6]: https://github.com/akira-io/billing-sdk-go/releases/tag/v0.1.6

## [0.1.5] — 2026-05-15

### Changed

- Version bump to align all three SDKs (JS, Rust, Go) on the same
  release number. No code or API changes.

[0.1.5]: https://github.com/akira-io/billing-sdk-go/releases/tag/v0.1.5

## [0.1.3] — 2026-05-15

### Added

- New `license.go` helpers for `offline_snapshot` products:
  `DecodeLicense`, `VerifyLicense` (Ed25519 via stdlib
  `crypto/ed25519`), `ComputeRemaining`, `IsExpired`, `IsInGrace`,
  `CanUseUpdate`, `PeriodResetAt`.
- `Client.LicenseSyncUsage` POST `/api/licenses/sync-usage` to
  apply local usage deltas and receive a re-signed snapshot.
- `UsagePayload.Count int` (`json:"count,omitempty"`) for
  variable-count realtime tracking (e.g. AI token usage).
- Types: `LicensingMode`, `UsagePeriod`, `UsageFeatureState`,
  `LicenseSnapshotPayload`, `LicenseSyncUsagePayload`,
  `LicenseSyncUsageResponse`.

## [0.1.2] — 2026-05-15

### Added

- `UsagePayload` carries optional `Platform`, `DeviceType`, and
  `AppVersion` so the server can record device metadata alongside
  the usage counter. Authenticated and anonymous endpoints both
  accept the new fields.

[0.1.2]: https://github.com/akira-io/billing-sdk-go/releases/tag/v0.1.2

## [0.1.1] — 2026-05-15

### Added

- `TrackAnonymousUsage(ctx, payload)` — `POST /api/v1/usage/anonymous`.
  HMAC-only endpoint (no bearer) for metering devices that have not yet
  authenticated. The server applies the limits defined on the product's
  `anonymous_plan`.

[0.1.1]: https://github.com/akira-io/billing-sdk-go/releases/tag/v0.1.1

## [0.1.0] — 2026-05-15

First public release. Go client for the Akira Billing API. Mirrors the Rust
and JS SDKs and shares the same HMAC wire protocol.

### Client surface

- OTP login: `RequestOTP`, `VerifyOTP` (auto-stores the bearer).
- Customer profile: `CustomerMe`.
- License lifecycle: `LicenseCheck`, `LicenseActivate`, `LicenseRefresh`.
  Activation and refresh return `SignedLicense` (key_id, algorithm, base64
  payload, base64 signature, valid_until) so clients can verify the envelope
  offline with the matching Ed25519 public key.
- Entitlements snapshot: `Entitlements`.
- Stripe billing portal short-lived URL: `BillingPortal(returnURL)`.
- Usage tracking: `TrackUsage` with `check` / `increment` actions.
- Trials: `StartTrial(planKey)`.
- Plans listing: `Plans()`.
- Downloads: `LatestRelease(channel)`, `IssueDownload(channel, platform)`.
- Unsigned key set fetch: `PublicLicenseKeys()` for build-time embedding of
  the Ed25519 verification keys.

### Tooling

- `Client.Do(ctx, method, path, body, out)` for typed signed requests.
- `Client.DoPublic(ctx, method, path, body, out)` for unauthenticated
  endpoints (no HMAC, no bearer).
- HMAC signing helpers exported as `Canonical`, `Sign`, `NewNonce` so
  callers can sign requests for endpoints the SDK has not yet typed.
- `APIError{Status, Code}` carries the server error payload.
- Standard `net/http` client; configurable timeout and transport.
- Shared signature test vectors against the Rust SDK ensure wire-level
  parity.

[0.1.0]: https://github.com/akira-io/billing-sdk-go/releases/tag/v0.1.0
