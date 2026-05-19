# Loopback OAuth

Desktop loopback PKCE OAuth flow. Binds a transient `127.0.0.1` listener, opens the system browser, waits for the provider callback, exchanges the code via the SDK, and stores the bearer on the client instance.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

The flow is exposed as a method on `*Client` (not a free function).

## Signature

```go
func (c *Client) LoopbackLogin(
    ctx context.Context,
    opts LoopbackOptions,
    openBrowser BrowserOpener,
) (*LoopbackOutcome, error)

type LoopbackOptions struct {
    Provider string
    Product  string
    Timeout  time.Duration         // default 5 minutes when ≤ 0
}

type LoopbackOutcome struct {
    Exchange OauthExchangeResponse
}

type BrowserOpener func(url string) error
```

`Provider`, `Product`, and `openBrowser` are required — `LoopbackLogin` returns an error when any is missing.

## Sequence

1. `net.Listen("tcp", "127.0.0.1:0")` — kernel picks a port.
2. `GeneratePkceChallenge` + `GenerateOauthState`.
3. `BuildOauthInitURL` with `RedirectURI = "http://127.0.0.1:{port}/cb"`.
4. `openBrowser(authURL)` — caller decides how the URL reaches the user.
5. `context.WithTimeout(ctx, Timeout)` bounds the wait.
6. `acceptLoopbackCallback` blocks on `listener.Accept`; reads the request line; writes the success HTML; extracts `code` + `state` from the query string.
7. Verify `state` matches the generated one.
8. `client.ExchangeOauthCode(ctx, { Code, CodeVerifier: pkce.Verifier })`.
9. `client.SetCustomerToken(exchange.AccessToken)`.

`Stop` of the listener is `defer`'d so any path closes it.

## Errors

All wrapped with `billing:` prefix:

| Cause | Message |
|-------|---------|
| Missing `Provider` | `billing: provider required` |
| Missing `Product` | `billing: product required` |
| Missing `openBrowser` | `billing: open_browser required` |
| Listener bind failure | `billing: bind callback: <io>` |
| PKCE generation failure | `billing: pkce: <crypto/rand>` |
| State generation failure | `billing: state: <crypto/rand>` |
| `openBrowser` rejection | `billing: open browser: <upstream>` |
| Callback timeout / ctx cancel | `billing: oauth callback: <ctx err>` |
| `accept` failure | `billing: accept callback: <io>` |
| Stream read failure | `billing: read request line: <io>` |
| Malformed request line | `billing: malformed request line` |
| URL parse failure | `billing: parse url: <upstream>` |
| Missing `code` query | `billing: callback missing code` |
| Missing `state` query | `billing: callback missing state` |
| State mismatch | `billing: oauth state mismatch` |
| `ExchangeOauthCode` non-2xx | `billing: exchange code: <APIError>` |

Branch via `errors.Is(err, context.DeadlineExceeded)` for timeouts, `errors.As(err, &apiErr)` for downstream `*APIError`.

## BrowserOpener

```go
type BrowserOpener func(url string) error
```

Any function with this shape works. The `desktop` sub-package provides a default:

```go
import "github.com/akira-io/billing-sdk-go/desktop"

outcome, err := client.LoopbackLogin(ctx, billing.LoopbackOptions{
    Provider: "github",
    Product:  "unified-dev",
    Timeout:  5 * time.Minute,
}, desktop.OpenBrowser)
```

`desktop.OpenBrowser` shells out via:

- macOS → `open`
- Windows → `start`
- Linux / other → `xdg-open`

For Wails / Fyne / cross-process orchestration, wrap your own opener:

```go
opener := func(url string) error {
    return wailsRuntime.BrowserOpenURL(app.ctx, url)
}
outcome, err := client.LoopbackLogin(ctx, opts, opener)
```

## Server lifecycle

The listener is **single-shot**:

- One `accept()` call, then close (via `defer listener.Close()` and `defer conn.Close()`).
- Reads the first request line, extracts the URL, sends `HTTP/1.1 200 OK` + the success HTML.
- `conn.SetReadDeadline(time.Now().Add(15 * time.Second))` bounds the read.

The success page is inline HTML — no external assets, no network round-trip after the callback.

## Security notes

- The PKCE verifier never leaves the local process.
- `state` is regenerated per call — no risk of replay across logins.
- `net.Listen("tcp", "127.0.0.1:0")` binds the loopback interface only — never exposed to the LAN.
- The server reads only the request line; the rest of the HTTP request is ignored.
- The `state` mismatch check happens **before** `ExchangeOauthCode`, so a CSRF attempt cannot reach the billing API.

## Pairing with desktop session

The `desktop.SessionStore.Persist` helper writes `outcome.Exchange.AccessToken` to the OS keychain and back onto the client. See [41-desktop](41-desktop.md):

```go
session.Persist(client, outcome.Exchange.AccessToken)
```

---

Navigation: [← Lifecycle](33-lifecycle.md) · **Loopback** · [Desktop →](41-desktop.md)
