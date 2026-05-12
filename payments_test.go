package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const paymentJSON = `{"id":"pay_1","workspaceId":"ws","customerId":"cu_1","subscriptionId":null,"invoiceId":"in_1","walletId":null,"amountMinor":5000,"feeMinor":50,"netMinor":4950,"currency":"USD","token":"USDC","chain":"base","status":"confirmed","failureReason":null,"description":null,"txHash":"0xabc","blockNumber":1234,"confirmations":12,"refundTxHash":null,"refundAmountMinor":null,"refundBroadcastAt":null,"refundedAt":null,"refundReason":null,"createdAt":"2026-05-12T15:00:00.000Z","confirmedAt":"2026-05-12T15:01:00.000Z"}`

const refundResponseJSON = `{"payment":` + paymentJSON + `,"unsignedTx":{"chain":"base","token":"USDC","to":"0xtoken","amountMinor":5000,"instructions":"sign and broadcast"}}`

func TestPayments_List(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+paymentJSON+`],"nextCursor":""}`)
	out, err := c.Payments.List(bgCtx(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 {
		t.Fatalf("data: %+v", out.Data)
	}
}

func TestPayments_List_Query(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	_, _ = c.Payments.List(bgCtx(), &ListPaymentsQuery{CustomerID: "cu_1", Status: PaymentConfirmed, Limit: 5})
	q := s.lastRequest(t).Query
	for _, want := range []string{"customerId=cu_1", "status=confirmed", "limit=5"} {
		if !strings.Contains(q, want) {
			t.Errorf("missing %q in %q", want, q)
		}
	}
}

func TestPayments_Retrieve(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentJSON)
	out, err := c.Payments.Retrieve(bgCtx(), "pay_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != PaymentConfirmed {
		t.Fatalf("status: %s", out.Status)
	}
	if out.TxHash == nil || *out.TxHash != "0xabc" {
		t.Fatalf("txHash: %+v", out.TxHash)
	}
}

func TestPayments_Refund_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, refundResponseJSON)
	out, err := c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{AmountMinor: 5000, Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Payment.ID != "pay_1" {
		t.Fatalf("payment: %+v", out.Payment)
	}
	if out.UnsignedTx.Chain != ChainBase {
		t.Fatalf("chain: %s", out.UnsignedTx.Chain)
	}
}

func TestPayments_Refund_Method(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, refundResponseJSON)
	_, _ = c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{})
	r := s.lastRequest(t)
	if r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/payments/pay_1/refund" {
		t.Fatalf("path: %s", r.Path)
	}
	if r.Headers.Get("Idempotency-Key") == "" {
		t.Fatalf("missing key")
	}
}

func TestPayments_Refund_StepUpRequired(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(401, `{"error":{"code":"aal_required","message":"step up"}}`)
	_, err := c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{})
	var target *StepUpRequiredError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestPayments_RefundBroadcast(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, paymentJSON)
	_, err := c.Payments.RefundBroadcast(bgCtx(), "pay_1", RecordRefundBroadcastRequest{RefundTxHash: "0xdead"})
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/payments/pay_1/refund/broadcast" {
		t.Fatalf("path: %s", r.Path)
	}
	body := decodeBody[map[string]any](t, r.Body)
	if body["refundTxHash"] != "0xdead" {
		t.Fatalf("body: %+v", body)
	}
}

func TestPayments_Refund_SettlementError(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(400, `{"error":{"code":"signing_required","message":"sign"}}`)
	_, err := c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{})
	var target *SettlementError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Code != CodeSigningRequired {
		t.Fatalf("code: %s", target.Code)
	}
}
