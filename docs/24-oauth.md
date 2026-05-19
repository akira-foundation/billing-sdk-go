# OAuth

PKCE primitives + init-URL builder. No secret, no signature. Pairs with `Client.ExchangeOauthCode` on the trusted side.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

## PkceChallenge

```go
type PkceChallenge struct {
    Verifier  string
    Challenge string
    Method    string   // always "S256"
}
```

`Verifier` is 48 bytes of CSPRNG → `base64.RawURLEncoding`. `Challenge` is `base64.RawURLEncoding(sha256(verifier))`. Persist `Verifier` locally; transmit `Challenge` via the init URL.

## GeneratePkceChallenge

```go
func GeneratePkceChallenge() (PkceChallenge, error)
```

Pure CPU work. Returns an error only when `crypto/rand.Read` fails.

## GenerateOauthState

```go
func GenerateOauthState() (string, error)
```

24 bytes of CSPRNG → `base64.RawURLEncoding`. Echoes back via the provider redirect — compare before exchanging the code.

## BuildOauthInitURL

```go
type BuildOauthInitUrlOptions struct {
    BaseURL             string
    Provider            string
    Product             string
    RedirectURI         string
    CodeChallenge       string
    CodeChallengeMethod string    // default "S256"
    State               string
}

func BuildOauthInitURL(opts BuildOauthInitUrlOptions) string
```

Returns `{BaseURL}/auth/{Provider}?product=…&redirect_uri=…&code_challenge=…&code_challenge_method=…&state=…`.

- `BaseURL` trailing slash is stripped.
- Provider is `url.PathEscape`'d.
- Query values go through `url.Values.Encode` (proper URL-encoding).
- `State` is omitted when empty — always pass one.

## Full desktop flow

```go
pkce, err := billing.GeneratePkceChallenge()
if err != nil {
    return err
}
state, err := billing.GenerateOauthState()
if err != nil {
    return err
}

store.Put("pkce_verifier", pkce.Verifier)
store.Put("oauth_state", state)

url := billing.BuildOauthInitURL(billing.BuildOauthInitUrlOptions{
    BaseURL:       "https://billing.akira.foundation",
    Provider:      "github",
    Product:       "unified-dev",
    RedirectURI:   "http://127.0.0.1:31337/cb",
    CodeChallenge: pkce.Challenge,
    State:         state,
})

openBrowser(url)
```

Callback handler:

```go
code := r.URL.Query().Get("code")
returned := r.URL.Query().Get("state")

if returned != store.Take("oauth_state") {
    http.Error(w, "oauth_state_mismatch", http.StatusBadRequest)
    return
}

verifier := store.Take("pkce_verifier")
exchange, err := client.ExchangeOauthCode(r.Context(), billing.OauthExchangePayload{
    Code:         code,
    CodeVerifier: verifier,
})
if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
// client now carries exchange.AccessToken
```

## Loopback shortcut

`Client.LoopbackLogin` (see [40-loopback](40-loopback.md)) binds a local listener, opens the browser, awaits the callback, and exchanges the code in one call. Prefer that for desktop apps.

## PKCE method override

`GeneratePkceChallenge` always emits `"S256"`. Override `CodeChallengeMethod` only when interoperating with a third-party client that cannot SHA-256 the verifier.

---

Navigation: [← Errors](12-errors.md) · **OAuth** · [License →](30-license.md)
