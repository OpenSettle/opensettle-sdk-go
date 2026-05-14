package opensettle

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func recordedQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query %q: %v", raw, err)
	}
	return v
}

func allRecorded(s *stubServer) []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

func TestCustomers_ListIter_SinglePage(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+customerJSON+`],"nextCursor":"","hasMore":false}`)

	it := c.Customers.ListIter(bgCtx(), nil)
	count := 0
	for it.Next() {
		if it.Item().ID != "cu_1" {
			t.Fatalf("id: %s", it.Item().ID)
		}
		count++
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if count != 1 {
		t.Fatalf("count: %d", count)
	}
}

func TestCustomers_ListIter_FollowsNextCursor(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	// Page 1: 1 item, hasMore=true.
	s.queue(200, `{"data":[`+customerJSON+`],"nextCursor":"cur_2","hasMore":true}`)
	// Page 2: 1 more item, hasMore=false.
	otherCustomer := strings.Replace(customerJSON, `"id":"cu_1"`, `"id":"cu_2"`, 1)
	s.queue(200, `{"data":[`+otherCustomer+`],"nextCursor":"","hasMore":false}`)

	it := c.Customers.ListIter(bgCtx(), nil)
	ids := []string{}
	for it.Next() {
		ids = append(ids, it.Item().ID)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(ids) != 2 || ids[0] != "cu_1" || ids[1] != "cu_2" {
		t.Fatalf("ids: %v", ids)
	}
}

func TestCustomers_ListIter_PassesCursorOnSubsequentCalls(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+customerJSON+`],"nextCursor":"cur_x","hasMore":true}`)
	s.queue(200, `{"data":[],"nextCursor":"","hasMore":false}`)

	it := c.Customers.ListIter(bgCtx(), nil)
	for it.Next() {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	reqs := allRecorded(s)
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}
	q := recordedQuery(t, reqs[1].Query)
	if q.Get("cursor") != "cur_x" {
		t.Fatalf("cursor on 2nd req: %q", q.Get("cursor"))
	}
}

func TestCustomers_ListIter_HasMoreFalseStops(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	// nextCursor present but hasMore=false → iterator stops anyway.
	s.queue(200, `{"data":[`+customerJSON+`],"nextCursor":"ignored","hasMore":false}`)
	it := c.Customers.ListIter(bgCtx(), nil)
	count := 0
	for it.Next() {
		count++
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if count != 1 {
		t.Fatalf("count: %d", count)
	}
}

func TestCustomers_ListIter_PropagatesErrors(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(500, `{"error":{"code":"internal_error","message":"x"}}`)
	c.http.maxRetries = 0
	it := c.Customers.ListIter(bgCtx(), nil)
	if it.Next() {
		t.Fatalf("expected Next to return false on error")
	}
	if it.Err() == nil {
		t.Fatalf("expected an error from Err()")
	}
	var apiErr *APIError
	if !errors.As(it.Err(), &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", it.Err(), it.Err())
	}
}

func TestCustomers_ListIter_PassesFiltersThroughEveryCall(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+customerJSON+`],"nextCursor":"cur_x","hasMore":true}`)
	s.queue(200, `{"data":[],"nextCursor":"","hasMore":false}`)

	it := c.Customers.ListIter(bgCtx(), &ListCustomersQuery{Status: CustomerActive, Limit: 10})
	for it.Next() {
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	reqs := allRecorded(s)
	q0 := recordedQuery(t, reqs[0].Query)
	q1 := recordedQuery(t, reqs[1].Query)
	if q0.Get("status") != "active" {
		t.Fatalf("status on 1st: %q", q0.Get("status"))
	}
	if q1.Get("status") != "active" {
		t.Fatalf("status on 2nd: %q", q1.Get("status"))
	}
	if q1.Get("limit") != "10" {
		t.Fatalf("limit on 2nd: %q", q1.Get("limit"))
	}
}

func TestProducts_ListIter_BasicShape(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+productJSON+`],"nextCursor":"","hasMore":false}`)
	it := c.Products.ListIter(bgCtx(), nil)
	if !it.Next() {
		t.Fatalf("expected 1 item")
	}
	if it.Item().ID != "prod_1" {
		t.Fatalf("id: %s", it.Item().ID)
	}
	if it.Next() {
		t.Fatalf("expected end of iteration")
	}
}

func TestSubscriptions_ListIter_BasicShape(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+subscriptionJSON+`],"nextCursor":"","hasMore":false}`)
	it := c.Subscriptions.ListIter(bgCtx(), nil)
	if !it.Next() {
		t.Fatalf("expected 1 item")
	}
	if it.Item().ID != "sub_1" {
		t.Fatalf("id: %s", it.Item().ID)
	}
}

func TestPayments_ListIter_BasicShape(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+paymentJSON+`],"nextCursor":"","hasMore":false}`)
	it := c.Payments.ListIter(bgCtx(), nil)
	if !it.Next() {
		t.Fatalf("expected 1 item")
	}
	if it.Item().ID != "pay_1" {
		t.Fatalf("id: %s", it.Item().ID)
	}
	if it.Next() {
		t.Fatalf("expected end of iteration")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestPayments_ListIter_FollowsNextCursor(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+paymentJSON+`],"nextCursor":"cur_2","hasMore":true}`)
	other := strings.Replace(paymentJSON, `"id":"pay_1"`, `"id":"pay_2"`, 1)
	s.queue(200, `{"data":[`+other+`],"nextCursor":"","hasMore":false}`)

	it := c.Payments.ListIter(bgCtx(), &ListPaymentsQuery{CustomerID: "cu_1", Limit: 5})
	ids := []string{}
	for it.Next() {
		ids = append(ids, it.Item().ID)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(ids) != 2 || ids[0] != "pay_1" || ids[1] != "pay_2" {
		t.Fatalf("ids: %v", ids)
	}
	reqs := allRecorded(s)
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}
	q1 := recordedQuery(t, reqs[1].Query)
	if q1.Get("cursor") != "cur_2" {
		t.Fatalf("cursor on 2nd: %q", q1.Get("cursor"))
	}
	if q1.Get("customerId") != "cu_1" {
		t.Fatalf("customerId on 2nd: %q", q1.Get("customerId"))
	}
}

func TestInvoices_ListIter_BasicShape(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+invoiceJSON+`],"nextCursor":"","hasMore":false}`)
	it := c.Invoices.ListIter(bgCtx(), nil)
	if !it.Next() {
		t.Fatalf("expected 1 item")
	}
	if it.Item().ID != "in_1" {
		t.Fatalf("id: %s", it.Item().ID)
	}
	if it.Next() {
		t.Fatalf("expected end of iteration")
	}
}

func TestInvoices_ListIter_FollowsNextCursor(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+invoiceJSON+`],"nextCursor":"cur_2","hasMore":true}`)
	other := strings.Replace(invoiceJSON, `"id":"in_1"`, `"id":"in_2"`, 1)
	s.queue(200, `{"data":[`+other+`],"nextCursor":"","hasMore":false}`)

	it := c.Invoices.ListIter(bgCtx(), nil)
	ids := []string{}
	for it.Next() {
		ids = append(ids, it.Item().ID)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 items, got %d", len(ids))
	}
}
