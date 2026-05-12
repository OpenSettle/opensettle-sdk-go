# OpenSettle Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/OpenSettle/opensettle-sdk-go.svg)](https://pkg.go.dev/github.com/OpenSettle/opensettle-sdk-go)

Official Go SDK for the [OpenSettle](https://opensettle.io) API — stablecoin billing on Base, Ethereum, Polygon, Arbitrum, Solana, and Tron. Non-custodial: OpenSettle never holds your funds.

Hand-written, zero third-party runtime dependencies, idiomatic `context.Context` throughout, typed errors reachable with `errors.As`, signed-webhook verifier in a separate sub-package.

## Install

```sh
go get github.com/OpenSettle/opensettle-sdk-go
```

Requires Go 1.22+.

## Quick start

```go
package main

import (
    "context"
    "log"
    "os"

    opensettle "github.com/OpenSettle/opensettle-sdk-go"
)

func main() {
    client, err := opensettle.NewClient(
        os.Getenv("OPENSETTLE_KEY"),        // sk_test_... or sk_live_...
        os.Getenv("OPENSETTLE_WORKSPACE"),
        opensettle.WithTestMode(os.Getenv("ENV") != "production"),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    checkout, err := client.Checkouts.Create(ctx, opensettle.CreateCheckoutRequest{
        Mode:       opensettle.CheckoutPayment,
        CustomerID: "cu_1",
        InvoiceID:  "in_1",
        SuccessURL: "https://example.com/thanks",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("checkout %s, expires %s", checkout.ID, checkout.ExpiresAt)
}
```

## Resources

The SDK mirrors the OpenSettle REST API one-to-one:

| Field             | Endpoint                                          |
| ----------------- | ------------------------------------------------- |
| `client.Checkouts`        | `/v1/workspaces/<ws>/checkouts`           |
| `client.Customers`        | `/v1/workspaces/<ws>/customers`           |
| `client.Invoices`         | `/v1/workspaces/<ws>/invoices`            |
| `client.Payments`         | `/v1/workspaces/<ws>/payments`            |
| `client.Products`         | `/v1/workspaces/<ws>/products` (+ prices) |
| `client.Subscriptions`    | `/v1/workspaces/<ws>/subscriptions`       |
| `client.WebhookEndpoints` | `/v1/workspaces/<ws>/webhook_endpoints`   |

Every method takes `ctx context.Context` as the first argument, returns `(*ResultType, error)`, and propagates request IDs in error values.

## Typed errors

```go
checkout, err := client.Payments.Refund(ctx, paymentID, opensettle.InitiateRefundRequest{})
if err != nil {
    var rl *opensettle.RateLimitError
    if errors.As(err, &rl) {
        time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
        return retry()
    }
    var stepUp *opensettle.StepUpRequiredError
    if errors.As(err, &stepUp) {
        return promptStepUp()
    }
    return err
}
```

Every API error code in the platform's 13-code taxonomy maps to a concrete Go type:

| Error type                          | Codes                                                      |
| ----------------------------------- | ---------------------------------------------------------- |
| `*InvalidRequestError`              | `invalid_request`                                          |
| `*InvalidStateTransitionError`      | `invalid_state_transition`                                 |
| `*AuthenticationError`              | `unauthorized`                                             |
| `*ForbiddenError`                   | `forbidden`                                                |
| `*NotFoundError`                    | `not_found`                                                |
| `*ConflictError`                    | `conflict`                                                 |
| `*RateLimitError` (`.RetryAfter`)   | `rate_limited`                                             |
| `*SettlementError`                  | `chain_reverted`, `insufficient_confirmations`, `signing_required` |
| `*StepUpRequiredError`              | `aal_required`                                             |
| `*APIError`                         | `internal_error` and unknown codes                         |
| `*NetworkError`                     | transport-layer failures                                   |

`OpenSettleError` is the embedded base; all subtypes carry `Code`, `Status`, `RequestID`, and `Param`.

## Webhooks

Verification lives in a sub-package so consumers can use it without dragging the rest of the SDK into their handler:

```go
import "github.com/OpenSettle/opensettle-sdk-go/webhooks"

func handle(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    v, err := webhooks.Verify(webhooks.Opts{
        RawBody:         body,
        SignatureHeader: r.Header.Get("x-opensettle-signature"),
        Secret:          os.Getenv("OPENSETTLE_WEBHOOK_SECRET"),
    })
    if err != nil {
        http.Error(w, "invalid", http.StatusBadRequest)
        return
    }
    process(v.Body, v.Timestamp)
}
```

`webhooks.Decode[T]` is the generic version when you want a typed event:

```go
ev, _, err := webhooks.Decode[PaymentConfirmedEvent](opts)
```

Signature scheme: HMAC-SHA256 over `<unix_seconds>.<raw_body>`, hex-encoded. Constant-time comparison via `hmac.Equal`. Default tolerance window is 5 minutes; tune via `Opts.Tolerance`.

## Configuration

```go
client, err := opensettle.NewClient(apiKey, workspaceID,
    opensettle.WithBaseURL("https://api.opensettle.io"),
    opensettle.WithHTTPClient(myHTTPClient),
    opensettle.WithMaxRetries(3),       // default 3; set 0 to disable
    opensettle.WithTimeout(30 * time.Second),
    opensettle.WithTestMode(true),      // refuses sk_live_ keys; CI circuit breaker
    opensettle.WithUserAgent("my-app/1.0"),
)
```

Retries cover 5xx + 429 + transport errors with exponential backoff capped at 4 s. The SDK respects `Retry-After` (both delta-seconds and HTTP-date forms). Idempotency keys are automatically attached to money-adjacent writes and shared across retry attempts.

## Versioning

Strict semver. The major version tracks the HTTP API major version (`v1`). Minor versions add features without breakage; patch versions are bug fixes only. See [CHANGELOG.md](./CHANGELOG.md).

## License

[MIT](./LICENSE)
