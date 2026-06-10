package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const checkoutJSON = `{"id":"co_1","workspaceId":"ws","mode":"payment","status":"open","customerId":"cu_1","invoiceId":"in_1","priceId":null,"amountMinor":1000,"currency":"USD","chain":"base","token":"USDC","description":null,"successUrl":"https://example/ok","cancelUrl":null,"expiresAt":"2026-05-12T16:00:00.000Z","completedAt":null,"metadata":null,"createdAt":"2026-05-12T15:00:00.000Z","hostedUrl":"/checkout/abc123"}`
const checkoutWrappedJSON = `{"checkout":` + checkoutJSON + `}`

func TestCheckouts_Create_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	out, err := c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{
		Mode:       CheckoutPayment,
		CustomerID: "cu_1",
		InvoiceID:  "in_1",
		SuccessURL: "https://example/ok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "co_1" {
		t.Fatalf("id: %q", out.ID)
	}
	if out.Status != CheckoutOpen {
		t.Fatalf("status: %s", out.Status)
	}
	if out.HostedURL != "/checkout/abc123" {
		t.Fatalf("hostedUrl: %q", out.HostedURL)
	}
}

func TestCheckouts_Create_Method(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{Mode: CheckoutPayment})
	r := s.lastRequest(t)
	if r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/checkouts" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestCheckouts_Create_AttachesIdempotencyKey(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{Mode: CheckoutPayment})
	if got := s.lastRequest(t).Headers.Get("Idempotency-Key"); got == "" {
		t.Fatalf("missing key")
	}
}

func TestCheckouts_Create_BodySerialization(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{
		Mode:       CheckoutSubscription,
		CustomerID: "cu_1",
		PriceID:    "pr_1",
		SuccessURL: "https://example/ok",
	})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	if body["mode"] != "subscription" {
		t.Fatalf("mode: %v", body["mode"])
	}
	if body["customerId"] != "cu_1" {
		t.Fatalf("customerId: %v", body["customerId"])
	}
	if body["priceId"] != "pr_1" {
		t.Fatalf("priceId: %v", body["priceId"])
	}
	if _, ok := body["invoiceId"]; ok {
		t.Fatalf("invoiceId should be omitempty")
	}
}

func TestCheckouts_Create_AdHocAmountSerialization(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{
		Mode:        CheckoutPayment,
		CustomerID:  "cu_1",
		Amount:      4200,
		Currency:    "EUR",
		Description: "Custom one-time charge",
		Chain:       ChainBase,
		Token:       TokenUSDC,
		SuccessURL:  "https://example/ok",
	})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	// JSON numbers decode to float64.
	if got, ok := body["amount"].(float64); !ok || got != 4200 {
		t.Fatalf("amount: %v (ok=%v)", body["amount"], ok)
	}
	if body["currency"] != "EUR" {
		t.Fatalf("currency: %v", body["currency"])
	}
	if body["description"] != "Custom one-time charge" {
		t.Fatalf("description: %v", body["description"])
	}
	// Only one charge source should be on the wire.
	if _, ok := body["invoiceId"]; ok {
		t.Fatalf("invoiceId should be omitempty")
	}
	if _, ok := body["priceId"]; ok {
		t.Fatalf("priceId should be omitempty")
	}
}

func TestCheckouts_Create_OmitsAdHocFieldsWhenUnset(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	// Invoice-based checkout: none of the ad-hoc fields are set, so none
	// of them should be serialized (amount==0 is the omitted zero value).
	_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{
		Mode:       CheckoutPayment,
		CustomerID: "cu_1",
		InvoiceID:  "in_1",
		SuccessURL: "https://example/ok",
	})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	for _, k := range []string{"amount", "currency", "description"} {
		if _, ok := body[k]; ok {
			t.Fatalf("%s should be omitempty when unset", k)
		}
	}
}

func TestCheckouts_Retrieve_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	out, err := c.Checkouts.Retrieve(bgCtx(), "co_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "co_1" {
		t.Fatalf("id: %q", out.ID)
	}
	if r := s.lastRequest(t); r.Method != http.MethodGet {
		t.Fatalf("method: %s", r.Method)
	}
}

func TestCheckouts_Retrieve_NotFound(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(404, `{"error":{"code":"not_found","message":"nope"}}`)
	_, err := c.Checkouts.Retrieve(bgCtx(), "co_x")
	var target *NotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestCheckouts_Create_InvalidRequest(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(400, `{"error":{"code":"invalid_request","message":"bad","param":"successUrl"}}`)
	_, err := c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{Mode: CheckoutPayment})
	var target *InvalidRequestError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Param != "successUrl" {
		t.Fatalf("param: %q", target.Param)
	}
}

func TestCheckouts_Create_Forbidden(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(403, `{"error":{"code":"forbidden","message":"no"}}`)
	_, err := c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{Mode: CheckoutPayment})
	var target *ForbiddenError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestCheckouts_BodyIsJSON(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, checkoutWrappedJSON)
	_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{Mode: CheckoutPayment, SuccessURL: "https://a"})
	got := string(s.lastRequest(t).Body)
	if !strings.HasPrefix(got, "{") || !strings.HasSuffix(got, "}") {
		t.Fatalf("got %q", got)
	}
}
