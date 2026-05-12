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
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID), requestOptions{}, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

func (r *InvoicesResource) Create(ctx context.Context, input CreateInvoiceRequest) (*Invoice, error) {
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// Send emails the hosted invoice link to the customer.
func (r *InvoicesResource) Send(ctx context.Context, invoiceID string) (*Invoice, error) {
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/send", requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}, &w)
	if err != nil {
		return nil, err
	}
	return w.Invoice, nil
}

// Remind re-sends a reminder for an unpaid invoice.
func (r *InvoicesResource) Remind(ctx context.Context, invoiceID string) (*Invoice, error) {
	var w invoiceWrapper
	err := r.http.request(ctx, "/invoices/"+url.PathEscape(invoiceID)+"/reminder", requestOptions{
		method:      http.MethodPost,
		idempotency: idempotency{mode: idempotencyAuto},
	}, &w)
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
