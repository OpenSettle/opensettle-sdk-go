package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// InvoicesResource exposes /v1/workspaces/<ws>/invoices.
type InvoicesResource struct {
	http *httpClient
}

type invoiceWrapper struct {
	Invoice *Invoice `json:"invoice"`
}

// List returns one page of invoices for the workspace. Pass nil for an
// unfiltered first page; use ListIter for cursor-driven full iteration.
func (r *InvoicesResource) List(ctx context.Context, query *ListInvoicesQuery) (*CursorPage[Invoice], error) {
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
		if query.From != "" {
			q["from"] = query.From
		}
		if query.To != "" {
			q["to"] = query.To
		}
	}
	out := &CursorPage[Invoice]{}
	err := r.http.request(ctx, "/invoices", requestOptions{query: q}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Retrieve fetches a single invoice by ID.
func (r *InvoicesResource) Retrieve(ctx context.Context, invoiceID string) (*Invoice, error) {
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID), requestOptions{}, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// Create makes a new invoice. Auto-attaches an Idempotency-Key; supply
// [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *InvoicesResource) Create(ctx context.Context, input CreateInvoiceRequest, opts ...RequestOption) (*Invoice, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// Send emails the hosted invoice link to the customer. Auto-attaches an
// Idempotency-Key; supply [WithIdempotencyKey] to override.
func (r *InvoicesResource) Send(ctx context.Context, invoiceID string, opts ...RequestOption) (*Invoice, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/send", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// Remind re-sends a reminder for an unpaid invoice. Auto-attaches an
// Idempotency-Key; supply [WithIdempotencyKey] to override.
func (r *InvoicesResource) Remind(ctx context.Context, invoiceID string, opts ...RequestOption) (*Invoice, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/reminder", reqOpts, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// Void marks an unpaid invoice as void. Terminal state.
func (r *InvoicesResource) Void(ctx context.Context, invoiceID string) (*Invoice, error) {
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/void", requestOptions{
		method: http.MethodPost,
	}, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// ListIter returns a cursor-driven iterator over all invoices matching
// the query.
func (r *InvoicesResource) ListIter(ctx context.Context, query *ListInvoicesQuery) *Iter[Invoice] {
	return newIter(ctx, func(ctx context.Context, cursor string) (*CursorPage[Invoice], error) {
		q := ListInvoicesQuery{}
		if query != nil {
			q = *query
		}
		if cursor != "" {
			q.Cursor = cursor
		}
		return r.List(ctx, &q)
	})
}
