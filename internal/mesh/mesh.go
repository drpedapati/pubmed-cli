// Package mesh provides MeSH term lookup via NCBI E-utilities.
package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MeSHRecord represents a MeSH descriptor record.
type MeSHRecord struct {
	UI          string   `json:"ui"`
	Name        string   `json:"name"`
	ScopeNote   string   `json:"scope_note"`
	TreeNumbers []string `json:"tree_numbers"`
	EntryTerms  []string `json:"entry_terms"`
	Annotation  string   `json:"annotation,omitempty"`
}

// Client provides MeSH lookup functionality.
type Client struct {
	baseURL    string
	apiKey     string
	tool       string
	email      string
	httpClient *http.Client
}

// NewClient creates a new MeSH lookup client.
func NewClient(baseURL, apiKey, tool, email string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		tool:    tool,
		email:   email,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// esearchResult for parsing MeSH search.
type meshSearchResponse struct {
	Result meshSearchResult `json:"esearchresult"`
}

type meshSearchResult struct {
	Count  string   `json:"count"`
	IDList []string `json:"idlist"`
}

// Lookup searches for a MeSH term and returns its record.
func (c *Client) Lookup(ctx context.Context, term string) (*MeSHRecord, error) {
	if term == "" {
		return nil, fmt.Errorf("MeSH term cannot be empty")
	}

	// Step 1: Search for the term in MeSH database
	ids, err := c.searchMeSH(ctx, term)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("MeSH term %q not found", term)
	}

	// Step 2: Fetch the full record
	record, err := c.fetchMeSH(ctx, ids[0])
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Client) searchMeSH(ctx context.Context, term string) ([]string, error) {
	params := url.Values{}
	params.Set("db", "mesh")
	params.Set("term", term)
	params.Set("retmode", "json")
	c.addCommonParams(params)

	resp, err := c.doGet(ctx, "esearch.fcgi", params)
	if err != nil {
		return nil, fmt.Errorf("MeSH search failed: %w", err)
	}

	var result meshSearchResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing MeSH search response: %w", err)
	}

	return result.Result.IDList, nil
}

func (c *Client) fetchMeSH(ctx context.Context, uid string) (*MeSHRecord, error) {
	params := url.Values{}
	params.Set("db", "mesh")
	params.Set("id", uid)
	params.Set("rettype", "full")
	params.Set("retmode", "text")
	c.addCommonParams(params)

	body, err := c.doGet(ctx, "efetch.fcgi", params)
	if err != nil {
		return nil, fmt.Errorf("MeSH fetch failed: %w", err)
	}

	record := parseMeSHRecord(string(body))
	return &record, nil
}

func (c *Client) addCommonParams(params url.Values) {
	if c.apiKey != "" {
		params.Set("api_key", c.apiKey)
	}
	if c.tool != "" {
		params.Set("tool", c.tool)
	}
	if c.email != "" {
		params.Set("email", c.email)
	}
}

func (c *Client) doGet(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	fullURL := fmt.Sprintf("%s/%s?%s", c.baseURL, endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NCBI returned HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// parseMeSHRecord parses the NCBI MeSH full text format into a MeSHRecord.
func parseMeSHRecord(text string) MeSHRecord {
	record := MeSHRecord{}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "*NEWRECORD" {
			continue
		}

		parts := strings.SplitN(line, " = ", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "MH":
			record.Name = value
		case "UI":
			record.UI = value
		case "MS":
			record.ScopeNote = value
		case "MN":
			record.TreeNumbers = append(record.TreeNumbers, value)
		case "AN":
			record.Annotation = value
		case "ENTRY":
			// Entry terms have format: "Term|T047|..."
			entryParts := strings.SplitN(value, "|", 2)
			record.EntryTerms = append(record.EntryTerms, strings.TrimSpace(entryParts[0]))
		}
	}

	return record
}
