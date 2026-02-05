package eutils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the NCBI E-utilities base URL.
	DefaultBaseURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils"
	// DefaultTool identifies this application to NCBI.
	DefaultTool = "pubmed-cli"
	// DefaultEmail is the contact email sent to NCBI.
	DefaultEmail = "pubmed-cli@users.noreply.github.com"

	// Rate limits
	rateWithoutKey = 3  // requests per second without API key
	rateWithKey    = 10 // requests per second with API key
)

// Client is an HTTP client for NCBI E-utilities.
type Client struct {
	baseURL    string
	apiKey     string
	tool       string
	email      string
	httpClient *http.Client

	mu          sync.Mutex
	lastRequest time.Time
	rateLimit   time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the base URL for E-utilities requests.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithAPIKey sets the NCBI API key.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
		if key != "" {
			c.rateLimit = time.Second / time.Duration(rateWithKey)
		}
	}
}

// WithTool sets the tool parameter for NCBI requests.
func WithTool(tool string) Option {
	return func(c *Client) { c.tool = tool }
}

// WithEmail sets the email parameter for NCBI requests.
func WithEmail(email string) Option {
	return func(c *Client) { c.email = email }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new E-utilities client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		tool:      DefaultTool,
		email:     DefaultEmail,
		rateLimit: time.Second / time.Duration(rateWithoutKey),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// doGet performs a rate-limited GET request and returns the response body.
func (c *Client) doGet(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	// Rate limiting
	c.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(c.lastRequest)
	if elapsed < c.rateLimit {
		wait := c.rateLimit - elapsed
		c.mu.Unlock()
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		c.mu.Lock()
	}
	c.lastRequest = time.Now()
	c.mu.Unlock()

	// Add common params
	if c.apiKey != "" {
		params.Set("api_key", c.apiKey)
	}
	if c.tool != "" {
		params.Set("tool", c.tool)
	}
	if c.email != "" {
		params.Set("email", c.email)
	}

	fullURL := fmt.Sprintf("%s/%s?%s", c.baseURL, endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("NCBI rate limit exceeded (HTTP 429). Consider using an API key with --api-key or NCBI_API_KEY env var")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NCBI returned HTTP %d for %s", resp.StatusCode, endpoint)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return body, nil
}
