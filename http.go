package opensettle

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// httpClient is the SDK's transport layer. It is intentionally minimal:
// stdlib net/http plus the bits every API client needs — bearer auth,
// JSON encoding, idempotency-key injection, bounded retry with
// exponential backoff that honors Retry-After, and typed error mapping.
type httpClient struct {
	apiKey      string
	workspaceID string
	baseURL     string
	userAgent   string
	maxRetries  int
	timeout     time.Duration
	hc          *http.Client
	// now is injected for tests so we can control backoff math; production
	// uses time.Now.
	now func() time.Time
	// sleep is injected for tests so retries don't actually wait.
	sleep func(d time.Duration)
	// newKey is injected for tests so we can assert idempotency keys.
	newKey func() string
}

// requestOptions are the per-call knobs a resource module may set.
type requestOptions struct {
	method          string
	body            any
	query           map[string]any
	headers         map[string]string
	idempotency     idempotency
	timeoutOverride time.Duration
	retriesOverride int
	noWorkspace     bool
}

// idempotency captures the three possible states a caller can request:
// no key, generate one, or use a caller-supplied key.
type idempotency struct {
	mode  int // 0 = none, 1 = auto, 2 = explicit
	value string
}

const (
	idempotencyNone     = 0
	idempotencyAuto     = 1
	idempotencyExplicit = 2
)

const (
	defaultBaseURL    = "https://api.opensettle.io"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	// backoffCap is the upper bound on each per-attempt backoff.
	backoffCap = 4 * time.Second
)

var retriableStatuses = map[int]struct{}{
	408: {}, 425: {}, 429: {}, 500: {}, 502: {}, 503: {}, 504: {},
}

// request issues a workspace-scoped call (`/v1/workspaces/<ws>` prefix
// added automatically). out, if non-nil, receives the decoded JSON
// response body.
func (h *httpClient) request(ctx context.Context, path string, opts requestOptions, out any) error {
	u, err := h.buildURL(path, opts.query, opts.noWorkspace)
	if err != nil {
		return err
	}
	return h.do(ctx, u, opts, out)
}

func (h *httpClient) buildURL(path string, query map[string]any, noWorkspace bool) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	base := strings.TrimRight(h.baseURL, "/")
	var full string
	if noWorkspace {
		full = base + path
	} else {
		full = fmt.Sprintf("%s/v1/workspaces/%s%s", base, url.PathEscape(h.workspaceID), path)
	}
	if q := encodeQuery(query); q != "" {
		full += "?" + q
	}
	// Round-trip through url.Parse to catch bogus paths early; users get
	// a typed *InvalidRequestError rather than an opaque transport error.
	if _, err := url.Parse(full); err != nil {
		return "", &InvalidRequestError{&OpenSettleError{
			Code:    CodeInvalidRequest,
			Message: fmt.Sprintf("invalid url: %v", err),
		}}
	}
	return full, nil
}

func encodeQuery(query map[string]any) string {
	if len(query) == 0 {
		return ""
	}
	v := url.Values{}
	// Deterministic ordering so request URLs are stable in tests.
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := query[k]
		if val == nil {
			continue
		}
		switch t := val.(type) {
		case string:
			if t == "" {
				continue
			}
			v.Set(k, t)
		case bool:
			v.Set(k, strconv.FormatBool(t))
		case int:
			v.Set(k, strconv.Itoa(t))
		case int64:
			v.Set(k, strconv.FormatInt(t, 10))
		case float64:
			v.Set(k, strconv.FormatFloat(t, 'f', -1, 64))
		default:
			// Fall back to fmt for anything that has a sensible String.
			s := fmt.Sprintf("%v", t)
			if s == "" {
				continue
			}
			v.Set(k, s)
		}
	}
	return v.Encode()
}

func (h *httpClient) do(ctx context.Context, fullURL string, opts requestOptions, out any) error {
	method := opts.method
	if method == "" {
		method = http.MethodGet
	}

	var bodyBytes []byte
	if opts.body != nil {
		b, err := json.Marshal(opts.body)
		if err != nil {
			return &InvalidRequestError{&OpenSettleError{
				Code:    CodeInvalidRequest,
				Message: fmt.Sprintf("encode body: %v", err),
			}}
		}
		bodyBytes = b
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+h.apiKey)
	headers.Set("Accept", "application/json")
	headers.Set("User-Agent", h.userAgent)
	if bodyBytes != nil {
		headers.Set("Content-Type", "application/json")
	}
	for k, v := range opts.headers {
		headers.Set(k, v)
	}

	switch opts.idempotency.mode {
	case idempotencyAuto:
		if headers.Get("Idempotency-Key") == "" {
			headers.Set("Idempotency-Key", h.newKey())
		}
	case idempotencyExplicit:
		headers.Set("Idempotency-Key", opts.idempotency.value)
	}

	timeout := h.timeout
	if opts.timeoutOverride > 0 {
		timeout = opts.timeoutOverride
	}
	retries := h.maxRetries
	switch {
	case opts.retriesOverride > 0:
		retries = opts.retriesOverride
	case opts.retriesOverride == -1:
		// -1 is the explicit "disable retries" sentinel; 0 means
		// "leave the client default in place".
		retries = 0
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		// Per-attempt context — applies the configured timeout if the
		// caller's context doesn't already have a tighter deadline.
		reqCtx, cancel := contextWithTimeout(ctx, timeout)

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(reqCtx, method, fullURL, reqBody)
		if err != nil {
			cancel()
			return &InvalidRequestError{&OpenSettleError{
				Code:    CodeInvalidRequest,
				Message: fmt.Sprintf("build request: %v", err),
			}}
		}
		req.Header = headers.Clone()

		resp, err := h.hc.Do(req)
		if err != nil {
			cancel()
			// Surface caller-cancelled contexts cleanly as a non-retryable
			// transport error — retrying a cancelled context just produces
			// the same error.
			if ctx.Err() != nil {
				return newNetworkError(fmt.Sprintf("network error: %v", ctx.Err()))
			}
			lastErr = newNetworkError(fmt.Sprintf("network error: %v", err))
			if attempt < retries {
				h.sleep(backoffFor(attempt, 0))
				continue
			}
			return lastErr
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = newNetworkError(fmt.Sprintf("read body: %v", readErr))
			if attempt < retries {
				h.sleep(backoffFor(attempt, 0))
				continue
			}
			return lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
				return nil
			}
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(body, out); err != nil {
				// Successful HTTP code but body isn't JSON — surface as
				// APIError, not retryable. This is a server-side bug, not
				// a transport blip.
				return &APIError{&OpenSettleError{
					Code:    CodeInternalError,
					Message: "server returned 2xx with non-JSON body",
					Status:  resp.StatusCode,
				}}
			}
			return nil
		}

		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), h.now())
		apiErr := FromEnvelope(body, resp.StatusCode, retryAfter)

		if _, retriable := retriableStatuses[resp.StatusCode]; retriable && attempt < retries {
			wait := backoffFor(attempt, retryAfter)
			h.sleep(wait)
			lastErr = apiErr
			continue
		}
		return apiErr
	}
	if lastErr == nil {
		return newNetworkError("request loop exited without result")
	}
	return lastErr
}

// contextWithTimeout returns a derived context that's bounded by the
// shorter of the caller's existing deadline (if any) and the configured
// timeout. A zero or negative timeout means "no per-request bound" —
// we still return a derived context so the cancel func is well-defined.
func contextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	if d, ok := parent.Deadline(); ok {
		// Caller already has a tighter deadline — respect it.
		if time.Until(d) < timeout {
			return context.WithCancel(parent)
		}
	}
	return context.WithTimeout(parent, timeout)
}

// parseRetryAfter accepts the two legal Retry-After forms — delta-seconds
// or HTTP-date — and returns the wait in seconds. Returns 0 on parse
// failure so callers can fall through to exponential backoff.
func parseRetryAfter(value string, now time.Time) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(value, 64); err == nil {
		if secs < 0 {
			return 0
		}
		return secs
	}
	if t, err := http.ParseTime(value); err == nil {
		delta := t.Sub(now).Seconds()
		if delta < 0 {
			return 0
		}
		return delta
	}
	return 0
}

// backoffFor returns the wait before retry attempt `attempt` (0-indexed).
// If the server advertised a Retry-After we honor that; otherwise we use
// 250 ms · 2^attempt capped at backoffCap. Matches the Node SDK exactly.
func backoffFor(attempt int, retryAfterSeconds float64) time.Duration {
	if retryAfterSeconds > 0 {
		return time.Duration(retryAfterSeconds * float64(time.Second))
	}
	base := time.Duration(math.Pow(2, float64(attempt))) * 250 * time.Millisecond
	if base > backoffCap {
		return backoffCap
	}
	return base
}

// idempotencyFallbackCounter ensures fallback keys stay unique across
// concurrent goroutines on the same nanosecond when crypto/rand fails.
var idempotencyFallbackCounter uint64

// generateIdempotencyKey returns a UUIDv4 string. Used when the caller
// passes idempotencyAuto and didn't supply one of their own. No third-
// party deps — stdlib crypto/rand + a 16-byte format-and-set.
func generateIdempotencyKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should never fail on a working machine; if it does
		// we fall back to a time-derived value with an atomic counter so
		// concurrent callers on the same nanosecond still get unique keys.
		t := time.Now().UnixNano()
		n := atomic.AddUint64(&idempotencyFallbackCounter, 1)
		return fmt.Sprintf("os_%016x_%016x", t, n)
	}
	// Per RFC 4122 — version 4, variant 10.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst)
}
