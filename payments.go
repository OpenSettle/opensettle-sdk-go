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
func (r *PaymentsResource) Refund(ctx context.Context, paymentID string, input InitiateRefundRequest) (*InitiateRefundResponse, error) {
	out := &InitiateRefundResponse{}
	err := r.http.request(ctx, "/payments/"+url.PathEscape(paymentID)+"/refund", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RefundBroadcast tells OpenSettle the merchant has signed and
// broadcast the refund tx. The chain reader picks it up and flips
// status to refunded once it confirms.
func (r *PaymentsResource) RefundBroadcast(ctx context.Context, paymentID string, input RecordRefundBroadcastRequest) (*Payment, error) {
	var w paymentWrapper
	err := r.http.request(ctx, "/payments/"+url.PathEscape(paymentID)+"/refund/broadcast", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, &w)
	if err != nil {
		return nil, err
	}
	return w.Payment, nil
}
