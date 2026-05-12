package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// CheckoutsResource exposes /v1/workspaces/<ws>/checkouts.
type CheckoutsResource struct {
	http *httpClient
}

// Create starts a hosted checkout session. Body is required; the request
// is sent with an auto-generated Idempotency-Key to make retries safe.
func (r *CheckoutsResource) Create(ctx context.Context, input CreateCheckoutRequest) (*Checkout, error) {
	out := &Checkout{}
	err := r.http.request(ctx, "/checkouts", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Retrieve fetches a checkout session by id.
func (r *CheckoutsResource) Retrieve(ctx context.Context, id string) (*Checkout, error) {
	out := &Checkout{}
	err := r.http.request(ctx, "/checkouts/"+url.PathEscape(id), requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
