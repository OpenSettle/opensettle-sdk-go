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
