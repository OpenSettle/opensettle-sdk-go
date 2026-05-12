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
	out := &WebhookEndpoint{}
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Create makes a new endpoint. The response includes the plaintext
// signing secret exactly once — store it immediately.
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
	out := &WebhookEndpoint{}
	err := r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{
		method: http.MethodPatch,
		body:   input,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *WebhookEndpointsResource) Delete(ctx context.Context, endpointID string) error {
	return r.http.request(ctx, "/webhook_endpoints/"+url.PathEscape(endpointID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}

// RotateSecret rotates the signing secret. The returned secret is the
// new plaintext; the previous secret remains valid until
// RotationGraceUntil so consumers can roll without downtime.
func (r *WebhookEndpointsResource) RotateSecret(ctx context.Context, endpointID string, input RotateWebhookSecretRequest) (*RotateWebhookSecretResponse, error) {
	out := &RotateWebhookSecretResponse{}
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
