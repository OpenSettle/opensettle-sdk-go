package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// WebhookEndpointsResource exposes /v1/workspaces/<ws>/webhook_endpoints.
type WebhookEndpointsResource struct {
	http *httpClient
}

type endpointWrapper struct {
	Endpoint *WebhookEndpoint `json:"endpoint"`
}

// List returns every webhook endpoint configured for the workspace.
// The endpoint is small enough that the API doesn't paginate.
func (r *WebhookEndpointsResource) List(ctx context.Context) ([]WebhookEndpoint, error) {
	out := &rawList[WebhookEndpoint]{}
	err := r.http.request(ctx, "/webhook_endpoints", requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Retrieve fetches a single webhook endpoint by ID. The signing secret
// is never returned here — only Create and RotateSecret produce it.
func (r *WebhookEndpointsResource) Retrieve(ctx context.Context, endpointID string) (*WebhookEndpoint, error) {
	var w endpointWrapper
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{}, &w)
	if err != nil {
		return nil, err
	}
	return w.Endpoint, nil
}

// Create makes a new endpoint. The response includes the plaintext
// signing secret exactly once — store it immediately. Multi-key
// envelope {endpoint, signingSecret} preserved.
//
// Auto-attaches an Idempotency-Key; supply [WithIdempotencyKey] to
// override.
func (r *WebhookEndpointsResource) Create(ctx context.Context, input CreateWebhookEndpointRequest, opts ...RequestOption) (*CreateWebhookEndpointResponse, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	out := &CreateWebhookEndpointResponse{}
	err := r.http.request(ctx, "/webhook_endpoints", reqOpts, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Update patches a webhook endpoint. Fields left nil on input are
// unchanged. A non-nil Events replaces the allow-list entirely.
func (r *WebhookEndpointsResource) Update(ctx context.Context, endpointID string, input UpdateWebhookEndpointRequest) (*WebhookEndpoint, error) {
	var w endpointWrapper
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{
		method: http.MethodPatch,
		body:   input,
	}, &w)
	if err != nil {
		return nil, err
	}
	return w.Endpoint, nil
}

// Delete permanently removes a webhook endpoint. In-flight retries are
// dropped; consider Update with Status=disabled instead if you want to
// pause without losing the configuration.
func (r *WebhookEndpointsResource) Delete(ctx context.Context, endpointID string) error {
	return r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}

// RotateSecret rotates the signing secret. Returns the same
// {endpoint, signingSecret} envelope as Create — store SigningSecret
// immediately. The endpoint takes no request body.
//
// There is NO grace window: the previous secret stops verifying
// immediately, so deploy the new secret before (or atomically with)
// calling this. Step-up auth (AAL=2) required; API-key callers receive
// *StepUpRequiredError. Auto-attaches an Idempotency-Key; supply
// [WithIdempotencyKey] to override.
func (r *WebhookEndpointsResource) RotateSecret(ctx context.Context, endpointID string, opts ...RequestOption) (*CreateWebhookEndpointResponse, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	out := &CreateWebhookEndpointResponse{}
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID)+"/rotate", reqOpts, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Test emits a sample event for the endpoint to verify wiring. It takes no
// request body — the server generates the sample payload and fans it out
// asynchronously through the normal delivery + retry pipeline. The returned
// EventID identifies the emitted event; poll the events / deliveries
// endpoints to observe whether your endpoint accepted it.
func (r *WebhookEndpointsResource) Test(ctx context.Context, endpointID string) (*TestWebhookEndpointResponse, error) {
	out := &TestWebhookEndpointResponse{}
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID)+"/test", requestOptions{
		method: http.MethodPost,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
