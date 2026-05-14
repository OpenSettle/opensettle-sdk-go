package opensettle_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	opensettle "github.com/OpenSettle/opensettle-sdk-go"
)

func ExampleNewClient() {
	client, err := opensettle.NewClient(
		os.Getenv("OPENSETTLE_KEY"),
		os.Getenv("OPENSETTLE_WORKSPACE"),
		opensettle.WithTestMode(true),
		opensettle.WithTimeout(15*time.Second),
		opensettle.WithMaxRetries(3),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleCheckoutsResource_Create() {
	client, _ := opensettle.NewClient("sk_test_x", "ws_test", opensettle.WithTestMode(true))
	ctx := context.Background()

	checkout, err := client.Checkouts.Create(ctx, opensettle.CreateCheckoutRequest{
		Mode:       opensettle.CheckoutPayment,
		CustomerID: "cu_123",
		InvoiceID:  "in_123",
		SuccessURL: "https://example.com/thanks",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(checkout.HostedURL)
}

func ExampleCustomersResource_ListIter() {
	client, _ := opensettle.NewClient("sk_test_x", "ws_test", opensettle.WithTestMode(true))
	ctx := context.Background()

	it := client.Customers.ListIter(ctx, &opensettle.ListCustomersQuery{
		Status: opensettle.CustomerActive,
		Limit:  100,
	})
	for it.Next() {
		fmt.Println(it.Item().ID)
	}
	if err := it.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleWaitFor() {
	client, _ := opensettle.NewClient("sk_test_x", "ws_test", opensettle.WithTestMode(true))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	payment, err := opensettle.WaitFor(
		ctx,
		client.Payments.Retrieve,
		"pay_123",
		func(p *opensettle.Payment) bool { return p.Status == opensettle.PaymentConfirmed },
		opensettle.WaitOptions{Timeout: 2 * time.Minute, Interval: 2 * time.Second},
	)
	if err != nil {
		var timeoutErr *opensettle.WaitTimeoutError
		if errors.As(err, &timeoutErr) {
			last, _ := timeoutErr.Last.(*opensettle.Payment)
			log.Fatalf("timed out; last status: %s", last.Status)
		}
		log.Fatal(err)
	}
	fmt.Println(payment.TxHash)
}

func ExampleWithIdempotencyKey() {
	client, _ := opensettle.NewClient("sk_test_x", "ws_test", opensettle.WithTestMode(true))
	ctx := context.Background()

	// Use the merchant's order id as the idempotency key — any number of
	// retries (from this machine or another) will collapse to the same
	// server-side checkout.
	checkout, err := client.Checkouts.Create(ctx,
		opensettle.CreateCheckoutRequest{
			Mode:       opensettle.CheckoutPayment,
			CustomerID: "cu_123",
			InvoiceID:  "in_123",
			SuccessURL: "https://example.com/thanks",
		},
		opensettle.WithIdempotencyKey("order:42"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(checkout.ID)
}

func ExampleClient_typedErrors() {
	client, _ := opensettle.NewClient("sk_test_x", "ws_test", opensettle.WithTestMode(true))
	ctx := context.Background()

	_, err := client.Payments.Refund(ctx, "pay_123", opensettle.InitiateRefundRequest{})
	if err == nil {
		return
	}
	var rl *opensettle.RateLimitError
	if errors.As(err, &rl) {
		time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
		return
	}
	var stepUp *opensettle.StepUpRequiredError
	if errors.As(err, &stepUp) {
		// Caller needs to re-auth with AAL=2 before the refund will succeed.
		log.Fatal("step-up required")
	}
	var settle *opensettle.SettlementError
	if errors.As(err, &settle) && settle.Code == opensettle.CodeSigningRequired {
		// Customer's wallet needs to re-approve the spend allowance.
		log.Fatal("re-approval required")
	}
	log.Fatal(err)
}
