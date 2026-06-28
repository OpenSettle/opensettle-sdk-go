# Changelog

All notable changes to `github.com/OpenSettle/opensettle-sdk-go` are listed here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Major versions track the HTTP API major version (`v1`).

## [0.5.1] - 2026-06-28

Brings the SDK to parity with the published `@opensettle/sdk` / `opensettle`
`0.5.1` release. All changes are additive and backward-compatible.

### Added

- **`Customer.LifetimeValueMinor int`** (`json:"lifetimeValueMinor"`) — the
  settled lifetime value in MINOR units, computed live by the API as
  `SUM(amountMinor)` over payments with status `confirmed`/`refunded`.
  Present on every customer-returning endpoint (List / Retrieve / Create /
  Update). **This is the field to use for LTV display.** The existing
  `Customer.LifetimeValue` field mirrors the stored `lifetime_value` column,
  which is a never-written cache that is effectively always `0` — it is now
  documented as deprecated; read `LifetimeValueMinor` instead.

- **`Payment` carries sanctions-screening fields.** Three new fields mirror
  the API serializer (`apps/api/src/services/payments.ts`):
    - `ScreeningVerdict ScreeningVerdict` (`json:"screeningVerdict"`) — one of
      the new `ScreeningNotScreened` / `ScreeningClean` / `ScreeningFlagged` /
      `ScreeningError` consts. Default with the in-house no-op provider is
      `ScreeningNotScreened` on every row.
    - `ScreeningProvider *string` (`json:"screeningProvider"`)
    - `ScreeningScreenedAt *string` (`json:"screeningScreenedAt"`)

- **`ListPaymentsQuery` gains `ScreeningVerdict`, `SubscriptionID`, `From`,
  and `To`.** `ScreeningVerdict` filters the ops triage surface;
  `SubscriptionID` narrows to one subscription; `From`/`To` are inclusive
  ISO-8601 bounds on `createdAt` (any ISO 8601 string; `To` must be `>=`
  `From`) for windowing a reporting period / CSV export.

- **`ListInvoicesQuery` gains `From` and `To`** — same inclusive ISO-8601
  `createdAt` bounds as payments, for reporting-period filtering / CSV export.

- **`CreateCheckoutRequest` gains `Amount`, `Currency`, and `Description`**
  for ad-hoc one-time charges (`Mode=payment` only). The API now accepts
  exactly ONE charge source for `Mode=payment`: `InvoiceID` (an existing
  invoice), `PriceID` (a one-time price), or `Amount` — a brand-new ad-hoc
  path that needs no pre-created invoice or product price.
    - `Amount int` (`json:"amount,omitempty"`) — the charge in MINOR units
      (cents). Must be a positive integer; the zero value is omitted, so an
      unset `Amount` is never sent. Pair with `Chain` + `Token`.
    - `Currency string` (`json:"currency,omitempty"`) — ISO-4217 code for the
      ad-hoc `Amount`; defaults to USD server-side.
    - `Description string` (`json:"description,omitempty"`) — optional
      buyer-facing description for an ad-hoc `Amount` checkout.

### Notes

- `Products.DeletePrice` already returns `error` only (no envelope), matching
  the API's `204 No Content`; no change was needed (the `0.5.1` JS/Python
  `deletePrice → void` fix was a type-correctness fix that the Go signature
  never had wrong).
- `UpdateWebhookEndpointRequest.Description *string` already supports clearing
  the field (nil pointer), matching the `description: string | null` the
  server accepts.

## [0.5.0] - 2026-05-15

### Added

- **`WithIdempotencyKey(key string)`** — public per-call option that lets
  callers supply their own Idempotency-Key on any money-adjacent write
  (`Checkouts.Create`, `Customers.Create`, `Invoices.Create/Send/Remind`,
  `Payments.Refund/RefundBroadcast`, `Products.Create/CreatePrice`,
  `Subscriptions.Create/ChangePlan`, `WebhookEndpoints.Create/RotateSecret`).
  Useful when you have a natural deterministic id (e.g. your DB row id):

      client.Checkouts.Create(ctx, req, opensettle.WithIdempotencyKey("order:42"))

  When omitted, the SDK continues to auto-generate a UUIDv4 key as before.
  Keys are preserved across retry attempts in both modes.

- **`Iter[T]` is now stack-safe across arbitrarily long streams of empty
  pages.** Previously `Iter.Next()` self-recursed; now loops internally.
  Public API unchanged.

- **`ChainID` type** (alias-compatible with the legacy `ChainId` name) —
  Go convention uses all-caps initialisms. `ChainId` is preserved as a
  deprecated type alias (`type ChainId = ChainID`); existing code keeps
  compiling with no diff.

- Extensive new tests: every error subtype's `Unwrap()` is now covered by
  a table test, `WithIdempotencyKey` has parity coverage across all 12
  write methods plus a retry-key-preservation test, `Payments.ListIter`
  and `Invoices.ListIter` paths are covered, and `buildURL`'s failure path
  is exercised directly. Coverage rose from 85.3% → 90%+.

- New `example_test.go` provides runnable Examples for `NewClient`,
  `Checkouts.Create`, `Customers.ListIter`, `WaitFor`, `WithIdempotencyKey`,
  and the typed-error switch pattern. These render on pkg.go.dev.

- Comprehensive godoc on every exported type, request/response struct,
  enum const, and resource method — pkg.go.dev now has a full description
  on every symbol.

### Fixed

- **`Checkout` struct now matches the API response shape.** Three drifts
  resolved:
  - Added `Description *string` field (server returns it; was previously
    silently dropped during JSON decode). Note: this re-adds the field
    that 0.3.0 removed — the API does return `description` and the 0.3.0
    note was based on stale schema reads. See 0.3.0 entry below.
  - Added `HostedURL string` field (relative URL path for the buyer-facing
    hosted checkout; concatenate with the web origin to redirect).
  - Removed `UpdatedAt string` field (the API never returns this; readers
    would always see `""`).

- `generateIdempotencyKey` fallback path (crypto/rand failure) is now
  collision-safe under concurrent calls: appends an atomic counter to the
  nanosecond timestamp.

- Replaced hand-rolled query-string sort with `sort.Strings` from stdlib.

### Breaking

- Code that reads `checkout.UpdatedAt` no longer compiles. Replace with a
  derived timestamp from your own store, or use `checkout.CreatedAt` /
  `checkout.CompletedAt` if those fit your use case.

### Deprecated

- `ChainId` (lowercase d) is deprecated in favor of `ChainID`. The alias
  keeps existing code compiling; new code should prefer `ChainID`. The
  alias will remain until at least 1.0.

## [0.3.0] - 2026-05-12

### Added

- **Iterator API for paginated list endpoints** — `c.Customers.ListIter(ctx, q)`
  (and analogues on Products, Invoices, Payments, Subscriptions) returns
  `*Iter[T]` that walks every page transparently:

      it := c.Customers.ListIter(ctx, &opensettle.ListCustomersQuery{Status: opensettle.CustomerActive})
      for it.Next() {
          fmt.Println(it.Item().ID)
      }
      if err := it.Err(); err != nil { return err }

- **`WaitFor` polling helper** — `opensettle.WaitFor(ctx, c.Payments.Retrieve,
  "pay_…", func(p *Payment) bool { return p.Status == opensettle.PaymentConfirmed },
  opensettle.WaitOptions{Timeout: 2*time.Minute, Interval: 2*time.Second})`.
  Returns `*WaitTimeoutError` on timeout (carries the last-observed
  resource) and wraps `context.Canceled` cleanly.

### Fixed

- **`Checkout.Description` field removed.** *(See 0.4.0 — re-added once
  the actual API schema was re-verified; this removal was a false-positive
  driven by a stale schema read. Apologies for the churn.)*

## [0.2.0] - 2026-05-12

**Breaking** — discovered via live smoke against `api.opensettle.io`.
The 0.1.0 SDK was unmarshalling singleton responses into the resource
struct directly, but the API returns `{"customer": {…}}`,
`{"product": {…}}`, etc. — so every `Retrieve`/`Create`/`Update` was
returning a zero-valued struct.

### Fixed

- **Singleton response envelope unwrapping.** Each resource method
  now unmarshals into an internal wrapper struct
  (`struct { Customer *Customer ` + "`json:\"customer\"`" + ` }` etc.) and
  returns the inner pointer. Lists (`{data, nextCursor, hasMore}`)
  and multi-key envelopes (`Refund` returns `{payment, unsignedTx}`;
  `Create`/`RotateSecret` on webhook endpoints returns
  `{endpoint, signingSecret}`) pass through unchanged.
- **`RotateWebhookSecretResponse` shape**: was
  `{secret, rotationGraceUntil}`, never what the API actually returns.
  Replaced with a type alias to `CreateWebhookEndpointResponse`
  (the actual shape: `{Endpoint, SigningSecret}`). Callers using
  `.Secret` should switch to `.SigningSecret`, and `.RotationGraceUntil`
  now lives on the embedded `Endpoint.RotationGraceUntil` field.
- Test fixtures across all resources updated to match the real API's
  envelope shape — the prior fixtures masked the bug.

### Migration

- `RotateSecret(…).Secret` → `RotateSecret(…).SigningSecret`
- `RotateSecret(…).RotationGraceUntil` → `RotateSecret(…).Endpoint.RotationGraceUntil`
- No other source changes; resource method signatures are unchanged
  (same `*Customer, error` etc. returns — just now correctly populated).

## [0.1.0] - 2026-05-12

### Added

- Initial Go SDK release. Hand-written, zero third-party runtime dependencies.
- `Client` with functional options: `WithBaseURL`, `WithHTTPClient`,
  `WithMaxRetries`, `WithTimeout`, `WithTestMode`, `WithUserAgent`.
- Eight resources at full parity with `@opensettle/sdk` Node SDK v0.2.0:
  Checkouts, Customers, Invoices, Payments, Products (+ Prices),
  Subscriptions, WebhookEndpoints.
- Typed error hierarchy (15 codes) reachable via `errors.As`:
  `*InvalidRequestError`, `*InvalidStateTransitionError`,
  `*AuthenticationError`, `*ForbiddenError`, `*NotFoundError`,
  `*ConflictError`, `*RateLimitError` (with `RetryAfter`),
  `*SettlementError`, `*StepUpRequiredError`, `*APIError`,
  `*NetworkError`.
- HTTP transport: bounded retries on 5xx + 429 + transport errors,
  exponential backoff capped at 4 s, `Retry-After` honored (numeric and
  HTTP-date), context-driven cancellation/timeouts, automatic
  `Idempotency-Key` injection for money-adjacent writes.
- `webhooks` sub-package: HMAC-SHA256 verifier with timing-safe comparison,
  default 300 s tolerance window, typed `*VerificationError` with reason
  enum, generic `Decode[T]` helper for typed events.
