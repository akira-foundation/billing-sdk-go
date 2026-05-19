# Installation

## Add the module

```bash
go get github.com/akira-io/billing-sdk-go@latest
```

Or pin to a specific tag:

```bash
go get github.com/akira-io/billing-sdk-go@v0.3.7
```

Go 1.21+ recommended. `go.mod` declares `go 1.25.0` (the version the module was authored against) — older toolchains should still compile but are not tested.

## Imports

```go
import (
    billing "github.com/akira-io/billing-sdk-go"
    "github.com/akira-io/billing-sdk-go/desktop"
)
```

The root package exports the `Client`, signature helpers, OAuth primitives, license / gate / usage / lifecycle types, and the loopback flow.

The `desktop/` sub-package carries the OS keychain, AES-256-GCM token cipher, device fingerprint, session store, and auth refresh helper. Pull it only on desktop builds — the package's transitive deps include `keyring` and `machineid` which may require system libraries on Linux.

## Module dependencies

`go.mod` (indirect for transitive crypto):

```
github.com/danieljoos/wincred         // Windows credential helper (via keyring)
github.com/denisbrodbeck/machineid    // machine-id reader
github.com/godbus/dbus/v5             // Secret Service over D-Bus (Linux keyring)
github.com/zalando/go-keyring         // unified keyring facade
golang.org/x/sys                      // syscalls
```

No external HTTP dependency — uses the stdlib `net/http`. AES-GCM, HMAC, and SHA-256 come from `crypto/*`. Ed25519 from `crypto/ed25519`.

## Build constraints

The desktop sub-package compiles for `darwin`, `linux`, and `windows`. The keychain backend is selected at runtime by `zalando/go-keyring`; no build tags are needed.

## Verify

```bash
go build ./...
go test ./...
```

`go test ./...` runs:

- Signature fixture tests (`tests/fixtures/signature-vectors.json` — same JSON the Rust SDK consumes).
- Unit tests for `gate`, `license`, `lifecycle`, `oauth`, `signature`, `usage`.

---

Navigation: [← Index](00-index.md) · **Installation** · [Quickstart →](02-quickstart.md)
