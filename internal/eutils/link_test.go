package eutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCitedBy_Success(t *testing.T) {
	fixture := loadTestdata(t, "elink_citedin.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("dbfrom"); got != "pubmed" {
			t.Errorf("expected dbfrom=pubmed, got %q", got)
		}
		if got := q.Get("db"); got != "pubmed" {
			t.Errorf("expected db=pubmed, got %q", got)
		}
		if got := q.Get("id"); got != "38123456" {
			t.Errorf("expected id=38123456, got %q", got)
		}
		if got := q.Get("linkname"); got != "pubmed_pubmed_citedin" {
			t.Errorf("expected linkname=pubmed_pubmed_citedin, got %q", got)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	result, err := c.CitedBy(context.Background(), "38123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SourceID != "38123456" {
		t.Errorf("expected source ID '38123456', got %q", result.SourceID)
	}
	if len(result.Links) != 5 {
		t.Fatalf("expected 5 links, got %d", len(result.Links))
	}
	if result.Links[0].ID != "39000001" {
		t.Errorf("expected first link ID '39000001', got %q", result.Links[0].ID)
	}
}

func TestReferences_Success(t *testing.T) {
	fixture := loadTestdata(t, "elink_refs.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("linkname"); got != "pubmed_pubmed_refs" {
			t.Errorf("expected linkname=pubmed_pubmed_refs, got %q", got)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	result, err := c.References(context.Background(), "38123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(result.Links))
	}
}

func TestRelated_WithScores(t *testing.T) {
	fixture := loadTestdata(t, "elink_related.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("linkname"); got != "pubmed_pubmed" {
			t.Errorf("expected linkname=pubmed_pubmed, got %q", got)
		}
		if got := q.Get("cmd"); got != "neighbor_score" {
			t.Errorf("expected cmd=neighbor_score, got %q", got)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	result, err := c.Related(context.Background(), "38123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Links) != 4 {
		t.Fatalf("expected 4 links, got %d", len(result.Links))
	}
	if result.Links[0].Score != 98765432 {
		t.Errorf("expected first score 98765432, got %d", result.Links[0].Score)
	}
	if result.Links[0].ID != "38500001" {
		t.Errorf("expected first link ID '38500001', got %q", result.Links[0].ID)
	}
}

func TestLink_EmptyResults(t *testing.T) {
	fixture := loadTestdata(t, "elink_empty.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))

	result, err := c.CitedBy(context.Background(), "99999999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(result.Links))
	}
}

func TestLink_EmptyPMID(t *testing.T) {
	c := NewClient(WithAPIKey("test"))

	_, err := c.CitedBy(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty PMID")
	}

	_, err = c.References(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty PMID")
	}

	_, err = c.Related(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty PMID")
	}
}

func TestLink_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	_, err := c.CitedBy(context.Background(), "12345")
	if err == nil {
		t.Error("expected error for server error")
	}
}
