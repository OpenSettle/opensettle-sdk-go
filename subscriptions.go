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

func (r *SubscriptionsResource) Retrieve(ctx context.Context, subID string) (*Subscription, error) {
	out := &Subscription{}
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID), requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SubscriptionsResource) Create(ctx context.Context, input CreateSubscriptionRequest) (*Subscription, error) {
	out := &Subscription{}
	err := r.http.request(ctx, "/subscriptions", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SubscriptionsResource) Pause(ctx context.Context, subID string) (*Subscription, error) {
	out := &Subscription{}
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/pause", requestOptions{
		method: http.MethodPost,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SubscriptionsResource) Resume(ctx context.Context, subID string) (*Subscription, error) {
	out := &Subscription{}
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/resume", requestOptions{
		method: http.MethodPost,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Cancel ends a subscription. Mode controls timing (immediately vs
// at_period_end). Default is at_period_end when input.Mode is empty.
// Reason is recorded on the audit log.
func (r *SubscriptionsResource) Cancel(ctx context.Context, subID string, input CancelSubscriptionRequest) (*Subscription, error) {
	out := &Subscription{}
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/cancel", requestOptions{
		method: http.MethodPost,
		body:   input,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ChangePlan swaps the subscription to a new price. ProrationMode
// controls whether the customer is billed immediately for the delta or
// at the next period boundary.
func (r *SubscriptionsResource) ChangePlan(ctx context.Context, subID string, input ChangePlanRequest) (*Subscription, error) {
	out := &Subscription{}
	err := r.http.request(ctx, "/subscriptions/"+url.PathEscape(subID)+"/change_plan", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
