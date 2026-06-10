package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// PaymentLinksResource exposes /v1/workspaces/<ws>/payment_links.
//
// A payment link is a reusable, scan-and-pay hosted page (unlike a
// single-use Checkout). Create one with a fixed amount, a one-time Price,
// or an open "name your price" amount; share its absolute URL; many guests
// can pay it until you Deactivate it.
type PaymentLinksResource struct {
	http *httpClient
}

type paymentLinkWrapper struct {
	PaymentLink *PaymentLink `json:"paymentLink"`
}

// Create makes a new payment link. Supply exactly ONE amount source on the
// request (Amount, PriceID, or OpenAmount=true). The response is the
// {paymentLink} envelope, unwrapped to the link.
//
// Auto-attaches an Idempotency-Key (retries are safe); supply
// [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *PaymentLinksResource) Create(ctx context.Context, input CreatePaymentLinkRequest, opts ...RequestOption) (*PaymentLink, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w paymentLinkWrapper
	err := r.http.request(ctx, "/payment_links", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.PaymentLink, nil
}

// List returns every payment link configured for the workspace. The
// collection is small enough that the API doesn't paginate it — the
// response is a flat { "data": [...] } envelope.
func (r *PaymentLinksResource) List(ctx context.Context) ([]PaymentLink, error) {
	out := &rawList[PaymentLink]{}
	err := r.http.request(ctx, "/payment_links", requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Deactivate disables a payment link so it can no longer be paid. Existing
// payments already made through it are unaffected. The server responds with
// { "ok": true }; this method returns only an error.
func (r *PaymentLinksResource) Deactivate(ctx context.Context, paymentLinkID string) error {
	return r.http.request(ctx, "/payment_links/"+url.PathEscape(paymentLinkID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}
