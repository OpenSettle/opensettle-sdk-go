package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// SubscriptionsResource exposes /v1/workspaces/<ws>/subscriptions.
type SubscriptionsResource struct {
	http *httpClient
}

type subscriptionWrapper struct {
	Subscription *Subscription `json:"subscription"`
}

// List returns one page of subscriptions. Pass nil for an unfiltered
// first page; use ListIter for cursor-driven full iteration.
func (r *SubscriptionsResource) List(ctx context.Context, query *ListSubscriptionsQuery) (*CursorPage[Subscription], error) {
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
	out := &CursorPage[Subscription]{}
	err := r.http.request(ctx, "/subscriptions", requestOptions{query: q}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Retrieve fetches a single subscription by ID.
func (r *SubscriptionsResource) Retrieve(ctx context.Context, subID string) (*Subscription, error) {
	var w subscriptionWrapper
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID), requestOptions{}, &w)
	if err != nil {
		return nil, err
	}
	return w.Subscription, nil
}

// Create starts a new subscription. Auto-attaches an Idempotency-Key;
// supply [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *SubscriptionsResource) Create(ctx context.Context, input CreateSubscriptionRequest, opts ...RequestOption) (*Subscription, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w subscriptionWrapper
	err := r.http.request(ctx, "/subscriptions", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Subscription, nil
}

// Pause stops billing on a subscription without canceling it. Resume to
// continue. No proration; the next billing date shifts forward by the
// paused interval. Auto-attaches an Idempotency-Key (the endpoint requires
// one); supply [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *SubscriptionsResource) Pause(ctx context.Context, subID string, opts ...RequestOption) (*Subscription, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w subscriptionWrapper
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/pause", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Subscription, nil
}

// Resume reactivates a paused subscription. Billing restarts from the
// shifted next-billing date set when the subscription was paused.
// Auto-attaches an Idempotency-Key (the endpoint requires one); supply
// [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *SubscriptionsResource) Resume(ctx context.Context, subID string, opts ...RequestOption) (*Subscription, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w subscriptionWrapper
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/resume", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Subscription, nil
}

// Cancel ends a subscription. Mode controls timing (immediately vs
// at_period_end). Default is at_period_end when input.Mode is empty.
// Reason is recorded on the audit log. Auto-attaches an Idempotency-Key
// (the endpoint requires one); supply [WithIdempotencyKey] to use a
// caller-chosen key instead.
func (r *SubscriptionsResource) Cancel(ctx context.Context, subID string, input CancelSubscriptionRequest, opts ...RequestOption) (*Subscription, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w subscriptionWrapper
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/cancel", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Subscription, nil
}

// ChangePlan swaps the subscription to a new price. ProrationMode
// controls whether the customer is billed immediately for the delta or
// at the next period boundary. Auto-attaches an Idempotency-Key; supply
// [WithIdempotencyKey] to override.
func (r *SubscriptionsResource) ChangePlan(ctx context.Context, subID string, input ChangePlanRequest, opts ...RequestOption) (*Subscription, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w subscriptionWrapper
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/change_plan", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Subscription, nil
}

// ListIter returns a cursor-driven iterator over all subscriptions
// matching the query.
func (r *SubscriptionsResource) ListIter(ctx context.Context, query *ListSubscriptionsQuery) *Iter[Subscription] {
	return newIter(ctx, func(ctx context.Context, cursor string) (*CursorPage[Subscription], error) {
		q := ListSubscriptionsQuery{}
		if query != nil {
			q = *query
		}
		if cursor != "" {
			q.Cursor = cursor
		}
		return r.List(ctx, &q)
	})
}
