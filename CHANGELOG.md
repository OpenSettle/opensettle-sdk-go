# Changelog

All notable changes to `github.com/OpenSettle/opensettle-sdk-go` are listed here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Major versions track the HTTP API major version (`v1`).

## [0.1.0] - 2026-05-12

### Added

- Initial Go SDK release. Hand-written, zero third-party runtime dependencies.
- `Client` with functional options: `WithBaseURL`, `WithHTTPClient`,
  `WithMaxRetries`, `WithTimeout`, `WithTestMode`, `WithUserAgent`.
- Eight resources at full parity with `@opensettle/sdk` Node SDK v0.2.0:
  Checkouts, Customers, Invoices, Payments, Products (+ Prices),
  Subscriptions, WebhookEndpoints.
- Typed error hierarchy (13 codes) reachable via `errors.As`:
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
