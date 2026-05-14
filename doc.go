// Package opensettle is the official Go SDK for the OpenSettle API.
//
// OpenSettle is a non-custodial stablecoin billing platform: USDC and
// USDT subscriptions, one-shot checkouts, and invoices on Base, Ethereum,
// Polygon, Arbitrum, Solana, and Tron.
//
// # Quick start
//
//	import "github.com/OpenSettle/opensettle-sdk-go"
//
//	client, err := opensettle.NewClient(
//	    os.Getenv("OPENSETTLE_KEY"),
//	    os.Getenv("OPENSETTLE_WORKSPACE"),
//	    opensettle.WithTestMode(os.Getenv("ENV") != "production"),
//	)
//	if err != nil { log.Fatal(err) }
//
//	checkout, err := client.Checkouts.Create(ctx, opensettle.CreateCheckoutRequest{ /* ... */ })
//
// # Typed errors
//
// Every API error is mapped to a typed value reachable via errors.As:
//
//	if _, err := client.Payments.Refund(ctx, paymentID, req); err != nil {
//	    var rl *opensettle.RateLimitError
//	    if errors.As(err, &rl) {
//	        time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
//	        return retry()
//	    }
//	    var settle *opensettle.SettlementError
//	    if errors.As(err, &settle) && settle.Code == opensettle.CodeSigningRequired {
//	        return promptCustomerToReapprove()
//	    }
//	    return err
//	}
//
// # Pagination
//
// Every List endpoint has a sibling ListIter that returns an [Iter]
// walking every page transparently:
//
//	it := client.Customers.ListIter(ctx, &opensettle.ListCustomersQuery{Status: opensettle.CustomerActive})
//	for it.Next() {
//	    fmt.Println(it.Item().ID)
//	}
//	if err := it.Err(); err != nil { … }
//
// # Polling
//
// [WaitFor] is a polling helper for scripts and CI; production code should
// prefer webhooks. It calls retrieve every opts.Interval until the until
// predicate succeeds, then returns the resource:
//
//	pmt, err := opensettle.WaitFor(ctx,
//	    client.Payments.Retrieve, "pay_…",
//	    func(p *opensettle.Payment) bool { return p.Status == opensettle.PaymentConfirmed },
//	    opensettle.WaitOptions{Timeout: 2 * time.Minute, Interval: 2 * time.Second},
//	)
//
// On timeout you get a [*WaitTimeoutError] that carries the
// last-observed resource so you can inspect the partial state.
//
// # Idempotency keys
//
// Every money-adjacent write (Create, Refund, RotateSecret, …) auto-attaches
// an Idempotency-Key that is preserved across retry attempts. Pass
// [WithIdempotencyKey] to use a caller-chosen key when you have a natural
// deterministic id (e.g. your DB row id):
//
//	checkout, err := client.Checkouts.Create(ctx, req,
//	    opensettle.WithIdempotencyKey("order:" + order.ID),
//	)
//
// # Webhooks
//
// Webhook signature verification lives in the sub-package
// github.com/OpenSettle/opensettle-sdk-go/webhooks so consumers can use
// it without dragging the full SDK into their webhook handler.
package opensettle
