package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const subscriptionJSON = `{"id":"sub_1","workspaceId":"ws","customerId":"cu_1","productId":"prod_1","priceId":"price_1","amountMinor":1000,"currency":"USD","chain":"base","token":"USDC","status":"active","autopay":"allowance","allowanceTx":null,"allowanceRemaining":null,"trialEndsAt":null,"startedAt":"2026-05-12T15:00:00.000Z","currentPeriodEnd":"2026-06-12T15:00:00.000Z","nextBillingDate":"2026-06-12T15:00:00.000Z","canceledAt":null,"cancelReason":null,"pausedAt":null,"mrrMinor":1000,"metadata":null,"createdAt":"2026-05-12T15:00:00.000Z"}`

func TestSubscriptions_List(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+subscriptionJSON+`],"nextCursor":""}`)
	out, err := c.Subscriptions.List(bgCtx(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 {
		t.Fatal()
	}
}

func TestSubscriptions_List_Query(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	_, _ = c.Subscriptions.List(bgCtx(), &ListSubscriptionsQuery{CustomerID: "cu_1", Status: SubActive, Limit: 25})
	q := s.lastRequest(t).Query
	for _, want := range []string{"customerId=cu_1", "status=active", "limit=25"} {
		if !strings.Contains(q, want) {
			t.Errorf("missing %q in %q", want, q)
		}
	}
}

func TestSubscriptions_Retrieve(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	out, err := c.Subscriptions.Retrieve(bgCtx(), "sub_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != SubActive {
		t.Fatalf("status: %s", out.Status)
	}
}

func TestSubscriptions_Create(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	_, err := c.Subscriptions.Create(bgCtx(), CreateSubscriptionRequest{
		CustomerID: "cu_1",
		PriceID:    "price_1",
		Chain:      ChainBase,
		Token:      TokenUSDC,
		Autopay:    AutopayAllowance,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Headers.Get("Idempotency-Key") == "" {
		t.Fatalf("missing key")
	}
}

func TestSubscriptions_Pause(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	_, err := c.Subscriptions.Pause(bgCtx(), "sub_1")
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/subscriptions/sub_1/pause" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestSubscriptions_Resume(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	_, err := c.Subscriptions.Resume(bgCtx(), "sub_1")
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/subscriptions/sub_1/resume" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestSubscriptions_Cancel(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	_, err := c.Subscriptions.Cancel(bgCtx(), "sub_1", CancelSubscriptionRequest{Mode: CancelImmediately, Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	if body["mode"] != "immediately" {
		t.Fatalf("mode: %v", body["mode"])
	}
	if body["reason"] != "test" {
		t.Fatalf("reason: %v", body["reason"])
	}
}

func TestSubscriptions_Cancel_EmptyBody(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	_, err := c.Subscriptions.Cancel(bgCtx(), "sub_1", CancelSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	if _, ok := body["mode"]; ok {
		t.Fatalf("mode should be omitempty when empty")
	}
}

func TestSubscriptions_ChangePlan(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, subscriptionJSON)
	_, err := c.Subscriptions.ChangePlan(bgCtx(), "sub_1", ChangePlanRequest{PriceID: "price_2", ProrationMode: ProrationImmediately})
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/subscriptions/sub_1/change_plan" {
		t.Fatalf("path: %s", r.Path)
	}
	if r.Headers.Get("Idempotency-Key") == "" {
		t.Fatalf("missing key")
	}
	body := decodeBody[map[string]any](t, r.Body)
	if body["priceId"] != "price_2" {
		t.Fatalf("priceId: %v", body["priceId"])
	}
	if body["prorationMode"] != "immediately" {
		t.Fatalf("prorationMode: %v", body["prorationMode"])
	}
}

func TestSubscriptions_Cancel_InvalidStateTransition(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(422, `{"error":{"code":"invalid_state_transition","message":"already canceled"}}`)
	_, err := c.Subscriptions.Cancel(bgCtx(), "sub_1", CancelSubscriptionRequest{})
	var target *InvalidStateTransitionError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}
