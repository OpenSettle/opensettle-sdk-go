package opensettle

// RequestOption is a per-call knob that can be supplied to any
// money-adjacent write method (Create, Refund, RotateSecret, etc.). The
// SDK applies all options after its own defaults, so options always win.
//
// Today the only option is WithIdempotencyKey; the type is exported so
// the SDK can grow per-call options without breaking method signatures.
type RequestOption func(*requestConfig)

// requestConfig is the resolved per-call config produced by collapsing
// all RequestOption values supplied to a method.
type requestConfig struct {
	idempotencyKey string
}

// WithIdempotencyKey supplies a caller-chosen Idempotency-Key for this
// request. Use this when you want to assign a deterministic key (e.g.
// your own order id) so that retries from different machines or
// processes collapse to the same server-side operation.
//
// If WithIdempotencyKey is not supplied, the SDK auto-generates a
// UUIDv4 key for every money-adjacent write. Either way, the same key
// is reused across all retry attempts of a single call.
func WithIdempotencyKey(key string) RequestOption {
	return func(c *requestConfig) { c.idempotencyKey = key }
}

// applyTo folds the per-call config into a resource-method's
// requestOptions. Caller-supplied keys override the default auto mode.
func (c requestConfig) applyTo(opts *requestOptions) {
	if c.idempotencyKey != "" {
		opts.idempotency = idempotency{mode: idempotencyExplicit, value: c.idempotencyKey}
	}
}

// newRequestConfig collapses a variadic option slice into a single
// resolved config. Empty input yields the zero-value config (= "no
// per-call overrides").
func newRequestConfig(opts []RequestOption) requestConfig {
	var c requestConfig
	for _, o := range opts {
		o(&c)
	}
	return c
}
