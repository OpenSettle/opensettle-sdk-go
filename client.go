package opensettle

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// ConfigError is returned by NewClient when the caller's config is
// internally inconsistent (e.g. test-mode with a live key). Kept distinct
// from API-side errors so callers don't have to know they need a
// transport before hitting an obvious config bug.
type ConfigError struct{ Message string }

func (e *ConfigError) Error() string { return "opensettle config: " + e.Message }

// Option configures a Client. Use NewClient(...).
type Option func(*clientConfig)

type clientConfig struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	timeout    time.Duration
	// testMode is *bool so we can distinguish "not asserted" (nil) from
	// "asserted true" and "asserted false".
	testMode  *bool
	userAgent string
}

// WithBaseURL overrides the API host. Defaults to https://api.opensettle.io.
func WithBaseURL(url string) Option {
	return func(c *clientConfig) { c.baseURL = url }
}

// WithHTTPClient replaces the underlying *http.Client. Useful for
// injecting custom transports, proxies, or mocks. The SDK still applies
// its own retry and timeout logic on top.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = hc }
}

// WithMaxRetries sets the retry budget for transient failures (5xx, 429,
// transport errors). Default 3. Set 0 to disable retries entirely.
func WithMaxRetries(n int) Option {
	return func(c *clientConfig) { c.maxRetries = n }
}

// WithTimeout sets the per-request timeout. Default 30s.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) { c.timeout = d }
}

// WithTestMode asserts the apiKey environment matches. With true, the
// SDK refuses sk_live_… keys; with false it refuses sk_test_…. Useful as
// a circuit breaker in CI. Default is unasserted (either accepted).
func WithTestMode(test bool) Option {
	return func(c *clientConfig) { c.testMode = &test }
}

// WithUserAgent appends a caller-supplied product token to the SDK's
// User-Agent. The default is `opensettle-go/<Version>`; with this option
// the header becomes `opensettle-go/<Version> <userAgent>`.
func WithUserAgent(ua string) Option {
	return func(c *clientConfig) { c.userAgent = ua }
}

// Client is the top-level handle. One per workspace. Safe for concurrent
// use by multiple goroutines.
type Client struct {
	http             *httpClient
	Checkouts        *CheckoutsResource
	Customers        *CustomersResource
	Invoices         *InvoicesResource
	PaymentLinks     *PaymentLinksResource
	Payments         *PaymentsResource
	Products         *ProductsResource
	Subscriptions    *SubscriptionsResource
	WebhookEndpoints *WebhookEndpointsResource
}

// NewClient builds a Client. apiKey must start with sk_live_ or sk_test_;
// workspaceID is the merchant's workspace (required on every route).
func NewClient(apiKey, workspaceID string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, &ConfigError{Message: "apiKey is required"}
	}
	if workspaceID == "" {
		return nil, &ConfigError{Message: "workspaceID is required"}
	}

	cfg := clientConfig{
		baseURL:    defaultBaseURL,
		maxRetries: defaultMaxRetries,
		timeout:    defaultTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := assertAPIKeyEnvironment(apiKey, cfg.testMode); err != nil {
		return nil, err
	}

	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{}
	}

	ua := "opensettle-go/" + Version
	if cfg.userAgent != "" {
		ua = ua + " " + cfg.userAgent
	}

	hc := &httpClient{
		apiKey:      apiKey,
		workspaceID: workspaceID,
		baseURL:     cfg.baseURL,
		userAgent:   ua,
		maxRetries:  cfg.maxRetries,
		timeout:     cfg.timeout,
		hc:          cfg.httpClient,
		now:         time.Now,
		sleep:       time.Sleep,
		newKey:      generateIdempotencyKey,
	}

	c := &Client{http: hc}
	c.Checkouts = &CheckoutsResource{http: hc}
	c.Customers = &CustomersResource{http: hc}
	c.Invoices = &InvoicesResource{http: hc}
	c.PaymentLinks = &PaymentLinksResource{http: hc}
	c.Payments = &PaymentsResource{http: hc}
	c.Products = &ProductsResource{http: hc}
	c.Subscriptions = &SubscriptionsResource{http: hc}
	c.WebhookEndpoints = &WebhookEndpointsResource{http: hc}
	return c, nil
}

func assertAPIKeyEnvironment(apiKey string, testMode *bool) error {
	isLive := strings.HasPrefix(apiKey, "sk_live_")
	isTest := strings.HasPrefix(apiKey, "sk_test_")
	if !isLive && !isTest {
		return &ConfigError{Message: "apiKey must start with sk_live_ or sk_test_"}
	}
	if testMode != nil {
		if *testMode && isLive {
			return &ConfigError{Message: "testMode=true but apiKey is sk_live_ (refusing to send live traffic)"}
		}
		if !*testMode && isTest {
			return &ConfigError{Message: "testMode=false but apiKey is sk_test_ (refusing to send to test API)"}
		}
	}
	return nil
}

// IsOpenSettleError reports whether err is or wraps an *OpenSettleError.
// Convenience around errors.As for the broad case.
func IsOpenSettleError(err error) bool {
	var target *OpenSettleError
	return errors.As(err, &target)
}
