# Akira Billing SDK · HMAC Request Protocol

> Version: 1.0.0  ·  Status: stable  ·  Owners: Akira Foundation

This document is the **source of truth** for how Akira desktop apps (Spectra, Debugger, etc.) sign HTTP requests sent to the Billing API. The Go SDK and the Rust crate **must** implement this protocol bit-for-bit and pass the shared test vectors at `tests/fixtures/billing-sdk/signature-vectors.json`.

## Scope

Applies to authenticated endpoints under `/api/v1/me/*` and `/api/v1/products/*/trial`. Public endpoints (`/api/v1/products/{key}/plans`) do **not** require a signature.

## Headers

Every signed request **must** carry the following headers:

| Header | Value |
|---|---|
| `X-Akira-Product` | Product slug, lowercase, e.g. `spectra` |
| `X-Akira-Timestamp` | Unix epoch seconds, integer, UTC |
| `X-Akira-Nonce` | 32 lowercase hex chars (16 random bytes) |
| `X-Akira-Signature` | 64 lowercase hex chars (HMAC-SHA256 output) |

Header names are case-insensitive on the wire (RFC 7230) but the canonical form above is what apps SHOULD send.

## Canonical string

The canonical string that is HMAC'd is built as follows, joined with `\n` (single LF, no CR):

```
{product}\n{timestamp}\n{nonce}\n{method}\n{path}\n{body_sha256}
```

Where:

- `product` — exact value of `X-Akira-Product`, no normalisation.
- `timestamp` — exact value of `X-Akira-Timestamp`, decimal string.
- `nonce` — exact value of `X-Akira-Nonce`.
- `method` — uppercase HTTP verb (`GET`, `POST`, `PATCH`, ...).
- `path` — request path **with** leading slash, **without** query string. Example: `/api/v1/me/licenses/check`.
- `body_sha256` — `hex(sha256(body_bytes))`, lowercase. For empty body, use the SHA-256 of zero bytes: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

## Signature

```
signature = hex(hmac_sha256(secret, canonical))
```

Where `secret` is the per-product `hmac_secret` issued to each app at build time. Comparison MUST be constant-time (`hash_equals`, `crypto/subtle`, `subtle::ConstantTimeEq`).

## Server validation

The middleware MUST:

1. Reject if any signing header is missing → `401 Unauthorized` with body `{"error":"missing_signature_headers"}`.
2. Reject if `X-Akira-Product` does not resolve to an active product → `401 Unauthorized` with body `{"error":"unknown_product"}`.
3. Reject if `|now - timestamp| > 300` seconds → `401 Unauthorized` with body `{"error":"timestamp_skew"}`.
4. Reject if the nonce has been seen in the last 600 seconds (Phase F) → `401 Unauthorized` with body `{"error":"nonce_replay"}`.
5. Recompute the signature using `product.hmac_secret`. If mismatch and `product.previous_hmac_secret` is set and `now - product.hmac_rotated_at < 86400`, retry with the previous secret. If still mismatch → `401 Unauthorized` with body `{"error":"bad_signature"}`.

The middleware MUST place the resolved `Product` on the request as `$request->attributes->set('akira_product', $product)` so downstream controllers can read it without re-resolving.

## Key rotation

Admins rotate a product's `hmac_secret` from the admin UI. On rotation:

1. The current secret moves to `previous_hmac_secret`.
2. A new random 32-byte hex secret is generated and stored as `hmac_secret`.
3. `hmac_rotated_at` is set to `now()`.
4. Both old and new secrets verify successfully for **24 hours**, after which only the new one is accepted.

This grace window lets builds-in-flight (CI pipelines, queued releases) finish without immediately invalidating their embedded secret.

## Sample (test vector #1)

Inputs:

- `secret` = `7d1c6e5b4a3928170f1e2d3c4b5a69788796a5b4c3d2e1f00102030405060708`
- `product` = `spectra`
- `timestamp` = `1714532400`
- `nonce` = `0123456789abcdef0123456789abcdef`
- `method` = `GET`
- `path` = `/api/v1/me/products/spectra/license`
- `body` = `` (empty)

Canonical string:

```
spectra\n1714532400\n0123456789abcdef0123456789abcdef\nGET\n/api/v1/me/products/spectra/license\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

Expected signature:

```
999833bb1a68f3a1b86cb9438f2df9e7ac2c08caa7e19817b91eaaec27cf33e8
```

> The full machine-readable fixtures with extra cases live at `tests/fixtures/billing-sdk/signature-vectors.json`. SDK suites consume that file directly.

## Error codes summary

| Status | `error` | Meaning |
|---|---|---|
| 401 | `missing_signature_headers` | One or more signing headers absent |
| 401 | `unknown_product` | `X-Akira-Product` does not match an active product |
| 401 | `timestamp_skew` | Timestamp outside ±300s window |
| 401 | `bad_signature` | HMAC mismatch (both current and previous secret) |
| 401 | `nonce_replay` | Nonce reused within 600s (Phase F) |

## Versioning

This spec is versioned via the document header. Backwards-incompatible changes (new required header, different canonical string layout, etc.) bump the major version and require coordinated SDK + backend release.
