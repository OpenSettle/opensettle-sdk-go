package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const customerJSON = `{"id":"cu_1","workspaceId":"ws","email":"a@b","name":"A B","wallet":null,"country":null,"status":"active","activeSubscriptions":0,"lifetimeValue":0,"lifetimeValueMinor":12345,"metadata":null,"createdAt":"2026-05-12T15:00:00.000Z","deletedAt":null}`
const customerWrappedJSON = `{"customer":` + customerJSON + `}`

func TestCustomers_List_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+customerJSON+`],"nextCursor":"cur_2","hasMore":true}`)
	out, err := c.Customers.List(bgCtx(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 || out.Data[0].ID != "cu_1" {
		t.Fatalf("got %+v", out)
	}
	if out.NextCursor != "cur_2" {
		t.Fatalf("cursor: %s", out.NextCursor)
	}
	if !out.HasMore {
		t.Fatalf("hasMore")
	}
}

func TestCustomers_List_Query(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	_, _ = c.Customers.List(bgCtx(), &ListCustomersQuery{
		Status: CustomerAtRisk,
		Q:      "alice",
		Cursor: "cur",
		Limit:  50,
	})
	got := s.lastRequest(t).Query
	for _, want := range []string{"status=at_risk", "q=alice", "cursor=cur", "limit=50"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestCustomers_Retrieve_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, customerWrappedJSON)
	out, err := c.Customers.Retrieve(bgCtx(), "cu_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "cu_1" {
		t.Fatalf("id: %s", out.ID)
	}
	// LifetimeValueMinor carries the live-computed LTV; the legacy
	// LifetimeValue is the always-0 stored cache.
	if out.LifetimeValueMinor != 12345 {
		t.Errorf("lifetimeValueMinor: %d", out.LifetimeValueMinor)
	}
	if out.LifetimeValue != 0 {
		t.Errorf("lifetimeValue (legacy, want 0): %d", out.LifetimeValue)
	}
}

func TestCustomers_Retrieve_NotFound(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(404, `{"error":{"code":"not_found","message":"x"}}`)
	_, err := c.Customers.Retrieve(bgCtx(), "cu_missing")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("got %T", err)
	}
}

func TestCustomers_Create_HappyPath(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, customerWrappedJSON)
	out, err := c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "a@b", Name: "A B"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "cu_1" {
		t.Fatalf("id: %s", out.ID)
	}
	if r := s.lastRequest(t); r.Method != http.MethodPost {
		t.Fatalf("method: %s", r.Method)
	}
}

func TestCustomers_Create_AttachesIdempotencyKey(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, customerWrappedJSON)
	_, _ = c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "a@b"})
	if k := s.lastRequest(t).Headers.Get("Idempotency-Key"); k == "" {
		t.Fatalf("missing key")
	}
}

func TestCustomers_Update_PATCH(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, customerWrappedJSON)
	name := "New Name"
	_, err := c.Customers.Update(bgCtx(), "cu_1", UpdateCustomerRequest{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodPatch {
		t.Fatalf("method: %s", r.Method)
	}
	body := decodeBody[map[string]any](t, r.Body)
	if body["name"] != "New Name" {
		t.Fatalf("body: %+v", body)
	}
}

func TestCustomers_Update_NoIdempotencyKey(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, customerWrappedJSON)
	name := "X"
	_, _ = c.Customers.Update(bgCtx(), "cu_1", UpdateCustomerRequest{Name: &name})
	if k := s.lastRequest(t).Headers.Get("Idempotency-Key"); k != "" {
		t.Fatalf("should not have idempotency key: %q", k)
	}
}

func TestCustomers_Delete_DELETE(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(204, "")
	if err := c.Customers.Delete(bgCtx(), "cu_1"); err != nil {
		t.Fatal(err)
	}
	if r := s.lastRequest(t); r.Method != http.MethodDelete {
		t.Fatalf("method: %s", r.Method)
	}
}

func TestCustomers_Delete_Conflict(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(409, `{"error":{"code":"conflict","message":"has subs"}}`)
	err := c.Customers.Delete(bgCtx(), "cu_1")
	var target *ConflictError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestCustomers_Create_ValidationErrorPropagates(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(400, `{"error":{"code":"invalid_request","message":"bad email","param":"email"}}`)
	_, err := c.Customers.Create(bgCtx(), CreateCustomerRequest{Email: "not-an-email"})
	var target *InvalidRequestError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Param != "email" {
		t.Fatalf("param: %q", target.Param)
	}
}

func TestCustomers_Path(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, customerWrappedJSON)
	_, _ = c.Customers.Retrieve(bgCtx(), "cu_1")
	if got := s.lastRequest(t).Path; got != "/v1/workspaces/ws_test/customers/cu_1" {
		t.Fatalf("path: %s", got)
	}
}
