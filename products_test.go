package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const productJSON = `{"id":"prod_1","workspaceId":"ws","name":"Pro","description":null,"active":true,"metadata":null,"createdAt":"2026-05-12T15:00:00.000Z"}`
const priceJSON = `{"id":"price_1","workspaceId":"ws","productId":"prod_1","amount":1000,"currency":"USD","interval":"month","active":true,"metadata":null,"createdAt":"2026-05-12T15:00:00.000Z"}`
const productWrappedJSON = `{"product":` + productJSON + `}`
const priceWrappedJSON = `{"price":` + priceJSON + `}`

func TestProducts_List(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+productJSON+`],"nextCursor":""}`)
	out, err := c.Products.List(bgCtx(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 {
		t.Fatal()
	}
}

func TestProducts_List_ActiveTrue(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	active := true
	_, _ = c.Products.List(bgCtx(), &ListProductsQuery{Active: &active})
	q := s.lastRequest(t).Query
	if !strings.Contains(q, "active=true") {
		t.Fatalf("got %q", q)
	}
}

func TestProducts_List_ActiveFalse(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[],"nextCursor":""}`)
	active := false
	_, _ = c.Products.List(bgCtx(), &ListProductsQuery{Active: &active})
	q := s.lastRequest(t).Query
	if !strings.Contains(q, "active=false") {
		t.Fatalf("got %q", q)
	}
}

func TestProducts_Retrieve(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, productWrappedJSON)
	out, err := c.Products.Retrieve(bgCtx(), "prod_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Pro" {
		t.Fatalf("name: %s", out.Name)
	}
}

func TestProducts_Create(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, productWrappedJSON)
	_, err := c.Products.Create(bgCtx(), CreateProductRequest{Name: "Pro"})
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

func TestProducts_Update(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, productWrappedJSON)
	name := "Pro+"
	_, err := c.Products.Update(bgCtx(), "prod_1", UpdateProductRequest{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodPatch {
		t.Fatalf("method: %s", r.Method)
	}
}

func TestProducts_Delete(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(204, "")
	if err := c.Products.Delete(bgCtx(), "prod_1"); err != nil {
		t.Fatal(err)
	}
}

func TestProducts_Delete_ConflictWhenInUse(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(409, `{"error":{"code":"conflict","message":"subscriptions still reference this"}}`)
	err := c.Products.Delete(bgCtx(), "prod_1")
	var target *ConflictError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestProducts_ListPrices(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+priceJSON+`]}`)
	out, err := c.Products.ListPrices(bgCtx(), "prod_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "price_1" {
		t.Fatalf("got %+v", out)
	}
	if got := s.lastRequest(t).Path; got != "/v1/workspaces/ws_test/products/prod_1/prices" {
		t.Fatalf("path: %s", got)
	}
}

func TestProducts_CreatePrice(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, priceWrappedJSON)
	out, err := c.Products.CreatePrice(bgCtx(), "prod_1", CreatePriceRequest{Amount: 1000, Interval: PriceMonth})
	if err != nil {
		t.Fatal(err)
	}
	if out.Interval != PriceMonth {
		t.Fatalf("interval: %s", out.Interval)
	}
	if got := s.lastRequest(t).Headers.Get("Idempotency-Key"); got == "" {
		t.Fatalf("missing key")
	}
}

func TestProducts_CreatePrice_BodySerialization(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, priceWrappedJSON)
	_, _ = c.Products.CreatePrice(bgCtx(), "prod_1", CreatePriceRequest{Amount: 2500, Currency: "EUR", Interval: PriceYear})
	body := decodeBody[map[string]any](t, s.lastRequest(t).Body)
	if body["amount"].(float64) != 2500 {
		t.Fatalf("amount: %v", body["amount"])
	}
	if body["currency"] != "EUR" {
		t.Fatalf("currency: %v", body["currency"])
	}
	if body["interval"] != "year" {
		t.Fatalf("interval: %v", body["interval"])
	}
}

func TestProducts_DeletePrice(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(204, "")
	if err := c.Products.DeletePrice(bgCtx(), "price_1"); err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodDelete {
		t.Fatalf("method: %s", r.Method)
	}
	if r.Path != "/v1/workspaces/ws_test/prices/price_1" {
		t.Fatalf("path: %s", r.Path)
	}
}

func TestProducts_DeletePrice_Conflict(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(409, `{"error":{"code":"conflict","message":"in use"}}`)
	err := c.Products.DeletePrice(bgCtx(), "price_1")
	var target *ConflictError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}
