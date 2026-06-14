package opensettle

import (
	"context"
	"net/http"
	"net/url"
)

// ProductsResource exposes /v1/workspaces/<ws>/products and the nested
// /prices collection.
type ProductsResource struct {
	http *httpClient
}

// List returns one page of products. Pass nil for an unfiltered first
// page; use ListIter for cursor-driven full iteration.
func (r *ProductsResource) List(ctx context.Context, query *ListProductsQuery) (*CursorPage[Product], error) {
	q := map[string]any{}
	if query != nil {
		if query.Cursor != "" {
			q["cursor"] = query.Cursor
		}
		if query.Limit > 0 {
			q["limit"] = query.Limit
		}
		if query.Active != nil {
			q["active"] = *query.Active
		}
	}
	out := &CursorPage[Product]{}
	err := r.http.request(ctx, "/products", requestOptions{query: q}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Retrieve fetches a single product by ID.
func (r *ProductsResource) Retrieve(ctx context.Context, productID string) (*Product, error) {
	var wrapper struct {
		Product *Product `json:"product"`
	}
	err := r.http.request(ctx, "/products/"+url.PathEscape(productID), requestOptions{}, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper.Product, nil
}

// Create makes a new product. Auto-attaches an Idempotency-Key; supply
// [WithIdempotencyKey] to use a caller-chosen key instead.
func (r *ProductsResource) Create(ctx context.Context, input CreateProductRequest, opts ...RequestOption) (*Product, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var wrapper struct {
		Product *Product `json:"product"`
	}
	err := r.http.request(ctx, "/products", reqOpts, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper.Product, nil
}

// Update patches a product. Fields left nil on input are unchanged.
func (r *ProductsResource) Update(ctx context.Context, productID string, input UpdateProductRequest) (*Product, error) {
	var wrapper struct {
		Product *Product `json:"product"`
	}
	err := r.http.request(ctx, "/products/"+url.PathEscape(productID), requestOptions{
		method: http.MethodPatch,
		body:   input,
	}, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper.Product, nil
}

// Delete is a hard-delete. Returns *ConflictError (409) if any
// subscription still references the product.
func (r *ProductsResource) Delete(ctx context.Context, productID string) error {
	return r.http.request(ctx, "/products/"+url.PathEscape(productID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}

// ListPrices returns the prices attached to a product.
func (r *ProductsResource) ListPrices(ctx context.Context, productID string) ([]Price, error) {
	out := &rawList[Price]{}
	err := r.http.request(ctx, "/products/"+url.PathEscape(productID)+"/prices", requestOptions{}, out)
	if err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CreatePrice attaches a new price to a product. Auto-attaches an
// Idempotency-Key; supply [WithIdempotencyKey] to override.
func (r *ProductsResource) CreatePrice(ctx context.Context, productID string, input CreatePriceRequest, opts ...RequestOption) (*Price, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}
	cfg.applyTo(&reqOpts)
	var wrapper struct {
		Price *Price `json:"price"`
	}
	err := r.http.request(ctx, "/products/"+url.PathEscape(productID)+"/prices", reqOpts, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper.Price, nil
}

// UpdatePrice patches a price (PATCH /prices/<id>). Only Active and
// Metadata are mutable — amount, currency, and interval are immutable
// once a price exists. Fields left nil on input are unchanged.
//
// Unlike CreatePrice, this endpoint does not require an Idempotency-Key
// (a PATCH of a single named price is naturally idempotent), so no key is
// attached by default; pass [WithIdempotencyKey] if you want to supply one.
func (r *ProductsResource) UpdatePrice(ctx context.Context, priceID string, input UpdatePriceRequest, opts ...RequestOption) (*Price, error) {
	cfg := newRequestConfig(opts)
	reqOpts := requestOptions{
		method: http.MethodPatch,
		body:   input,
	}
	cfg.applyTo(&reqOpts)
	var wrapper struct {
		Price *Price `json:"price"`
	}
	err := r.http.request(ctx, "/prices/"+url.PathEscape(priceID), reqOpts, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper.Price, nil
}

// DeletePrice hard-deletes a price. Returns *ConflictError (409) if any
// subscription still references it.
func (r *ProductsResource) DeletePrice(ctx context.Context, priceID string) error {
	return r.http.request(ctx, "/prices/"+url.PathEscape(priceID), requestOptions{
		method: http.MethodDelete,
	}, nil)
}

// ListIter returns a cursor-driven iterator over all products matching
// the query.
func (r *ProductsResource) ListIter(ctx context.Context, query *ListProductsQuery) *Iter[Product] {
	return newIter(ctx, func(ctx context.Context, cursor string) (*CursorPage[Product], error) {
		q := ListProductsQuery{}
		if query != nil {
			q = *query
		}
		if cursor != "" {
			q.Cursor = cursor
		}
		return r.List(ctx, &q)
	})
}
