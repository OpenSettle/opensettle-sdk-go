package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// CustomersResource exposes /v1/workspaces/<ws>/customers.
type CustomersResource struct {
	http *httpClient
}

// List returns a cursor-paginated page of customers.
func (r *CustomersResource) List(ctx context.Context, query *ListCustomersQuery) (*CursorPage[Customer], error) {
	q := map[string]any{}
	if query != nil {
		if query.Cursor != "" {
			q["cursor"] = query.Cursor
		}
		if query.Limit > 0 {
			q["limit"] = query.Limit
		}
		if query.Status != "" {
			q["status"] = string(query.Status)
		}
		if query.Q != "" {
			q["q"] = query.Q
		}
	}
	out := &CursorPage[Customer]{}
	err := r.http.request(ctx, "/customers", requestOptions{query: q}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Retrieve fetches a customer by id.
func (r *CustomersResource) Retrieve(ctx context.Context, customerID string) (*Customer, error) {
	out := &Customer{}
	err := r.http.request(ctx, "/customers/"+url.PathEscape(customerID), requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Create makes a new customer. Auto-attaches an Idempotency-Key.
func (r *CustomersResource) Create(ctx context.Context, input CreateCustomerRequest) (*Customer, error) {
	out := &Customer{}
	err := r.http.request(ctx, "/customers", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Update applies a partial update.
func (r *CustomersResource) Update(ctx context.Context, customerID string, input UpdateCustomerRequest) (*Customer, error) {
	out := &Customer{}
	err := r.http.request(ctx, "/customers/"+url.PathEscape(customerID), requestOptions{
		method: http.MethodPatch,
		body:   input,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Delete is a soft-delete: PII is scrubbed but historical references
// continue to resolve.
func (r *CustomersResource) Delete(ctx context.Context, customerID string) error {
	return r.http.request(ctx, "/customers/"+url.PathEscape(customerID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}
