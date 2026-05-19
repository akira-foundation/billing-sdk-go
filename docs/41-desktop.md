# Desktop helpers

Sub-package with OS keychain, AES-256-GCM token cipher, device fingerprint, session store, auth-refresh helper, and a portable `OpenBrowser` shim.

```go
import (
    billing "github.com/akira-io/billing-sdk-go"
    "github.com/akira-io/billing-sdk-go/desktop"
)
```

Pulled dependencies (transitive of the sub-package): `keyring`, `machineid`, `dbus/v5`, `wincred`. On Linux CI runners, install `libsecret-1-dev` and run a headless Secret Service if you want the keychain backend to work.

## OpenBrowser

```go
func OpenBrowser(url string) error
```

Shells out via `exec.Command(name, args...).Start()`:

- macOS → `open <url>`
- Windows → `rundll32 url.dll,FileProtocolHandler <url>`
- Linux / other → `xdg-open <url>`

Returns `fmt.Errorf("desktop: open browser: %w", err)` on `Start` failure. The launched process is detached (not waited for).

Used by `*Client.LoopbackLogin` — pass as the `BrowserOpener`:

```go
outcome, err := client.LoopbackLogin(ctx, opts, desktop.OpenBrowser)
```

## TokenCipher

AES-256-GCM string encryption keyed by a 32-byte secret.

```go
type TokenCipher struct { /* … */ }

func NewTokenCipher(key [32]byte) *TokenCipher

func (c *TokenCipher) Encrypt(plaintext string) (string, error)
func (c *TokenCipher) Decrypt(encoded string) (string, error)

func GenerateKey() ([32]byte, error)
```

Format: `base64( nonce[12] ‖ ciphertext )`. A fresh nonce is generated per call (12 bytes from `crypto/rand`).

```go
key, _ := desktop.GenerateKey()
cipher := desktop.NewTokenCipher(key)

blob, _ := cipher.Encrypt(`{"refresh":"abc","exp":123}`)
os.WriteFile("session.bin", []byte(blob), 0o600)

plain, _ := cipher.Decrypt(string(must(os.ReadFile("session.bin"))))
```

## KeyStore

```go
type KeyStore struct {
    DebugFilePath string
    /* private keyring field */
}

func NewKeyStore(keyring TokenKeyring) KeyStore

func (k KeyStore) LoadOrCreate() ([32]byte, error)
```

Loads a 32-byte AES key from the OS keychain. Generates and stores a fresh key on first run. When `DebugFilePath` is non-empty, the key is mirrored to that file so unsigned dev rebuilds can decrypt previously persisted data — macOS ad-hoc signatures lose keychain ACL on each rebuild and would otherwise effectively rotate the key.

The debug file is read **before** the keychain (highest priority). Set it conditionally on a build flag in production:

```go
ks := desktop.NewKeyStore(keyring)
if os.Getenv("DEBUG") == "1" {
    ks.DebugFilePath = filepath.Join(appData, "unified-dev", "key.b64")
}
key, err := ks.LoadOrCreate()
```

## TokenKeyring

```go
type TokenKeyring struct {
    Service string
    Account string
}

func NewTokenKeyring(service, account string) TokenKeyring

func (k TokenKeyring) Get() (string, bool, error)
func (k TokenKeyring) Set(value string) error
func (k TokenKeyring) Delete() error
```

Thin wrapper over `zalando/go-keyring`. `Service` is conventionally the bundle id (`io.akira.unified-dev`); `Account` is the customer id or `"default"`.

`Get` returns `("", false, nil)` when the entry is missing (not an error). `Delete` returns `nil` for a missing entry — idempotent.

## DeviceFingerprintFor

```go
type DeviceFingerprint struct {
    Fingerprint string
    Platform    string
    AppVersion  string
}

func DeviceFingerprintFor(appVersion string) DeviceFingerprint
```

Hashes `machineid.ID() :: runtime.GOOS :: appVersion` with SHA-256. When `machineid` fails (e.g. sandboxed environments), the seed is `"unknown"` — fingerprint stays usable but is no longer per-machine. The raw machine id never leaves the function.

```go
fp := desktop.DeviceFingerprintFor("0.9.0")
client.LicenseActivate(ctx, billing.LicenseActivatePayload{
    Product:     "unified-dev",
    DeviceType:  "desktop",
    Platform:    &fp.Platform,
    Fingerprint: fp.Fingerprint,
})
```

## SessionStore

```go
type SessionStore struct { /* … */ }

func NewSessionStore(k TokenKeyring) *SessionStore

func (s *SessionStore) Hydrate(client *billing.Client) (bool, error)
func (s *SessionStore) Persist(client *billing.Client, token string) error
func (s *SessionStore) Clear(client *billing.Client) error
func (s *SessionStore) HasToken() bool
```

| Method | Behaviour |
|--------|-----------|
| `Hydrate` | Reads the token from the keyring, pushes onto the client. Returns `false` when absent. |
| `Persist` | Writes to keyring AND `client.SetCustomerToken`. |
| `Clear` | Deletes the keyring entry AND `client.SetCustomerToken("")`. |
| `HasToken` | Best-effort check; returns `false` on any error. |

## RefreshAuth

```go
type AuthSnapshot struct {
    Authenticated bool
    Licensed      bool
    Customer      *billing.Customer
    Features      []string
}

func GuestSnapshot() AuthSnapshot
func (s AuthSnapshot) HasFeature(key string) bool

type RefreshOptions struct {
    Product         string
    FallbackFeature string
}

func RefreshAuth(ctx context.Context, client *billing.Client, opts RefreshOptions) (AuthSnapshot, error)
```

Sequence:

1. No bearer → `GuestSnapshot()`.
2. `client.CustomerMe(ctx)` — 401 → return guest; other errors propagate.
3. `client.CustomerFeatures(ctx, Product)` — failure falls through to `[]string{}`.
4. `Licensed = len(features) > 0 ? true : client.LicenseCheck(ctx, ...).Allowed`.

`RefreshAuth` does **not** clear the bearer on guest fallback — caller decides whether to `session.Clear`.

## CheckoutURL + Resolve

```go
func CheckoutURL(baseURL, product string) string

type EnvSpec struct {
    Name          string
    BakedDefault  string
    Overridable   bool
    FallbackValue string
}

func Resolve(spec EnvSpec) string
```

`CheckoutURL(baseURL, product)` returns `{baseURL}/plans?product={url-encoded product}`. Trailing slash on `baseURL` is stripped.

`Resolve(spec)` reads `os.Getenv(spec.Name)` when `Overridable` is true and the env var is set; otherwise returns `BakedDefault`, with `FallbackValue` as the final default. Typical use: ship a build with `BakedDefault` wired via `ldflags`, allow `Overridable: true` only on debug builds.

```go
baseURL := desktop.Resolve(desktop.EnvSpec{
    Name:          "AKIRA_BILLING_URL",
    BakedDefault:  bakedBillingURL,   // ldflags-injected
    Overridable:   isDebugBuild,
    FallbackValue: "https://billing.akira.foundation",
})
```

## Worked example — desktop boot

```go
keyring := desktop.NewTokenKeyring("io.akira.unified-dev", "default")
session := desktop.NewSessionStore(keyring)

client := billing.NewClient(baseURL, "unified-dev", productSecret)
session.Hydrate(client)

snapshot, err := desktop.RefreshAuth(ctx, client, desktop.RefreshOptions{
    Product:         "unified-dev",
    FallbackFeature: "general",
})
if err != nil {
    return err
}

if !snapshot.Authenticated {
    outcome, err := client.LoopbackLogin(ctx, billing.LoopbackOptions{
        Provider: "github",
        Product:  "unified-dev",
        Timeout:  5 * time.Minute,
    }, desktop.OpenBrowser)
    if err != nil {
        return err
    }
    session.Persist(client, outcome.Exchange.AccessToken)
}

fp := desktop.DeviceFingerprintFor("0.9.0")
activated, err := client.LicenseActivate(ctx, billing.LicenseActivatePayload{
    Product:     "unified-dev",
    DeviceType:  "desktop",
    Platform:    &fp.Platform,
    Fingerprint: fp.Fingerprint,
    AppVersion:  ptr("0.9.0"),
})
if err != nil {
    return err
}

raw, _ := json.Marshal(activated.License)
os.WriteFile("license.json", raw, 0o600)
```

---

Navigation: [← Loopback](40-loopback.md) · **Desktop** · [Protocol →](60-protocol.md)
