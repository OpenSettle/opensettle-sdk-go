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
func (r *WebhookEndpointsResource) Create(ctx context.Context, input CreateWebhookEndpointRequest) (*CreateWebhookEndpointResponse, error) {
	out := &CreateWebhookEndpointResponse{}
	err := r.http.request(ctx, "/webhook_endpoints", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

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

func (r *WebhookEndpointsResource) Delete(ctx context.Context, endpointID string) error {
	return r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}

// RotateSecret rotates the signing secret. Returns the same
// {endpoint, signingSecret} envelope as Create — store SigningSecret
// immediately. Step-up auth (AAL=2) required; API-key callers receive
// *StepUpRequiredError.
func (r *WebhookEndpointsResource) RotateSecret(ctx context.Context, endpointID string, input RotateWebhookSecretRequest) (*CreateWebhookEndpointResponse, error) {
	out := &CreateWebhookEndpointResponse{}
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID)+"/rotate", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Test fires a sample event at the endpoint synchronously to verify
// wiring. The returned status is the response status the endpoint
// returned to OpenSettle.
func (r *WebhookEndpointsResource) Test(ctx context.Context, endpointID string, input TestWebhookEndpointRequest) (*TestWebhookEndpointResponse, error) {
	out := &TestWebhookEndpointResponse{}
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID)+"/test", requestOptions{
		method: http.MethodPost,
		body:   input,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
