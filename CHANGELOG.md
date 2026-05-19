# Changelog

All notable changes to `billing-sdk-go` are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the module adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-05-19

### Added

- OS/Arch/Platform with DetectPlatform, Slug, DownloadURL + PickDownloadURL.

### Changed

- Split flat package into per-concern sub-packages.
- Strip narrative GoDoc on self-evident exports.
- Remove all leading // comments from sub-packages.

### Documentation

- Full module reference under docs/ with tens-block numbering.

## [0.3.6] - 2026-05-17

### Fixed

- Parse error fallback to message field.

## [0.3.5] - 2026-05-17

### Fixed

- Use path param for CustomerFeatures.

## [0.3.4] - 2026-05-17

### Added

- AuthSnapshot + RefreshAuth helper.

## [0.3.3] - 2026-05-16

### Added

- CustomerFeatures endpoint method.

## [0.3.2] - 2026-05-16

### Added

- TokenCipher (AES-256-GCM) and KeyStore with debug file fallback.

## [0.3.1] - 2026-05-16

### Added

- SDK CheckoutURL helper.

## [0.3.0] - 2026-05-16

### Added

- SDK desktop package and OAuth loopback helper.

## [0.2.0] - 2026-05-16

### Added

- Add Gate, UsageTracker, LicenseState helpers.

## [0.1.9] - 2026-05-16

### Added

- App metadata + user installations.

## [0.1.8] - 2026-05-15

### Added

- GithubInstallationToken + entitlement on exchange response.

## [0.1.7] - 2026-05-15

### Added

- PKCE helpers + ListOauthProviders + ExchangeOauthCode.

## [0.1.6] - 2026-05-15

### Added

- Plan-driven updates_window_days + fallback ratchet support.

## [0.1.3] - 2026-05-15

### Added

- Offline_snapshot helpers + sync-usage + variable count.

## [0.1.2] - 2026-05-15

### Added

- UsagePayload metadata fields.

## [0.1.1] - 2026-05-15

### Added

- TrackAnonymousUsage endpoint.

## [0.1.0] - 2026-05-15

### Added

- Initial Go SDK for the Akira Billing API.
- Add release manifest + issue + complete endpoints.
- Customer + license + portal + usage + pubkeys endpoints.

### Documentation

- Expand usage, error handling, configuration tables.
- Vendor protocol spec locally and fix link.
- Seed CHANGELOG with the 0.1.0 entry.

### Fixed

- Drop stripe_price_id, add is_coming_soon.


