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
	}
	out := &CursorPage[Invoice]{}
	err := r.http.request(ctx, "/invoices", requestOptions{query: q}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *InvoicesResource) Retrieve(ctx context.Context, invoiceID string) (*Invoice, error) {
	out := &Invoice{}
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID), requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *InvoicesResource) Create(ctx context.Context, input CreateInvoiceRequest) (*Invoice, error) {
	out := &Invoice{}
	err := r.http.request(ctx, "/invoices", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Send emails the hosted invoice link to the customer.
func (r *InvoicesResource) Send(ctx context.Context, invoiceID string) (*Invoice, error) {
	out := &Invoice{}
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/send", requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Remind re-sends a reminder for an unpaid invoice.
func (r *InvoicesResource) Remind(ctx context.Context, invoiceID string) (*Invoice, error) {
	out := &Invoice{}
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/reminder", requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Void marks an unpaid invoice as void. Terminal state.
func (r *InvoicesResource) Void(ctx context.Context, invoiceID string) (*Invoice, error) {
	out := &Invoice{}
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/void", requestOptions{
		method: http.MethodPost,
	}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
