package opensettle

import (
	"errors"
	"net/http"
	"testing"
)

const endpointJSON = `{"id":"we_1","workspaceId":"ws","url":"https://example/webhook","description":null,"events":["*"],"status":"enabled","successRate":1.0,"rotationGraceUntil":null,"createdAt":"2026-05-12T15:00:00.000Z"}`
const endpointWrappedJSON = `{"endpoint":` + endpointJSON + `}`
const createEndpointResponseJSON = `{"endpoint":` + endpointJSON + `,"signingSecret":"whsec_test_abc"}`

func TestWebhookEndpoints_List(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"data":[`+endpointJSON+`]}`)
	out, err := c.WebhookEndpoints.List(bgCtx())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "we_1" {
		t.Fatalf("got %+v", out)
	}
}

func TestWebhookEndpoints_Retrieve(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, endpointWrappedJSON)
	out, err := c.WebhookEndpoints.Retrieve(bgCtx(), "we_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != WebhookEnabled {
		t.Fatalf("status: %s", out.Status)
	}
}

func TestWebhookEndpoints_Create(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, createEndpointResponseJSON)
	out, err := c.WebhookEndpoints.Create(bgCtx(), CreateWebhookEndpointRequest{
		URL:    "https://example/webhook",
		Events: []string{"payment.confirmed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.SigningSecret != "whsec_test_abc" {
		t.Fatalf("secret: %s", out.SigningSecret)
	}
	if out.Endpoint.ID != "we_1" {
		t.Fatalf("endpoint id: %s", out.Endpoint.ID)
	}
	if got := s.lastRequest(t).Headers.Get("Idempotency-Key"); got == "" {
		t.Fatalf("missing key")
	}
}

func TestWebhookEndpoints_Create_InvalidURL(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(400, `{"error":{"code":"invalid_request","message":"must be https","param":"url"}}`)
	_, err := c.WebhookEndpoints.Create(bgCtx(), CreateWebhookEndpointRequest{URL: "http://insecure"})
	var target *InvalidRequestError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Param != "url" {
		t.Fatalf("param: %q", target.Param)
	}
}

func TestWebhookEndpoints_Update(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, endpointWrappedJSON)
	status := WebhookDisabled
	_, err := c.WebhookEndpoints.Update(bgCtx(), "we_1", UpdateWebhookEndpointRequest{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	r := s.lastRequest(t)
	if r.Method != http.MethodPatch {
		t.Fatalf("method: %s", r.Method)
	}
	body := decodeBody[map[string]any](t, r.Body)
	if body["status"] != "disabled" {
		t.Fatalf("status: %v", body["status"])
	}
}

func TestWebhookEndpoints_Delete(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(204, "")
	if err := c.WebhookEndpoints.Delete(bgCtx(), "we_1"); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookEndpoints_RotateSecret(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"endpoint":`+endpointJSON+`,"signingSecret":"whsec_new"}`)
	out, err := c.WebhookEndpoints.RotateSecret(bgCtx(), "we_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.SigningSecret != "whsec_new" {
		t.Fatalf("signingSecret: %s", out.SigningSecret)
	}
	if out.Endpoint.ID != "we_1" {
		t.Fatalf("endpoint.id: %s", out.Endpoint.ID)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/webhook_endpoints/we_1/rotate" {
		t.Fatalf("path: %s", r.Path)
	}
	if r.Headers.Get("Idempotency-Key") == "" {
		t.Fatalf("missing key")
	}
	// /rotate takes no request body — there is no grace window.
	if len(r.Body) != 0 {
		t.Fatalf("expected empty body, got: %q", string(r.Body))
	}
}

func TestWebhookEndpoints_Test(t *testing.T) {
	s := newStubServer()
	defer s.Close()
	c := newTestClient(t, s)
	s.queue(200, `{"eventId":"evt_123"}`)
	out, err := c.WebhookEndpoints.Test(bgCtx(), "we_1")
	if err != nil {
		t.Fatal(err)
	}
	if out.EventID != "evt_123" {
		t.Fatalf("eventId: %s", out.EventID)
	}
	r := s.lastRequest(t)
	if r.Path != "/v1/workspaces/ws_test/webhook_endpoints/we_1/test" {
		t.Fatalf("path: %s", r.Path)
	}
	// /test takes no request body — the server generates the sample payload.
	if len(r.Body) != 0 {
		t.Fatalf("expected empty body, got: %q", string(r.Body))
	}
}
