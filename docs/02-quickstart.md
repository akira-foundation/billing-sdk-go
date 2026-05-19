# Quickstart

Two snippets — a backend service signing requests, and a desktop app driving loopback OAuth.

## Backend service

```go
package main

import (
    "context"
    "fmt"
    "os"

    billing "github.com/akira-io/billing-sdk-go"
)

func main() {
    client := billing.NewClient(
        os.Getenv("AKIRA_BILLING_URL"),
        "unified-dev",
        os.Getenv("AKIRA_BILLING_SECRET"),
    )

    ctx := context.Background()

    res, err := client.VerifyOTP(ctx, billing.OtpVerifyPayload{
        Email:    "user@example.com",
        Code:     "123456",
        DeviceFP: "",
    })
    if err != nil {
        panic(err)
    }

    // `VerifyOTP` stored the bearer on the client; subsequent calls authenticate.
    me, err := client.CustomerMe(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println("signed in as", me.Email)
    _ = res
}
```

The `Client` value is cheap to copy but **not** safe for concurrent token rotation — clone it per request if multiple goroutines mutate `CustomerToken`. The underlying `*http.Client` is shared and connection-pooled.

## Desktop loopback OAuth

```go
package main

import (
    "context"
    "log"
    "os"

    billing "github.com/akira-io/billing-sdk-go"
    "github.com/akira-io/billing-sdk-go/desktop"
)

func main() {
    client := billing.NewClient(
        os.Getenv("AKIRA_BILLING_URL"),
        "unified-dev",
        os.Getenv("AKIRA_BILLING_SECRET"),
    )

    outcome, err := client.LoopbackLogin(context.Background(), billing.LoopbackOptions{
        Provider: "github",
        Product:  "unified-dev",
        Timeout:  5 * time.Minute,
    }, desktop.OpenBrowser)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("signed in as", outcome.Exchange.Customer.Email)
}
```

`desktop.OpenBrowser` shells out to `open` / `xdg-open` / `start` depending on the OS. Plug in your own `OpenBrowser` for Wails / Fyne / native integrations.

## License runtime

```go
import (
    billing "github.com/akira-io/billing-sdk-go"
)

res, err := client.LicenseRefresh(ctx, billing.LicenseRefreshPayload{
    Product:     "unified-dev",
    Fingerprint: fp,
})
if err != nil {
    return err
}

decoded, err := billing.DecodeLicense(&res.License)
if err != nil {
    return err
}

keys, err := client.PublicLicenseKeys(ctx)
if err != nil {
    return err
}
active := keys.Keys[0]
for _, k := range keys.Keys {
    if k.KeyID == res.License.KeyID {
        active = k
        break
    }
}

ok, err := billing.VerifyLicense(&res.License, active.PublicKeyBase64)
if err != nil || !ok {
    return fmt.Errorf("forged license")
}

remaining, _ := billing.ComputeRemaining(&decoded.Payload, "agent_run", localConsumed)
if remaining.IsZero() {
    return fmt.Errorf("quota reached")
}
```

`Gate` wraps the same sequence behind one `Require(ctx, "agent_run")` call — see [31-gate](31-gate.md).

## What to read next

- [10-client](10-client.md) — full `Client` method table
- [03-architecture](03-architecture.md) — trust model + package layout
- [31-gate](31-gate.md) — runtime feature gate

---

Navigation: [← Installation](01-installation.md) · **Quickstart** · [Architecture →](03-architecture.md)
