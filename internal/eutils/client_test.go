package eutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	if c.baseURL != DefaultBaseURL {
		t.Errorf("expected base URL %q, got %q", DefaultBaseURL, c.baseURL)
	}
	if c.tool != DefaultTool {
		t.Errorf("expected tool %q, got %q", DefaultTool, c.tool)
	}
	if c.email != DefaultEmail {
		t.Errorf("expected email %q, got %q", DefaultEmail, c.email)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	c := NewClient(
		WithBaseURL("http://localhost:9999"),
		WithAPIKey("test-key-123"),
		WithTool("my-tool"),
		WithEmail("test@example.com"),
	)
	if c.baseURL != "http://localhost:9999" {
		t.Errorf("expected base URL %q, got %q", "http://localhost:9999", c.baseURL)
	}
	if c.apiKey != "test-key-123" {
		t.Errorf("expected API key %q, got %q", "test-key-123", c.apiKey)
	}
	if c.tool != "my-tool" {
		t.Errorf("expected tool %q, got %q", "my-tool", c.tool)
	}
	if c.email != "test@example.com" {
		t.Errorf("expected email %q, got %q", "test@example.com", c.email)
	}
}

func TestClient_CommonParams(t *testing.T) {
	var receivedParams map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedParams = make(map[string]string)
		for k, v := range r.URL.Query() {
			receivedParams[k] = v[0]
		}
		w.Write([]byte(`{"esearchresult":{"count":"0","retmax":"20","retstart":"0","idlist":[],"querytranslation":"test"}}`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithAPIKey("my-api-key"),
		WithTool("pubmed-cli"),
		WithEmail("user@example.com"),
	)
	_, _ = c.Search(context.Background(), "test", nil)

	if receivedParams["api_key"] != "my-api-key" {
		t.Errorf("expected api_key %q, got %q", "my-api-key", receivedParams["api_key"])
	}
	if receivedParams["tool"] != "pubmed-cli" {
		t.Errorf("expected tool %q, got %q", "pubmed-cli", receivedParams["tool"])
	}
	if receivedParams["email"] != "user@example.com" {
		t.Errorf("expected email %q, got %q", "user@example.com", receivedParams["email"])
	}
}

func TestClient_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping rate limit test in short mode")
	}
	var requestCount int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.Write([]byte(`{"esearchresult":{"count":"0","retmax":"20","retstart":"0","idlist":[],"querytranslation":"test"}}`))
	}))
	defer srv.Close()

	// Client without API key: max 3 req/sec
	c := NewClient(WithBaseURL(srv.URL))

	start := time.Now()
	for i := 0; i < 4; i++ {
		_, _ = c.Search(context.Background(), "test", nil)
	}
	elapsed := time.Since(start)

	// 4 requests at 3/sec should take at least ~900ms (3 intervals of 333ms)
	if elapsed < 900*time.Millisecond {
		t.Errorf("rate limiting too fast: 4 requests completed in %v (expected >= 900ms)", elapsed)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	// Pre-cancelled context should fail immediately
	c := NewClient(
		WithBaseURL("http://127.0.0.1:1"), // won't connect
		WithAPIKey("test"),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Search(ctx, "test", nil)
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
}
