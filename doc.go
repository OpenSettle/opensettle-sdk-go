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
// # Webhooks
//
// Webhook signature verification lives in the sub-package
// github.com/OpenSettle/opensettle-sdk-go/webhooks so consumers can use
// it without dragging the full SDK into their webhook handler.
package opensettle
