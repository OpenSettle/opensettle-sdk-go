package opensettle

import (
	"strings"
	"testing"
	"time"
)

func TestWithIdempotencyKey_OverridesAutoOnCheckout(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)

	_, err := c.Checkouts.Create(
		bgCtx(),
		CreateCheckoutRequest{Mode: CheckoutPayment},
		WithIdempotencyKey("merchant-supplied-key-123"),
	)
	if err != nil {
		t.Fatal(err)
	}
	got := s.lastRequest(t).Headers.Get("Idempotency-Key")
	if got != "merchant-supplied-key-123" {
		t.Fatalf("Idempotency-Key: %q", got)
	}
}

func TestWithIdempotencyKey_OverridesAutoOnAllWriteResources(t *testing.T) {
	cases := []struct {
		name string
		call func(t *testing.T, c *Client) error
	}{
		{
			name: "Customers.Create",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "a@b.co"}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Invoices.Create",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Invoices.Create(bgCtx(), CreateInvoiceRequest{CustomerID: "cu_1"}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Invoices.Send",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Invoices.Send(bgCtx(), "in_1", WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Invoices.Remind",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Invoices.Remind(bgCtx(), "in_1", WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Payments.Refund",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Payments.RefundBroadcast",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Payments.RefundBroadcast(bgCtx(), "pay_1", RecordRefundBroadcastRequest{}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Products.Create",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Products.Create(bgCtx(), CreateProductRequest{Name: "X"}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Products.CreatePrice",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Products.CreatePrice(bgCtx(), "prod_1", CreatePriceRequest{}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Subscriptions.Create",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Subscriptions.Create(bgCtx(), CreateSubscriptionRequest{CustomerID: "cu_1", PriceID: "pr_1"}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "Subscriptions.ChangePlan",
			call: func(t *testing.T, c *Client) error {
				_, err := c.Subscriptions.ChangePlan(bgCtx(), "sub_1", ChangePlanRequest{}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "WebhookEndpoints.Create",
			call: func(t *testing.T, c *Client) error {
				_, err := c.WebhookEndpoints.Create(bgCtx(), CreateWebhookEndpointRequest{URL: "https://x.co/h"}, WithIdempotencyKey("k-customer"))
				return err
			},
		},
		{
			name: "WebhookEndpoints.RotateSecret",
			call: func(t *testing.T, c *Client) error {
				_, err := c.WebhookEndpoints.RotateSecret(bgCtx(), "we_1", WithIdempotencyKey("k-customer"))
				return err
			},
		},
	}
	// Every endpoint returns a generic-shaped success body. Each call
	// only inspects the Idempotency-Key header.
	successBody := `{"customer":{},"invoice":{},"checkout":{},"product":{},"price":{},"subscription":{},"payment":{},"endpoint":{},"signingSecret":"","unsignedTx":{"chain":"base","token":"USDC","to":"","amountMinor":0,"instructions":""}}`
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStubServer()
			defer s.Close()
			c := newTestClient(t, s)
			s.queue(200, successBody)
			if err := tc.call(t, c); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := s.lastRequest(t).Headers.Get("Idempotency-Key")
			if got != "k-customer" {
				t.Fatalf("Idempotency-Key: %q (want k-customer)", got)
			}
		})
	}
}

func TestWithIdempotencyKey_PreservedAcrossRetries(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	// First two attempts fail with retryable 503; third succeeds.
	s.queue(503, `{"error":{"code":"internal_error","message":"x"}}`)
	s.queue(503, `{"error":{"code":"internal_error","message":"x"}}`)
	s.queue(200, checkoutWrappedJSON)
	c.http.sleep = func(time.Duration) {} // race-skip the backoff
	c.http.maxRetries = 2

	_, err := c.Checkouts.Create(
		bgCtx(),
		CreateCheckoutRequest{Mode: CheckoutPayment},
		WithIdempotencyKey("retry-shared-key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, r := range allRecorded(s) {
		keys = append(keys, r.Headers.Get("Idempotency-Key"))
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 attempts, got %d", len(keys))
	}
	for i, k := range keys {
		if k != "retry-shared-key" {
			t.Fatalf("attempt %d had key %q", i, k)
		}
	}
	if !strings.Contains(s.lastRequest(t).Headers.Get("Idempotency-Key"), "retry-shared-key") {
		t.Fatalf("last key: %q", s.lastRequest(t).Headers.Get("Idempotency-Key"))
	}
}
