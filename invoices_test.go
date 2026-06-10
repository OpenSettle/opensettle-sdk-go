package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const invoiceJSON = `{"id":"in_1","workspaceId":"ws","number":"INV-1","customerId":"cu_1","subscriptionId":null,"amountMinor":5000,"currency":"USD","chain":"base","token":"USDC","status":"open","lineItems":[{"description":"item","quantity":1,"unitAmountMinor":5000}],"memo":null,"paymentId":null,"hostedUrl":"https://example/in_1","issuedAt":null,"dueAt":"2026-05-26T00:00:00.000Z","paidAt":null,"voidedAt":null,"metadata":null,"createdAt":"2026-05-12T15:00:00.000Z"}`
const invoiceWrappedJSON = `{"invoice":` + invoiceJSON + `}`

func TestInvoices_List(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+invoiceJSON+`],"nextCursor":""}`)
	out, err := c.Invoices.List(bgCtx(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 || out.Data[0].ID != "in_1" {
		t.Fatalf("got %+v", out)
	}
}

func TestInvoices_List_QueryParams(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	_, _ = c.Invoices.List(bgCtx(), &ListInvoicesQuery{
		CustomerID: "cu_1",
		Status:     InvoiceOpen,
		From:       "2026-04-01",
		To:         "2026-06-30",
		Limit:      10,
	})
	q := s.lastRequest(t).Query
	for _, want := range []string{"customerId=cu_1", "status=open", "from=2026-04-01", "to=2026-06-30", "limit=10"} {
		if !strings.Contains(q, want) {
			t.Errorf("missing %q in %q", want, q)
		}
	}
}

func TestInvoices_Retrieve(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, invoiceWrappedJSON)
	out, err := c.Invoices.Retrieve(bgCtx(), "in_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.HostedURL != "https://example/in_1" {
		t.Fatalf("got %q", out.HostedURL)
	}
}

func TestInvoices_Create_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, invoiceWrappedJSON)
	out, err := c.Invoices.Create(bgCtx(), CreateInvoiceRequest{
		CustomerID: "cu_1",
		Chain:      ChainBase,
		Token:      TokenUSDC,
		LineItems:  []LineItem{{Description: "item", Quantity: 1, UnitAmountMinor: 5000}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "in_1" {
		t.Fatalf("id: %s", out.ID)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
	if got := r.Headers.Get("Idempotency-Key"); got == "" {
		t.Fatal("missing idempotency key")
	}
}

func TestInvoices_Send_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, invoiceWrappedJSON)
	_, err := c.Invoices.Send(bgCtx(), "in_1")
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/invoices/in_1/send" {
		t.Fatalf("path: %s", r.Path)
	}
	if got := r.Headers.Get("Idempotency-Key"); got == "" {
		t.Fatal("missing key")
	}
}

func TestInvoices_Remind_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, invoiceWrappedJSON)
	_, err := c.Invoices.Remind(bgCtx(), "in_1")
	if err != nil {
		t.Fatal(err)
	}
	if got := s.lastRequest(t).Path; got != "/v1/workspaces/ws_test/invoices/in_1/reminder" {
		t.Fatalf("path: %s", got)
	}
}

func TestInvoices_Void_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, invoiceWrappedJSON)
	_, err := c.Invoices.Void(bgCtx(), "in_1")
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/invoices/in_1/void" {
		t.Fatalf("path: %s", r.Path)
	}
	// Void shouldn't auto-attach an idempotency key (matches Node SDK)
	if got := r.Headers.Get("Idempotency-Key"); got != "" {
		t.Fatalf("should not have key: %q", got)
	}
}

func TestInvoices_Void_InvalidStateTransition(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(422, `{"error":{"code":"invalid_state_transition","message":"already paid"}}`)
	_, err := c.Invoices.Void(bgCtx(), "in_1")
	var target *InvalidStateTransitionError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestInvoices_Create_LineItemsSerialize(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, invoiceWrappedJSON)
	_, _ = c.Invoices.Create(bgCtx(), CreateInvoiceRequest{
		CustomerID: "cu_1",
		Chain:      ChainEthereum,
		Token:      TokenUSDT,
		LineItems: []LineItem{
			{Description: "a", Quantity: 2, UnitAmountMinor: 1000},
			{Description: "b", Quantity: 1, UnitAmountMinor: 2000},
		},
	})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	items, ok := body["lineItems"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("lineItems: %+v", body["lineItems"])
	}
}

func TestInvoices_Create_RateLimited(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queueWith(cannedResponse{Status: 429, Body: `{"error":{"code":"rate_limited","message":"slow"}}`, Headers: map[string]string{"Retry-After": "5"}})
	_, err := c.Invoices.Create(bgCtx(), CreateInvoiceRequest{CustomerID: "cu_1"})
	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.RetryAfter != 5 {
		t.Fatalf("retryAfter: %v", target.RetryAfter)
	}
}
