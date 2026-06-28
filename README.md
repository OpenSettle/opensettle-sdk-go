# OpenSettle Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/OpenSettle/opensettle-sdk-go.svg)](https://pkg.go.dev/github.com/OpenSettle/opensettle-sdk-go)

Official Go SDK for the [OpenSettle](https://opensettle.io) API — stablecoin billing: USDC on Base, Ethereum, Polygon, Arbitrum, and Solana; USDT on Ethereum, Polygon, Arbitrum, Solana, and Tron. Hosted checkout is EVM-only. Non-custodial: OpenSettle never holds your funds.

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

## Hosted checkout is EVM-only

Hosted checkout currently supports EVM stablecoin settlement only — Base,
Ethereum, Polygon, and Arbitrum. The `CreateCheckoutRequest.Chain` field's
type accepts `"solana"` and `"tron"` because the API + wallet-verification
layer support them (`/wallets` accepts `chain: "solana" | "tron"` and the
chain reader detects inbound SPL / TRC-20 deposits to verified wallets),
but the customer-facing hosted page does not yet render those networks.
Pass an EVM `ChainId` here, or omit `Chain` and let the buyer pick on the
hosted page — only EVM options will be shown.

## Resources

The SDK mirrors the OpenSettle REST API one-to-one:

| Field             | Endpoint                                          |
| ----------------- | ------------------------------------------------- |
| `client.Checkouts`        | `/v1/workspaces/<ws>/checkouts`           |
| `client.Customers`        | `/v1/workspaces/<ws>/customers`           |
| `client.Invoices`         | `/v1/workspaces/<ws>/invoices`            |
| `client.Payments`         | `/v1/workspaces/<ws>/payments`            |
| `client.PaymentLinks`     | `/v1/workspaces/<ws>/payment_links`       |
| `client.Products`         | `/v1/workspaces/<ws>/products` (+ prices) |
| `client.Subscriptions`    | `/v1/workspaces/<ws>/subscriptions`       |
| `client.WebhookEndpoints` | `/v1/workspaces/<ws>/webhook_endpoints`   |

Every method takes `ctx context.Context` as the first argument, returns `(*ResultType, error)`, and propagates request IDs in error values.

## Pagination

Every `List` method has a sibling `ListIter` that walks every page transparently:

```go
it := client.Customers.ListIter(ctx, &opensettle.ListCustomersQuery{
    Status: opensettle.CustomerActive,
    Limit:  100,
})
for it.Next() {
    fmt.Println(it.Item().ID)
}
if err := it.Err(); err != nil {
    return err
}
```

The iterator lazy-fetches the first page on the first `Next()` call and keeps following the `nextCursor` until `hasMore` is false. Filters from the initial query (status, customerId, etc.) are preserved across every page.

## Polling

`WaitFor` polls a `Retrieve` method until a predicate is satisfied. Useful in scripts and CI; production code should prefer webhooks.

```go
pmt, err := opensettle.WaitFor(ctx,
    client.Payments.Retrieve,
    "pay_123",
    func(p *opensettle.Payment) bool { return p.Status == opensettle.PaymentConfirmed },
    opensettle.WaitOptions{Timeout: 2 * time.Minute, Interval: 2 * time.Second},
)
if err != nil {
    var to *opensettle.WaitTimeoutError
    if errors.As(err, &to) {
        last, _ := to.Last.(*opensettle.Payment)
        log.Printf("timed out; last status: %s", last.Status)
    }
    return err
}
```

## Idempotency keys

Every money-adjacent write — `Checkouts.Create`, `Customers.Create`, `Invoices.Create/Send/Remind`, `Payments.Refund/RefundBroadcast`, `Products.Create/CreatePrice`, `Subscriptions.Create/ChangePlan`, `WebhookEndpoints.Create/RotateSecret` — auto-attaches an `Idempotency-Key` that is preserved across retry attempts. To supply your own deterministic key (e.g. your DB row id) so retries from multiple machines collapse to the same server-side operation, pass `WithIdempotencyKey`:

```go
checkout, err := client.Checkouts.Create(ctx, req,
    opensettle.WithIdempotencyKey("order:" + order.ID),
)
```

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

Every API error code in the platform's 15-code taxonomy maps to a concrete Go type:

| Error type                          | Codes                                                      |
| ----------------------------------- | ---------------------------------------------------------- |
| `*InvalidRequestError`              | `invalid_request`                                          |
| `*InvalidStateTransitionError`      | `invalid_state_transition`                                 |
| `*AuthenticationError`              | `unauthorized`                                             |
| `*ForbiddenError`                   | `forbidden`                                                |
| `*RestrictedJurisdictionError`      | `restricted_jurisdiction` (carries `.Metadata`)            |
| `*KybRequiredError`                 | `kyb_required` (HTTP 403)                                  |
| `*AttestationRequiredError`         | `attestation_required` (HTTP 412)                          |
| `*NotFoundError`                    | `not_found`                                                |
| `*ConflictError`                    | `conflict`                                                 |
| `*RateLimitError` (`.RetryAfter`)   | `rate_limited`                                             |
| `*SettlementError`                  | `chain_reverted`, `insufficient_confirmations`, `signing_required` |
| `*StepUpRequiredError`              | `aal_required`                                             |
| `*APIError`                         | `internal_error` and unknown codes                         |
| `*NetworkError`                     | transport-layer failures                                   |

`OpenSettleError` is the embedded base; all subtypes carry `Code`, `Status`, `RequestID`, `Param`, and `Metadata`.

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

`webhooks.Decode[T]` is the generic version when you want a typed event. Define
a struct matching the payload fields you care about (the SDK ships event-type
constants, not per-event payload structs, so you model just what you need):

```go
type PaymentConfirmed struct {
    PaymentID   string `json:"paymentId"`
    AmountMinor int64  `json:"amountMinor"`
}

ev, _, err := webhooks.Decode[PaymentConfirmed](opts)
```

Signature scheme: header is `x-opensettle-signature: t=<unix_seconds>,v1=<hex>` where `v1` is HMAC-SHA256 of `<unix_seconds>.<raw_body>` with the per-endpoint signing secret. Constant-time comparison via `hmac.Equal`. Default tolerance window is 5 minutes; tune via `Opts.Tolerance`.

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

Retries cover 5xx + 429 + transport errors with exponential backoff capped at 4 s. The SDK respects `Retry-After` (both delta-seconds and HTTP-date forms). Idempotency keys are automatically attached to money-adjacent writes and shared across retry attempts — supply [`WithIdempotencyKey`](#idempotency-keys) if you want to provide your own.

## Versioning

Strict semver. The major version tracks the HTTP API major version (`v1`). Minor versions add features without breakage; patch versions are bug fixes only. See [CHANGELOG.md](./CHANGELOG.md).

## License

[MIT](./LICENSE)
