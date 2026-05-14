package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// PaymentsResource exposes /v1/workspaces/<ws>/payments.
type PaymentsResource struct {
	http *httpClient
}

type paymentWrapper struct {
	Payment *Payment `json:"payment"`
}

// List returns one page of payments for the workspace. Pass nil for an
// unfiltered first page; use ListIter for cursor-driven full iteration.
func (r *PaymentsResource) List(ctx context.Context, query *ListPaymentsQuery) (*CursorPage[Payment], error) {
	q := map[string]any{}
	if query != nil {
		if query.Cursor != "" {
			q["cursor"] = query.Cursor
		}
		if query.Limit > 0 {
			q["limit"] = query.Limit
		}
		if query.CustomerID != "" {
			q["customerId"] = query.CustomerID
		}
		if query.Status != "" {
			q["status"] = string(query.Status)
		}
	}
	out := &CursorPage[Payment]{}
	err := r.http.request(ctx, "/payments", requestOptions{query: q}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Retrieve fetches a single payment by ID.
func (r *PaymentsResource) Retrieve(ctx context.Context, paymentID string) (*Payment, error) {
	var w paymentWrapper
	err := r.http.request(ctx, "/payments/"+url.PathEscape(paymentID), requestOptions{}, &w)
	if err != nil {
		return nil, err
	}
	return w.Payment, nil
}

// Refund initiates a refund. Returns a multi-key envelope
// {payment, unsignedTx} — the merchant's wallet signs unsignedTx and
// broadcasts it; OpenSettle never holds funds.
//
// Step-up auth (AAL=2) is required on this route, surfaced as
// *StepUpRequiredError for API-key callers. Sessions get through if
// they re-authed within freshWithinSeconds.
//
// Auto-attaches an Idempotency-Key; supply [WithIdempotencyKey] to use a
// caller-chosen key (recommended for refund flows where the caller has a
// natural deterministic id, e.g. the refund's row id in your DB).
func (r *PaymentsResource) Refund(ctx context.Context, paymentID string, input InitiateRefundRequest, opts ...RequestOption) (*InitiateRefundResponse, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	out := &InitiateRefundResponse{}
	err := r.http.request(ctx, "/payments/"+url.PathEscape(paymentID)+"/refund", reqOpts, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RefundBroadcast tells OpenSettle the merchant has signed and
// broadcast the refund tx. The chain reader picks it up and flips
// status to refunded once it confirms. Auto-attaches an Idempotency-Key;
// supply [WithIdempotencyKey] to override.
func (r *PaymentsResource) RefundBroadcast(ctx context.Context, paymentID string, input RecordRefundBroadcastRequest, opts ...RequestOption) (*Payment, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w paymentWrapper
	err := r.http.request(ctx, "/payments/"+url.PathEscape(paymentID)+"/refund/broadcast", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Payment, nil
}

// ListIter returns a cursor-driven iterator over all payments matching
// the query.
func (r *PaymentsResource) ListIter(ctx context.Context, query *ListPaymentsQuery) *Iter[Payment] {
	return newIter(ctx, func(ctx context.Context, cursor string) (*CursorPage[Payment], error) {
		q := ListPaymentsQuery{}
		if query != nil {
			q = *query
		}
		if cursor != "" {
			q.Cursor = cursor
		}
		return r.List(ctx, &q)
	})
}
