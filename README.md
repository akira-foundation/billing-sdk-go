# billing-sdk-go

Go client for the Akira Billing API.

## Install

```bash
go get github.com/akira-foundation/billing-sdk-go
```

## Usage

```go
import "github.com/akira-foundation/billing-sdk-go"

client := billing.NewClient(
    "https://billing.akira.foundation",
    "spectra",
    // injected at build time via -ldflags
    productSecret,
)

ctx := context.Background()

// public plans
plans, err := client.Plans(ctx)

// OTP login
_ = client.RequestOTP(ctx, billing.OtpRequestPayload{Email: "kid@example.com"})
resp, err := client.VerifyOTP(ctx, billing.OtpVerifyPayload{
    Email: "kid@example.com",
    Code:  "123456",
})
// resp.AccessToken is now stored on the client; subsequent calls auto-sign + auth
```

## Build-time secret injection

```bash
go build -ldflags "-X main.productSecret=$SPECTRA_BILLING_SECRET" ./cmd/spectra
```

## Wire protocol

Signature scheme documented in
[akira-foundation/billing](https://github.com/akira-foundation/billing/blob/main/docs/billing-sdk/protocol.md).

Tests against the shared vectors at `tests/fixtures/signature-vectors.json`
prove the canonical string and HMAC match the backend bit for bit.
