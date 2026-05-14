package opensettle

import "context"

// Iter is a cursor-driven iterator over a paginated API resource.
// It auto-fetches the next page when the current page is drained and
// stops when the server signals hasMore=false.
//
// Resource List methods continue to return a single page for backward
// compatibility; iterators are returned by separate ListIter (or
// Iter-suffixed) methods on each resource — see the resource files.
//
// Usage:
//
//	it := c.Customers.ListIter(ctx, &opensettle.ListCustomersQuery{Status: opensettle.CustomerActive})
//	for it.Next() {
//	    fmt.Println(it.Item().ID)
//	}
//	if err := it.Err(); err != nil { … }
type Iter[T any] struct {
	ctx       context.Context
	fetchPage func(ctx context.Context, cursor string) (*CursorPage[T], error)
	page      *CursorPage[T]
	pos       int
	item      *T
	err       error
	done      bool
}

// Next advances the iterator. Returns false when the iteration is
// exhausted (call Err to distinguish "natural end" from "error").
func (it *Iter[T]) Next() bool {
	if it.done || it.err != nil {
		return false
	}
	// Lazy-fetch the first page when called before any data.
	if it.page == nil {
		if !it.advancePage("") {
			return false
		}
	}
	// Loop so a stream of empty intermediate pages can't recurse and blow the stack.
	for {
		if it.pos < len(it.page.Data) {
			it.item = &it.page.Data[it.pos]
			it.pos++
			return true
		}
		if !it.page.HasMore || it.page.NextCursor == "" {
			it.done = true
			return false
		}
		if !it.advancePage(it.page.NextCursor) {
			return false
		}
	}
}

// Item returns the most-recently-yielded item. Only valid after Next
// has returned true.
func (it *Iter[T]) Item() *T { return it.item }

// Err returns any error that stopped the iteration. Nil on a clean end.
func (it *Iter[T]) Err() error { return it.err }

func (it *Iter[T]) advancePage(cursor string) bool {
	page, err := it.fetchPage(it.ctx, cursor)
	if err != nil {
		it.err = err
		return false
	}
	it.page = page
	it.pos = 0
	return true
}

// newIter is a small helper constructed by resource ListIter methods.
func newIter[T any](
	ctx context.Context,
	fetch func(ctx context.Context, cursor string) (*CursorPage[T], error),
) *Iter[T] {
	return &Iter[T]{ctx: ctx, fetchPage: fetch}
}
