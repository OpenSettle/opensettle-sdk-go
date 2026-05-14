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

type checkoutWrapper struct {
	Checkout *Checkout `json:"checkout"`
}

// Create starts a hosted checkout session. Body is required; the request
// is sent with an auto-generated Idempotency-Key to make retries safe.
// Supply [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *CheckoutsResource) Create(ctx context.Context, input CreateCheckoutRequest, opts ...RequestOption) (*Checkout, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w checkoutWrapper
	err := r.http.request(ctx, "/checkouts", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Checkout, nil
}

// Retrieve fetches a checkout session by id.
func (r *CheckoutsResource) Retrieve(ctx context.Context, id string) (*Checkout, error) {
	var w checkoutWrapper
	err := r.http.request(ctx, "/checkouts/"+url.PathEscape(id), requestOptions{}, &w)
	if err != nil {
		return nil, err
	}
	return w.Checkout, nil
}
