// Package llm provides an OpenAI-compatible API client for LLM inference.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client wraps an OpenAI-compatible API endpoint.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// Option configures the LLM client.
type Option func(*Client)

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// NewClient creates a new LLM client with sensible defaults.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL: "https://api.openai.com/v1",
		model:   "gpt-4o",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}

	// Check environment variables
	if url := os.Getenv("LLM_BASE_URL"); url != "" {
		c.baseURL = url
	}
	if key := os.Getenv("LLM_API_KEY"); key == "" {
		if key = os.Getenv("OPENAI_API_KEY"); key != "" {
			c.apiKey = key
		}
	} else {
		c.apiKey = key
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		c.model = model
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for chat completions.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature"`
}

// ChatResponse is the response from chat completions.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete sends a chat completion request and returns the response text.
func (c *Client) Complete(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return c.CompleteMessages(ctx, []Message{{Role: "user", Content: prompt}}, maxTokens)
}

// CompleteMessages sends a chat completion request with multiple messages.
func (c *Client) CompleteMessages(ctx context.Context, messages []Message, maxTokens int) (string, error) {
	req := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
