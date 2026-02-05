package eutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetch_StructuredAbstract(t *testing.T) {
	fixture := loadTestdata(t, "efetch_response.xml")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("db"); got != "pubmed" {
			t.Errorf("expected db=pubmed, got %q", got)
		}
		if got := q.Get("id"); got != "38123456" {
			t.Errorf("expected id=38123456, got %q", got)
		}
		if got := q.Get("rettype"); got != "xml" {
			t.Errorf("expected rettype=xml, got %q", got)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	articles, err := c.Fetch(context.Background(), []string{"38123456"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}

	a := articles[0]

	// PMID
	if a.PMID != "38123456" {
		t.Errorf("expected PMID '38123456', got %q", a.PMID)
	}

	// Title
	expectedTitle := "EEG biomarkers in fragile X syndrome: a comprehensive review of spectral and connectivity measures."
	if a.Title != expectedTitle {
		t.Errorf("expected title %q, got %q", expectedTitle, a.Title)
	}

	// Structured abstract sections
	if len(a.AbstractSections) != 4 {
		t.Fatalf("expected 4 abstract sections, got %d", len(a.AbstractSections))
	}
	if a.AbstractSections[0].Label != "BACKGROUND" {
		t.Errorf("expected first section label 'BACKGROUND', got %q", a.AbstractSections[0].Label)
	}

	// Full abstract should concatenate sections
	if a.Abstract == "" {
		t.Error("expected non-empty abstract")
	}

	// Authors
	if len(a.Authors) != 3 {
		t.Fatalf("expected 3 authors, got %d", len(a.Authors))
	}
	if a.Authors[0].LastName != "Pedapati" {
		t.Errorf("expected first author 'Pedapati', got %q", a.Authors[0].LastName)
	}
	if a.Authors[0].ForeName != "Ernest V" {
		t.Errorf("expected fore name 'Ernest V', got %q", a.Authors[0].ForeName)
	}
	if a.Authors[0].Affiliation == "" {
		t.Error("expected non-empty affiliation for first author")
	}

	// Journal
	if a.Journal != "Molecular psychiatry" {
		t.Errorf("expected journal 'Molecular psychiatry', got %q", a.Journal)
	}
	if a.JournalAbbrev != "Mol Psychiatry" {
		t.Errorf("expected abbrev 'Mol Psychiatry', got %q", a.JournalAbbrev)
	}
	if a.Volume != "29" {
		t.Errorf("expected volume '29', got %q", a.Volume)
	}
	if a.Issue != "3" {
		t.Errorf("expected issue '3', got %q", a.Issue)
	}
	if a.Year != "2024" {
		t.Errorf("expected year '2024', got %q", a.Year)
	}

	// DOI
	if a.DOI != "10.1038/s41380-024-02456-7" {
		t.Errorf("expected DOI '10.1038/s41380-024-02456-7', got %q", a.DOI)
	}

	// PMCID
	if a.PMCID != "PMC10987654" {
		t.Errorf("expected PMCID 'PMC10987654', got %q", a.PMCID)
	}

	// MeSH terms
	if len(a.MeSHTerms) != 4 {
		t.Fatalf("expected 4 MeSH terms, got %d", len(a.MeSHTerms))
	}
	if a.MeSHTerms[0].Descriptor != "Fragile X Syndrome" {
		t.Errorf("expected first MeSH term 'Fragile X Syndrome', got %q", a.MeSHTerms[0].Descriptor)
	}
	if !a.MeSHTerms[0].MajorTopic {
		t.Error("expected first MeSH term to be major topic")
	}
	if len(a.MeSHTerms[0].Qualifiers) != 2 {
		t.Errorf("expected 2 qualifiers for first MeSH term, got %d", len(a.MeSHTerms[0].Qualifiers))
	}

	// Publication types
	if len(a.PublicationTypes) != 3 {
		t.Fatalf("expected 3 publication types, got %d", len(a.PublicationTypes))
	}

	// Language
	if a.Language != "eng" {
		t.Errorf("expected language 'eng', got %q", a.Language)
	}
}

func TestFetch_SimpleAbstract(t *testing.T) {
	fixture := loadTestdata(t, "efetch_simple.xml")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	articles, err := c.Fetch(context.Background(), []string{"35999876"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}

	a := articles[0]

	// Unstructured abstract: single section without label
	if len(a.AbstractSections) != 1 {
		t.Fatalf("expected 1 abstract section, got %d", len(a.AbstractSections))
	}
	if a.AbstractSections[0].Label != "" {
		t.Errorf("expected empty label for unstructured abstract, got %q", a.AbstractSections[0].Label)
	}

	if a.Authors[0].LastName != "Smith" {
		t.Errorf("expected author 'Smith', got %q", a.Authors[0].LastName)
	}
	if a.DOI != "10.1523/JNEUROSCI.1234-22.2023" {
		t.Errorf("expected DOI, got %q", a.DOI)
	}
}

func TestFetch_MissingFields(t *testing.T) {
	fixture := loadTestdata(t, "efetch_missing_fields.xml")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	articles, err := c.Fetch(context.Background(), []string{"30000001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}

	a := articles[0]

	// No abstract
	if a.Abstract != "" {
		t.Errorf("expected empty abstract, got %q", a.Abstract)
	}

	// No authors
	if len(a.Authors) != 0 {
		t.Errorf("expected 0 authors, got %d", len(a.Authors))
	}

	// No DOI
	if a.DOI != "" {
		t.Errorf("expected empty DOI, got %q", a.DOI)
	}

	// No MeSH
	if len(a.MeSHTerms) != 0 {
		t.Errorf("expected 0 MeSH terms, got %d", len(a.MeSHTerms))
	}

	// Title should still be present
	if a.Title == "" {
		t.Error("expected non-empty title")
	}
}

func TestFetch_EmptyPMIDs(t *testing.T) {
	c := NewClient(WithAPIKey("test"))
	_, err := c.Fetch(context.Background(), nil)
	if err == nil {
		t.Error("expected error for empty PMIDs, got nil")
	}

	_, err = c.Fetch(context.Background(), []string{})
	if err == nil {
		t.Error("expected error for empty PMIDs slice, got nil")
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"))
	_, err := c.Fetch(context.Background(), []string{"12345"})
	if err == nil {
		t.Error("expected error for server error, got nil")
	}
}
