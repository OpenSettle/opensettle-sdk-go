package opensettle

import (
	"errors"
	"net/http"
	"testing"
)

const paymentLinkJSON = `{"id":"pl_1","url":"https://opensettle.io/pay/abc123","description":"Tip jar","priceId":null,"amountMinor":0,"openAmount":true,"minAmountMinor":50,"maxAmountMinor":100000,"presetAmounts":[500,1000,2500],"currency":"USD","chain":"base","token":"USDC","successUrl":"https://example/ok","active":true,"createdAt":"2026-06-10T15:00:00.000Z"}`
const paymentLinkWrappedJSON = `{"paymentLink":` + paymentLinkJSON + `}`

func ptrInt64(v int64) *int64 { return &v }
func ptrStr(v string) *string { return &v }
func ptrBool(v bool) *bool    { return &v }

func TestPaymentLinks_Create_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentLinkWrappedJSON)
	out, err := c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{
		OpenAmount:    ptrBool(true),
		MinAmount:     ptrInt64(50),
		MaxAmount:     ptrInt64(100000),
		PresetAmounts: []int64{500, 1000, 2500},
		Description:   ptrStr("Tip jar"),
		Chain:         ChainBase,
		Token:         TokenUSDC,
		SuccessURL:    ptrStr("https://example/ok"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "pl_1" {
		t.Fatalf("id: %q", out.ID)
	}
	if out.URL != "https://opensettle.io/pay/abc123" {
		t.Fatalf("url: %q", out.URL)
	}
	if !out.OpenAmount {
		t.Fatalf("openAmount should be true")
	}
	if out.MinAmountMinor == nil || *out.MinAmountMinor != 50 {
		t.Fatalf("minAmountMinor: %+v", out.MinAmountMinor)
	}
	if len(out.PresetAmounts) != 3 || out.PresetAmounts[2] != 2500 {
		t.Fatalf("presetAmounts: %+v", out.PresetAmounts)
	}
	if out.Chain != ChainBase || out.Token != TokenUSDC {
		t.Fatalf("chain/token: %s/%s", out.Chain, out.Token)
	}
}

func TestPaymentLinks_Create_Method(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentLinkWrappedJSON)
	_, _ = c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{Amount: ptrInt64(1000)})
	r := s.lastRequest(t)
	if r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/payment_links" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestPaymentLinks_Create_AttachesIdempotencyKey(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentLinkWrappedJSON)
	_, _ = c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{Amount: ptrInt64(1000)})
	if got := s.lastRequest(t).Headers.Get("Idempotency-Key"); got == "" {
		t.Fatalf("missing key")
	}
}

func TestPaymentLinks_Create_HonorsExplicitIdempotencyKey(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentLinkWrappedJSON)
	_, _ = c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{Amount: ptrInt64(1000)}, WithIdempotencyKey("my-key"))
	if got := s.lastRequest(t).Headers.Get("Idempotency-Key"); got != "my-key" {
		t.Fatalf("key: %q", got)
	}
}

func TestPaymentLinks_Create_FixedAmountBodySerialization(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentLinkWrappedJSON)
	_, _ = c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{
		Amount:      ptrInt64(4200),
		Currency:    ptrStr("EUR"),
		Description: ptrStr("One product"),
		Chain:       ChainBase,
		Token:       TokenUSDC,
		SuccessURL:  ptrStr("https://example/ok"),
	})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	// JSON numbers decode to float64.
	if got, ok := body["amount"].(float64); !ok || got != 4200 {
		t.Fatalf("amount: %v (ok=%v)", body["amount"], ok)
	}
	if body["currency"] != "EUR" {
		t.Fatalf("currency: %v", body["currency"])
	}
	if body["chain"] != "base" {
		t.Fatalf("chain: %v", body["chain"])
	}
	// Only the fixed-amount source should be on the wire.
	if _, ok := body["priceId"]; ok {
		t.Fatalf("priceId should be omitempty")
	}
	if _, ok := body["openAmount"]; ok {
		t.Fatalf("openAmount should be omitempty")
	}
	if _, ok := body["minAmount"]; ok {
		t.Fatalf("minAmount should be omitempty")
	}
}

func TestPaymentLinks_Create_OmitsUnsetFields(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentLinkWrappedJSON)
	// PriceID-based link: no amount / open-amount / bounds fields set, so
	// none of them should be serialized.
	_, _ = c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{
		PriceID:    ptrStr("pr_1"),
		SuccessURL: ptrStr("https://example/ok"),
	})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	if body["priceId"] != "pr_1" {
		t.Fatalf("priceId: %v", body["priceId"])
	}
	for _, k := range []string{"amount", "openAmount", "minAmount", "maxAmount", "presetAmounts", "currency", "description", "chain", "token", "metadata"} {
		if _, ok := body[k]; ok {
			t.Fatalf("%s should be omitempty when unset", k)
		}
	}
}

func TestPaymentLinks_Create_InvalidRequest(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(400, `{"error":{"code":"invalid_request","message":"exactly one amount source","param":"amount"}}`)
	_, err := c.PaymentLinks.Create(bgCtx(), CreatePaymentLinkRequest{})
	var target *InvalidRequestError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Param != "amount" {
		t.Fatalf("param: %q", target.Param)
	}
}

func TestPaymentLinks_List_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+paymentLinkJSON+`]}`)
	out, err := c.PaymentLinks.List(bgCtx())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "pl_1" {
		t.Fatalf("got %+v", out)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodGet {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/payment_links" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestPaymentLinks_List_Empty(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[]}`)
	out, err := c.PaymentLinks.List(bgCtx())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty, got %+v", out)
	}
}

func TestPaymentLinks_Deactivate_DELETE(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"ok":true}`)
	if err := c.PaymentLinks.Deactivate(bgCtx(), "pl_1"); err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodDelete {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/payment_links/pl_1" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestPaymentLinks_Deactivate_NotFound(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(404, `{"error":{"code":"not_found","message":"nope"}}`)
	err := c.PaymentLinks.Deactivate(bgCtx(), "pl_x")
	var target *NotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}
