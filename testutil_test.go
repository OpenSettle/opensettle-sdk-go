package opensettle

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// recordedRequest captures everything a test needs to assert about what
// the SDK actually sent over the wire.
type recordedRequest struct {
	Method  string
	Path    string
	RawPath string
	Query   string
	Headers http.Header
	Body    []byte
}

// stubServer is the shared httptest harness. It records every request
// and lets a test program the response per call.
type stubServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []recordedRequest
	// responses is a queue of canned responses, consumed in order.
	// When empty, the server returns 500 to make accidental over-calls
	// loud in tests.
	responses []cannedResponse
}

type cannedResponse struct {
	Status  int
	Body    string
	Headers map[string]string
	// Delay artificially slows the response (for timeout tests).
	Delay time.Duration
}

func newStubServer() *stubServer {
	s := &stubServer{}
	s.server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *stubServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	s.mu.Lock()
	s.requests = append(s.requests, recordedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		RawPath: r.URL.EscapedPath(),
		Query:   r.URL.RawQuery,
		Headers: r.Header.Clone(),
		Body:    body,
	})
	if len(s.responses) == 0 {
		s.mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"unexpected call in test"}}`))
		return
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	s.mu.Unlock()
	if resp.Delay > 0 {
		time.Sleep(resp.Delay)
	}
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if resp.Body != "" && resp.Headers["Content-Type"] == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	if resp.Status == 0 {
		resp.Status = http.StatusOK
	}
	w.WriteHeader(resp.Status)
	_, _ = w.Write([]byte(resp.Body))
}

func (s *stubServer) Close() { s.server.Close() }

// queue is the test-facing helper that appends a response.
func (s *stubServer) queue(status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses = append(s.responses, cannedResponse{Status: status, Body: body})
}

func (s *stubServer) queueWith(resp cannedResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses = append(s.responses, resp)
}

// lastRequest returns the last call the server received, or fatal-fails
// the test if none was made.
func (s *stubServer) lastRequest(t *testing.T) recordedRequest {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		t.Fatalf("no requests were made to the stub server")
	}
	return s.requests[len(s.requests)-1]
}

// newTestClient builds a Client wired to the stub server. By default
// retries are disabled (test isolation) and sleep is a no-op so backoff
// math doesn't slow tests. Individual tests can override these via the
// returned hooks.
func newTestClient(t *testing.T, s *stubServer) *Client {
	t.Helper()
	c, err := NewClient("sk_test_unit", "ws_test",
		WithBaseURL(s.server.URL),
		WithMaxRetries(0),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Replace random idempotency keys with deterministic values for
	// assertion stability, and make sleep instantaneous.
	c.http.newKey = func() string { return "test-idem-key" }
	c.http.sleep = func(time.Duration) {}
	return c
}

// withRetries returns the same Client wired to allow N retries.
func withRetries(c *Client, n int) *Client {
	c.http.maxRetries = n
	return c
}

func decodeBody[T any](t *testing.T, b []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}

func bgCtx() context.Context { return context.Background() }
