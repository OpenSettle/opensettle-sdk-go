package opensettle

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewClient_RequiresAPIKey(t *testing.T) {
	_, err := NewClient("", "ws_x")
	if err == nil {
		t.Fatal("expected error")
	}
	cfgErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("want *ConfigError, got %T", err)
	}
	if !strings.Contains(cfgErr.Error(), "apiKey is required") {
		t.Fatalf("unexpected message: %s", cfgErr.Error())
	}
}

func TestNewClient_RequiresWorkspaceID(t *testing.T) {
	_, err := NewClient("sk_test_x", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*ConfigError); !ok {
		t.Fatalf("want *ConfigError, got %T", err)
	}
}

func TestNewClient_RejectsMalformedKey(t *testing.T) {
	for _, key := range []string{"sk_foo_x", "pk_test_x", "live_key", "test_key", "x"} {
		_, err := NewClient(key, "ws_x")
		if err == nil {
			t.Fatalf("expected error for key %q", key)
		}
		if !strings.Contains(err.Error(), "sk_live_ or sk_test_") {
			t.Fatalf("wrong message for %q: %s", key, err.Error())
		}
	}
}

func TestNewClient_AcceptsLiveKey(t *testing.T) {
	c, err := NewClient("sk_live_abc", "ws_x")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c == nil {
		t.Fatal("client nil")
	}
}

func TestNewClient_AcceptsTestKey(t *testing.T) {
	c, err := NewClient("sk_test_abc", "ws_x")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c == nil {
		t.Fatal("client nil")
	}
}

func TestNewClient_TestModeTrueRejectsLiveKey(t *testing.T) {
	_, err := NewClient("sk_live_abc", "ws_x", WithTestMode(true))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "testMode=true") {
		t.Fatalf("unexpected: %s", err)
	}
}

func TestNewClient_TestModeFalseRejectsTestKey(t *testing.T) {
	_, err := NewClient("sk_test_abc", "ws_x", WithTestMode(false))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "testMode=false") {
		t.Fatalf("unexpected: %s", err)
	}
}

func TestNewClient_TestModeTrueAcceptsTestKey(t *testing.T) {
	_, err := NewClient("sk_test_abc", "ws_x", WithTestMode(true))
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewClient_TestModeFalseAcceptsLiveKey(t *testing.T) {
	_, err := NewClient("sk_live_abc", "ws_x", WithTestMode(false))
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	if c.http.baseURL != "https://api.opensettle.io" {
		t.Fatalf("got %q", c.http.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x", WithBaseURL("https://custom.example"))
	if err != nil {
		t.Fatal(err)
	}
	if c.http.baseURL != "https://custom.example" {
		t.Fatalf("got %q", c.http.baseURL)
	}
}

func TestWithMaxRetries(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x", WithMaxRetries(7))
	if err != nil {
		t.Fatal(err)
	}
	if c.http.maxRetries != 7 {
		t.Fatalf("got %d", c.http.maxRetries)
	}
}

func TestWithMaxRetriesZeroDisables(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x", WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	if c.http.maxRetries != 0 {
		t.Fatalf("got %d", c.http.maxRetries)
	}
}

func TestWithTimeout(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x", WithTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if c.http.timeout != 5*time.Second {
		t.Fatalf("got %v", c.http.timeout)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 1 * time.Second}
	c, err := NewClient("sk_test_x", "ws_x", WithHTTPClient(custom))
	if err != nil {
		t.Fatal(err)
	}
	if c.http.hc != custom {
		t.Fatal("custom http client not used")
	}
}

func TestWithUserAgentAppends(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x", WithUserAgent("my-app/1.0"))
	if err != nil {
		t.Fatal(err)
	}
	want := "opensettle-go/" + Version + " my-app/1.0"
	if c.http.userAgent != want {
		t.Fatalf("got %q, want %q", c.http.userAgent, want)
	}
}

func TestDefaultUserAgent(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	want := "opensettle-go/" + Version
	if c.http.userAgent != want {
		t.Fatalf("got %q, want %q", c.http.userAgent, want)
	}
}

func TestClient_ResourcesAttached(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	if c.Checkouts == nil ||
		c.Customers == nil ||
		c.Invoices == nil ||
		c.Payments == nil ||
		c.Products == nil ||
		c.Subscriptions == nil ||
		c.WebhookEndpoints == nil {
		t.Fatal("one or more resources are nil")
	}
}

func TestClient_DefaultRetries(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	if c.http.maxRetries != 3 {
		t.Fatalf("got %d", c.http.maxRetries)
	}
}

func TestClient_DefaultTimeout(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	if c.http.timeout != 30*time.Second {
		t.Fatalf("got %v", c.http.timeout)
	}
}

func TestClient_DefaultHTTPClient(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	if c.http.hc == nil {
		t.Fatal("http client nil")
	}
}

func TestClient_WorkspaceID(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_42")
	if err != nil {
		t.Fatal(err)
	}
	if c.http.workspaceID != "ws_42" {
		t.Fatalf("got %q", c.http.workspaceID)
	}
}

func TestClient_APIKey(t *testing.T) {
	c, err := NewClient("sk_test_secret", "ws_x")
	if err != nil {
		t.Fatal(err)
	}
	if c.http.apiKey != "sk_test_secret" {
		t.Fatalf("got %q", c.http.apiKey)
	}
}

func TestIsOpenSettleErrorTrueForSubtype(t *testing.T) {
	err := &NotFoundError{&OpenSettleError{Code: CodeNotFound, Status: 404}}
	if !IsOpenSettleError(err) {
		t.Fatal("expected true")
	}
}

func TestIsOpenSettleErrorFalseForOther(t *testing.T) {
	if IsOpenSettleError(&ConfigError{Message: "x"}) {
		t.Fatal("expected false")
	}
	if IsOpenSettleError(nil) {
		t.Fatal("expected false for nil")
	}
}

func TestConfigErrorString(t *testing.T) {
	e := &ConfigError{Message: "boom"}
	if !strings.HasPrefix(e.Error(), "opensettle config: boom") {
		t.Fatalf("got %q", e.Error())
	}
}

func TestNewClient_UserAgentEmptyDoesNotAppend(t *testing.T) {
	c, err := NewClient("sk_test_x", "ws_x", WithUserAgent(""))
	if err != nil {
		t.Fatal(err)
	}
	want := "opensettle-go/" + Version
	if c.http.userAgent != want {
		t.Fatalf("got %q want %q", c.http.userAgent, want)
	}
}

func TestNewClient_TestModeUnsetAcceptsEither(t *testing.T) {
	if _, err := NewClient("sk_live_x", "ws_x"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewClient("sk_test_x", "ws_x"); err != nil {
		t.Fatal(err)
	}
}
