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

func (r *ProductsResource) Create(ctx context.Context, input CreateProductRequest) (*Product, error) {
	var wrapper struct {
		Product *Product `json:"product"`
	}
	err := r.http.request(ctx, "/products", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper.Product, nil
}

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

// CreatePrice attaches a new price to a product.
func (r *ProductsResource) CreatePrice(ctx context.Context, productID string, input CreatePriceRequest) (*Price, error) {
	var wrapper struct {
		Price *Price `json:"price"`
	}
	err := r.http.request(ctx, "/products/"+url.PathEscape(productID)+"/prices", requestOptions{
		method:      http.MethodPost,
		body:        input,
		idempotency: idempotency{mode: idempotencyAuto},
	}, &wrapper)
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
