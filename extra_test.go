package opensettle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// extra_test.go is a grab-bag of additional edge cases that don't fit
// neatly into a single resource's file. Kept here so the per-resource
// files stay focused on their happy path.

// --- error-mapping sweep across status codes --------------------------

func TestStatusCode_400Codes(t *testing.T) {
	cases := []struct {
		status int
		code   string
		check  func(error) bool
	}{
		{400, "invalid_request", func(e error) bool { return errors.As(e, new(*InvalidRequestError)) }},
		{401, "unauthorized", func(e error) bool { return errors.As(e, new(*AuthenticationError)) }},
		{401, "aal_required", func(e error) bool { return errors.As(e, new(*StepUpRequiredError)) }},
		{403, "forbidden", func(e error) bool { return errors.As(e, new(*ForbiddenError)) }},
		{404, "not_found", func(e error) bool { return errors.As(e, new(*NotFoundError)) }},
		{409, "conflict", func(e error) bool { return errors.As(e, new(*ConflictError)) }},
		{422, "invalid_state_transition", func(e error) bool { return errors.As(e, new(*InvalidStateTransitionError)) }},
		{429, "rate_limited", func(e error) bool { return errors.As(e, new(*RateLimitError)) }},
		{500, "internal_error", func(e error) bool { return errors.As(e, new(*APIError)) }},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", tc.status, tc.code), func(t *testing.T) {
			s := newStubServer()
			defer s.Close()
			c := newTestClient(t, s)
			s.queue(tc.status, `{"error":{"code":"`+tc.code+`","message":"x"}}`)
			_, err := c.Customers.Retrieve(bgCtx(), "cu_1")
			if !tc.check(err) {
				t.Fatalf("wrong type for %d/%s: %T %v", tc.status, tc.code, err, err)
			}
		})
	}
}

func TestStatusCode_AllSettlementCodes(t *testing.T) {
	for _, code := range []string{"chain_reverted", "insufficient_confirmations", "signing_required"} {
		t.Run(code, func(t *testing.T) {
			s := newStubServer()
			defer s.Close()
			c := newTestClient(t, s)
			s.queue(400, `{"error":{"code":"`+code+`","message":"x"}}`)
			_, err := c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{})
			var target *SettlementError
			if !errors.As(err, &target) {
				t.Fatalf("got %T", err)
			}
			if string(target.Code) != code {
				t.Fatalf("code: %s", target.Code)
			}
		})
	}
}

// --- HTTP retry / backoff edge cases ----------------------------------

func TestHTTP_RetryThenSuccessAfterMultipleFailures(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 3)
	s.queue(500, "")
	s.queue(503, "")
	s.queue(502, "")
	s.queue(200, `{"id":"cu_1"}`)
	_, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if len(s.requests) != 4 {
		t.Fatalf("got %d requests", len(s.requests))
	}
}

func TestHTTP_RetryReturnsLastError(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 2)
	s.queue(500, "")
	s.queue(500, "")
	s.queue(503, `{"error":{"code":"internal_error","message":"final"}}`)
	_, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err == nil {
		t.Fatal()
	}
	if !strings.Contains(err.Error(), "final") && !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected 503 to win, got %v", err)
	}
}

func TestHTTP_RetryAfterRespectedOverBackoff(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 1)

	var slept []time.Duration
	c.http.sleep = func(d time.Duration) { slept = append(slept, d) }

	s.queueWith(cannedResponse{
		Status:  429,
		Body:    `{"error":{"code":"rate_limited","message":"slow"}}`,
		Headers: map[string]string{"Retry-After": "3"},
	})
	s.queue(200, `{"id":"cu_1"}`)
	_, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if len(slept) != 1 {
		t.Fatalf("expected 1 sleep, got %d", len(slept))
	}
	if slept[0] != 3*time.Second {
		t.Fatalf("expected 3s, got %v", slept[0])
	}
}

func TestHTTP_BackoffApplied(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 2)
	var slept []time.Duration
	c.http.sleep = func(d time.Duration) { slept = append(slept, d) }
	s.queue(500, "")
	s.queue(500, "")
	s.queue(200, `{"id":"cu_1"}`)
	_, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(slept) != 2 {
		t.Fatalf("slept: %v", slept)
	}
	if slept[0] != 250*time.Millisecond {
		t.Fatalf("first backoff: %v", slept[0])
	}
	if slept[1] != 500*time.Millisecond {
		t.Fatalf("second backoff: %v", slept[1])
	}
}

func TestHTTP_NoSleepOnNoRetry(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 0)
	var slept []time.Duration
	c.http.sleep = func(d time.Duration) { slept = append(slept, d) }
	s.queue(500, "")
	_, _ = c.Customers.Retrieve(bgCtx(), "cu_1")
	if len(slept) != 0 {
		t.Fatalf("should not sleep when retries=0, got %v", slept)
	}
}

// --- Header / body assertions -----------------------------------------

func TestHTTP_BearerTokenExactlyOnce(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"id":"cu_1"}`)
	_, _ = c.Customers.Retrieve(bgCtx(), "cu_1")
	got := s.lastRequest(t).Headers.Values("Authorization")
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestHTTP_UserAgentVersionEmbedded(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"id":"cu_1"}`)
	_, _ = c.Customers.Retrieve(bgCtx(), "cu_1")
	got := s.lastRequest(t).Headers.Get("User-Agent")
	if !strings.Contains(got, Version) {
		t.Fatalf("user-agent missing version %q: %q", Version, got)
	}
}

func TestHTTP_CustomUserAgentEmbedded(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c, err := NewClient("sk_test_x", "ws_test",
		WithBaseURL(s.server.URL),
		WithMaxRetries(0),
		WithUserAgent("dropzona/1.2.3"),
	)
	if err != nil {
		t.Fatal(err)
	}
	c.http.sleep = func(time.Duration) {}
	s.queue(200, `{"id":"cu_1"}`)
	_, _ = c.Customers.Retrieve(bgCtx(), "cu_1")
	got := s.lastRequest(t).Headers.Get("User-Agent")
	if !strings.Contains(got, "dropzona/1.2.3") || !strings.HasPrefix(got, "opensettle-go/") {
		t.Fatalf("got %q", got)
	}
}

func TestHTTP_RetriesShareSameIdempotencyKey(t *testing.T) {
	// Idempotency-Key is generated once and must be identical across
	// retried attempts so the server can dedupe.
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 1)
	s.queue(500, "")
	s.queue(200, customerJSON)
	_, err := c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "a@b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.requests) != 2 {
		t.Fatalf("got %d requests", len(s.requests))
	}
	k1 := s.requests[0].Headers.Get("Idempotency-Key")
	k2 := s.requests[1].Headers.Get("Idempotency-Key")
	if k1 == "" || k1 != k2 {
		t.Fatalf("keys differ across retries: %q vs %q", k1, k2)
	}
}

func TestHTTP_RetriesSendSameBody(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	c = withRetries(c, 1)
	s.queue(503, "")
	s.queue(200, customerJSON)
	_, err := c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "x@y"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.requests) != 2 {
		t.Fatalf("got %d requests", len(s.requests))
	}
	if string(s.requests[0].Body) != string(s.requests[1].Body) {
		t.Fatalf("bodies differ across retries: %q vs %q", s.requests[0].Body, s.requests[1].Body)
	}
}

// --- Concurrency ------------------------------------------------------

func TestClient_SafeForConcurrentUse(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	for i := 0; i < 50; i++ {
		s.queue(200, customerJSON)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Customers.Retrieve(bgCtx(), "cu_1")
			if err != nil {
				t.Errorf("got %v", err)
			}
		}()
	}
	wg.Wait()
	if len(s.requests) != 50 {
		t.Fatalf("got %d", len(s.requests))
	}
}

// --- Context propagation ----------------------------------------------

func TestHTTP_ContextDeadlinePropagates(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	ctx, cancel := context.WithTimeout(bgCtx(), 50*time.Millisecond)
	defer cancel()
	s.queueWith(cannedResponse{Status: 200, Body: `{"id":"cu_1"}`, Delay: 200 * time.Millisecond})
	_, err := c.Customers.Retrieve(ctx, "cu_1")
	if err == nil {
		t.Fatal("expected error")
	}
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("got %T: %v", err, err)
	}
}

// --- Resource path coverage -------------------------------------------

func TestPaths_Checkouts(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	cases := []struct {
		op   func()
		path string
	}{
		{func() {
			s.queue(200, checkoutJSON)
			_, _ = c.Checkouts.Create(bgCtx(), CreateCheckoutRequest{Mode: CheckoutPayment})
		}, "/v1/workspaces/ws_test/checkouts"},
		{func() { s.queue(200, checkoutJSON); _, _ = c.Checkouts.Retrieve(bgCtx(), "co_1") }, "/v1/workspaces/ws_test/checkouts/co_1"},
	}
	for _, tc := range cases {
		tc.op()
		if got := s.lastRequest(t).Path; got != tc.path {
			t.Errorf("got %s want %s", got, tc.path)
		}
	}
}

func TestPaths_Customers(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	_, _ = c.Customers.List(bgCtx(), nil)
	if got := s.lastRequest(t).Path; got != "/v1/workspaces/ws_test/customers" {
		t.Fatalf("got %s", got)
	}
}

func TestPaths_Subscriptions(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	cases := []struct {
		op   func()
		path string
	}{
		{func() { s.queue(200, subscriptionJSON); _, _ = c.Subscriptions.Pause(bgCtx(), "sub_1") }, "/v1/workspaces/ws_test/subscriptions/sub_1/pause"},
		{func() { s.queue(200, subscriptionJSON); _, _ = c.Subscriptions.Resume(bgCtx(), "sub_1") }, "/v1/workspaces/ws_test/subscriptions/sub_1/resume"},
		{func() {
			s.queue(200, subscriptionJSON)
			_, _ = c.Subscriptions.Cancel(bgCtx(), "sub_1", CancelSubscriptionRequest{})
		}, "/v1/workspaces/ws_test/subscriptions/sub_1/cancel"},
		{func() {
			s.queue(200, subscriptionJSON)
			_, _ = c.Subscriptions.ChangePlan(bgCtx(), "sub_1", ChangePlanRequest{PriceID: "price_2"})
		}, "/v1/workspaces/ws_test/subscriptions/sub_1/change_plan"},
	}
	for _, tc := range cases {
		tc.op()
		if got := s.lastRequest(t).Path; got != tc.path {
			t.Errorf("got %s want %s", got, tc.path)
		}
	}
}

func TestPaths_Invoices(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	cases := []struct {
		op   func()
		path string
	}{
		{func() { s.queue(200, invoiceJSON); _, _ = c.Invoices.Send(bgCtx(), "in_1") }, "/v1/workspaces/ws_test/invoices/in_1/send"},
		{func() { s.queue(200, invoiceJSON); _, _ = c.Invoices.Remind(bgCtx(), "in_1") }, "/v1/workspaces/ws_test/invoices/in_1/reminder"},
		{func() { s.queue(200, invoiceJSON); _, _ = c.Invoices.Void(bgCtx(), "in_1") }, "/v1/workspaces/ws_test/invoices/in_1/void"},
	}
	for _, tc := range cases {
		tc.op()
		if got := s.lastRequest(t).Path; got != tc.path {
			t.Errorf("got %s want %s", got, tc.path)
		}
	}
}

func TestPaths_Payments(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	cases := []struct {
		op   func()
		path string
	}{
		{func() {
			s.queue(200, refundResponseJSON)
			_, _ = c.Payments.Refund(bgCtx(), "pay_1", InitiateRefundRequest{})
		}, "/v1/workspaces/ws_test/payments/pay_1/refund"},
		{func() {
			s.queue(200, paymentJSON)
			_, _ = c.Payments.RefundBroadcast(bgCtx(), "pay_1", RecordRefundBroadcastRequest{RefundTxHash: "0x1"})
		}, "/v1/workspaces/ws_test/payments/pay_1/refund/broadcast"},
	}
	for _, tc := range cases {
		tc.op()
		if got := s.lastRequest(t).Path; got != tc.path {
			t.Errorf("got %s want %s", got, tc.path)
		}
	}
}

func TestPaths_WebhookEndpoints(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	cases := []struct {
		op   func()
		path string
	}{
		{func() { s.queue(200, `{"data":[]}`); _, _ = c.WebhookEndpoints.List(bgCtx()) }, "/v1/workspaces/ws_test/webhook_endpoints"},
		{func() { s.queue(200, endpointJSON); _, _ = c.WebhookEndpoints.Retrieve(bgCtx(), "we_1") }, "/v1/workspaces/ws_test/webhook_endpoints/we_1"},
		{func() {
			s.queue(200, `{"secret":"s","rotationGraceUntil":""}`)
			_, _ = c.WebhookEndpoints.RotateSecret(bgCtx(), "we_1", RotateWebhookSecretRequest{})
		}, "/v1/workspaces/ws_test/webhook_endpoints/we_1/rotate"},
		{func() {
			s.queue(200, `{"ok":true,"status":200,"latencyMs":1}`)
			_, _ = c.WebhookEndpoints.Test(bgCtx(), "we_1", TestWebhookEndpointRequest{EventType: "t"})
		}, "/v1/workspaces/ws_test/webhook_endpoints/we_1/test"},
	}
	for _, tc := range cases {
		tc.op()
		if got := s.lastRequest(t).Path; got != tc.path {
			t.Errorf("got %s want %s", got, tc.path)
		}
	}
}

func TestPaths_Products(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	cases := []struct {
		op   func()
		path string
	}{
		{func() { s.queue(200, `{"data":[]}`); _, _ = c.Products.ListPrices(bgCtx(), "prod_1") }, "/v1/workspaces/ws_test/products/prod_1/prices"},
		{func() {
			s.queue(200, priceJSON)
			_, _ = c.Products.CreatePrice(bgCtx(), "prod_1", CreatePriceRequest{Amount: 100, Interval: PriceMonth})
		}, "/v1/workspaces/ws_test/products/prod_1/prices"},
		{func() { s.queue(204, ""); _ = c.Products.DeletePrice(bgCtx(), "price_1") }, "/v1/workspaces/ws_test/prices/price_1"},
	}
	for _, tc := range cases {
		tc.op()
		if got := s.lastRequest(t).Path; got != tc.path {
			t.Errorf("got %s want %s", got, tc.path)
		}
	}
}

// --- Method coverage --------------------------------------------------

func TestMethods_AllResources(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)

	type call struct {
		op     func()
		method string
	}
	cases := []call{
		{func() { s.queue(200, customerJSON); _, _ = c.Customers.Retrieve(bgCtx(), "x") }, http.MethodGet},
		{func() {
			s.queue(200, customerJSON)
			_, _ = c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "a@b"})
		}, http.MethodPost},
		{func() {
			s.queue(200, customerJSON)
			n := "n"
			_, _ = c.Customers.Update(bgCtx(), "x", UpdateCustomerRequest{Name: &n})
		}, http.MethodPatch},
		{func() { s.queue(204, ""); _ = c.Customers.Delete(bgCtx(), "x") }, http.MethodDelete},
	}
	for _, tc := range cases {
		tc.op()
		got := s.lastRequest(t).Method
		if got != tc.method {
			t.Errorf("got %s want %s", got, tc.method)
		}
	}
}

// --- Query encoding -- broader cases ----------------------------------

func TestEncodeQuery_Int64(t *testing.T) {
	got := encodeQuery(map[string]any{"n": int64(99)})
	if got != "n=99" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeQuery_Float64(t *testing.T) {
	got := encodeQuery(map[string]any{"r": 1.5})
	if got != "r=1.5" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeQuery_BoolFalse(t *testing.T) {
	got := encodeQuery(map[string]any{"active": false})
	if got != "active=false" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeQuery_MultipleKeysSorted(t *testing.T) {
	got := encodeQuery(map[string]any{"z": "1", "a": "1", "m": "1"})
	if got != "a=1&m=1&z=1" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeQuery_FallbackToFmt(t *testing.T) {
	type cstr struct{ v string }
	cstrVal := cstr{v: "x"}
	got := encodeQuery(map[string]any{"obj": cstrVal})
	if !strings.HasPrefix(got, "obj=") {
		t.Fatalf("got %q", got)
	}
}

// --- BackoffFor more cases --------------------------------------------

func TestBackoffFor_NegativeAttempt(t *testing.T) {
	// Function expects non-negative; passing a weird value shouldn't
	// crash. attempt < 0 will math.Pow to a fraction, capped at 0+...
	got := backoffFor(-1, 0)
	if got < 0 {
		t.Fatalf("backoff went negative: %v", got)
	}
}

func TestBackoffFor_HighAttemptCaps(t *testing.T) {
	got := backoffFor(20, 0)
	if got != 4*time.Second {
		t.Fatalf("got %v", got)
	}
}

func TestBackoffFor_ZeroRetryAfterFallsToExponential(t *testing.T) {
	got := backoffFor(2, 0)
	if got != 1*time.Second {
		t.Fatalf("got %v", got)
	}
}

func TestBackoffFor_FloatRetryAfter(t *testing.T) {
	got := backoffFor(0, 0.5)
	if got != 500*time.Millisecond {
		t.Fatalf("got %v", got)
	}
}

// --- More retry-after parse cases -------------------------------------

func TestParseRetryAfter_Zero(t *testing.T) {
	if got := parseRetryAfter("0", time.Now()); got != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestParseRetryAfter_LargeNumber(t *testing.T) {
	if got := parseRetryAfter("3600", time.Now()); got != 3600 {
		t.Fatalf("got %v", got)
	}
}

func TestParseRetryAfter_HTTPDateNow(t *testing.T) {
	now := time.Now().UTC()
	got := parseRetryAfter(now.Format(http.TimeFormat), now)
	if got < 0 {
		t.Fatalf("got %v", got)
	}
}

// --- ConfigError ------------------------------------------------------

func TestConfigError_ErrorString_PrefixedConsistently(t *testing.T) {
	e := &ConfigError{Message: "x"}
	if e.Error() != "opensettle config: x" {
		t.Fatalf("got %q", e.Error())
	}
}

// --- Idempotency-Key uniqueness in production code --------------------

func TestGenerateIdempotencyKey_LooksLikeUUID(t *testing.T) {
	// Run twice; both should pass the format checks individually.
	for i := 0; i < 5; i++ {
		k := generateIdempotencyKey()
		if len(k) != 36 {
			t.Fatalf("iteration %d: len %d (%q)", i, len(k), k)
		}
	}
}

// --- Response decoding edge cases --------------------------------------

func TestHTTP_RetrieveWithNullableFields(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	// All optional fields null — should still decode cleanly.
	s.queue(200, `{"id":"cu_1","workspaceId":"ws","email":"a@b","name":"","wallet":null,"country":null,"status":"churned","activeSubscriptions":0,"lifetimeValue":0,"metadata":null,"createdAt":"","deletedAt":null}`)
	out, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Wallet != nil {
		t.Fatalf("wallet: %+v", out.Wallet)
	}
	if out.Status != CustomerChurned {
		t.Fatalf("status: %s", out.Status)
	}
}

func TestHTTP_RetrieveWithMetadata(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"id":"cu_1","workspaceId":"ws","email":"a@b","name":"","wallet":null,"country":null,"status":"active","activeSubscriptions":0,"lifetimeValue":0,"metadata":{"plan":"pro","seats":5},"createdAt":"","deletedAt":null}`)
	out, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Metadata["plan"] != "pro" {
		t.Fatalf("plan: %v", out.Metadata["plan"])
	}
	if v, ok := out.Metadata["seats"].(float64); !ok || v != 5 {
		t.Fatalf("seats: %v", out.Metadata["seats"])
	}
}

// --- Body marshal failure ---------------------------------------------

func TestHTTP_NonMarshalableBodyReturnsInvalidRequestError(t *testing.T) {
	// channels aren't JSON-marshalable; an internal request that
	// somehow gets one should surface as *InvalidRequestError.
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	err := c.http.request(bgCtx(), "/test", requestOptions{
		method: http.MethodPost,
		body:   map[string]any{"ch": make(chan int)},
	}, nil)
	var target *InvalidRequestError
	if !errors.As(err, &target) {
		t.Fatalf("got %T: %v", err, err)
	}
}

// --- Chain / Token constants ------------------------------------------

func TestChain_AllSupported(t *testing.T) {
	for _, c := range []ChainId{ChainBase, ChainEthereum, ChainPolygon, ChainArbitrum, ChainTron, ChainSolana} {
		if c == "" {
			t.Fatal("empty chain")
		}
	}
}

func TestToken_AllSupported(t *testing.T) {
	for _, tok := range []TokenSymbol{TokenUSDC, TokenUSDT} {
		if tok == "" {
			t.Fatal("empty token")
		}
	}
}

func TestPriceInterval_AllValues(t *testing.T) {
	want := map[PriceInterval]string{
		PriceOneTime: "one_time",
		PriceWeek:    "week",
		PriceMonth:   "month",
		PriceYear:    "year",
	}
	for k, v := range want {
		if string(k) != v {
			t.Errorf("got %q", k)
		}
	}
}

func TestPaymentStatus_AllValues(t *testing.T) {
	want := []string{"pending", "confirmed", "failed", "refunded", "reorged"}
	got := []PaymentStatus{PaymentPending, PaymentConfirmed, PaymentFailed, PaymentRefunded, PaymentReorged}
	for i, v := range got {
		if string(v) != want[i] {
			t.Errorf("got %q want %q", v, want[i])
		}
	}
}

func TestSubscriptionStatus_AllValues(t *testing.T) {
	want := []string{"trialing", "active", "past_due", "paused", "canceled"}
	got := []SubscriptionStatus{SubTrialing, SubActive, SubPastDue, SubPaused, SubCanceled}
	for i, v := range got {
		if string(v) != want[i] {
			t.Errorf("got %q want %q", v, want[i])
		}
	}
}

func TestInvoiceStatus_AllValues(t *testing.T) {
	want := []string{"draft", "open", "paid", "past_due", "void"}
	got := []InvoiceStatus{InvoiceDraft, InvoiceOpen, InvoicePaid, InvoicePastDue, InvoiceVoid}
	for i, v := range got {
		if string(v) != want[i] {
			t.Errorf("got %q want %q", v, want[i])
		}
	}
}

func TestCheckoutMode_AllValues(t *testing.T) {
	want := map[CheckoutMode]string{
		CheckoutPayment:      "payment",
		CheckoutSubscription: "subscription",
	}
	for k, v := range want {
		if string(k) != v {
			t.Errorf("got %q", k)
		}
	}
}

func TestCheckoutStatus_AllValues(t *testing.T) {
	got := []CheckoutStatus{CheckoutOpen, CheckoutPending, CheckoutSucceeded, CheckoutFailed, CheckoutExpired}
	want := []string{"open", "pending", "succeeded", "failed", "expired"}
	for i, v := range got {
		if string(v) != want[i] {
			t.Errorf("got %q want %q", v, want[i])
		}
	}
}

func TestAutopayMode_AllValues(t *testing.T) {
	want := []string{"allowance", "smart-wallet", "manual"}
	got := []AutopayMode{AutopayAllowance, AutopaySmartWallet, AutopayManual}
	for i, v := range got {
		if string(v) != want[i] {
			t.Errorf("got %q want %q", v, want[i])
		}
	}
}

func TestProrationMode_AllValues(t *testing.T) {
	want := map[ProrationMode]string{
		ProrationImmediately: "immediately",
		ProrationAtPeriodEnd: "at_period_end",
	}
	for k, v := range want {
		if string(k) != v {
			t.Error()
		}
	}
}

func TestCancelMode_AllValues(t *testing.T) {
	want := map[CancelMode]string{
		CancelImmediately: "immediately",
		CancelAtPeriodEnd: "at_period_end",
	}
	for k, v := range want {
		if string(k) != v {
			t.Error()
		}
	}
}

// --- Misc smoke ---------------------------------------------------------

func TestErrorString_Has_Code(t *testing.T) {
	for _, c := range []ErrorCode{CodeNotFound, CodeUnauthorized, CodeConflict} {
		err := FromEnvelope([]byte(`{"error":{"code":"`+string(c)+`","message":"x"}}`), 400, 0)
		if !strings.Contains(err.Error(), string(c)) {
			t.Errorf("%v not in %s", c, err.Error())
		}
	}
}

func TestRateLimitError_HasRetryAfter(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"rate_limited","message":"x"}}`), 429, 42)
	rl, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("got %T", err)
	}
	if rl.RetryAfter != 42 {
		t.Fatalf("got %v", rl.RetryAfter)
	}
}

func TestRateLimitError_Code(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"rate_limited","message":"x"}}`), 429, 0)
	rl := err.(*RateLimitError)
	if rl.Code != CodeRateLimited {
		t.Fatalf("got %s", rl.Code)
	}
}

func TestHTTP_LargeResponseBodyDecodes(t *testing.T) {
	// 500 customers in a single page — typical worst-case for List.
	var items []string
	for i := 0; i < 500; i++ {
		items = append(items, customerJSON)
	}
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+strings.Join(items, ",")+`],"nextCursor":""}`)
	out, err := c.Customers.List(bgCtx(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 500 {
		t.Fatalf("got %d", len(out.Data))
	}
}

func TestHTTP_RetryWithNetworkErrorThenSuccess(t *testing.T) {
	// Open server then close it; first attempt fails with a network
	// error, the server then comes back. We can't easily reopen so just
	// assert the retry happened by counting sleeps.
	s := newStubServer()
	s.queue(200, customerJSON) // never reached for first attempt
	c, err := NewClient("sk_test_x", "ws_x", WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	c.http.maxRetries = 1
	var slept int
	c.http.sleep = func(time.Duration) { slept++ }
	_, err = c.Customers.Retrieve(bgCtx(), "cu_1")
	if err == nil {
		t.Fatal("expected error")
	}
	if slept != 1 {
		t.Fatalf("expected one retry sleep, got %d", slept)
	}
	s.Close()
}

// --- Sanity: SDK version follows semver ish format --------------------

func TestVersion_Format(t *testing.T) {
	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		t.Fatalf("not three parts: %q", Version)
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			t.Fatalf("non-numeric segment %q in %q", p, Version)
		}
	}
}
