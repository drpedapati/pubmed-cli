package mesh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func loadTestdata(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", filename))
	if err != nil {
		t.Fatalf("failed to load testdata/%s: %v", filename, err)
	}
	return data
}

func TestLookup_Success(t *testing.T) {
	searchFixture := loadTestdata(t, "mesh_search.json")
	fetchFixture := loadTestdata(t, "mesh_fetch.txt")

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		path := r.URL.Path
		if path == "/esearch.fcgi" {
			q := r.URL.Query()
			if got := q.Get("db"); got != "mesh" {
				t.Errorf("expected db=mesh, got %q", got)
			}
			if got := q.Get("term"); got != "Fragile X Syndrome" {
				t.Errorf("expected term='Fragile X Syndrome', got %q", got)
			}
			w.Write(searchFixture)
		} else if path == "/efetch.fcgi" {
			q := r.URL.Query()
			if got := q.Get("db"); got != "mesh" {
				t.Errorf("expected db=mesh, got %q", got)
			}
			if got := q.Get("id"); got != "68005600" {
				t.Errorf("expected id=68005600, got %q", got)
			}
			w.Write(fetchFixture)
		} else {
			t.Errorf("unexpected path: %s", path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "pubmed-cli", "test@example.com")
	record, err := c.Lookup(context.Background(), "Fragile X Syndrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if record.UI != "D005600" {
		t.Errorf("expected UI 'D005600', got %q", record.UI)
	}
	if record.Name != "Fragile X Syndrome" {
		t.Errorf("expected name 'Fragile X Syndrome', got %q", record.Name)
	}
	if record.ScopeNote == "" {
		t.Error("expected non-empty scope note")
	}
	if len(record.TreeNumbers) == 0 {
		t.Error("expected at least one tree number")
	}
	if record.TreeNumbers[0] != "C10.597.606.360.320.322" {
		t.Errorf("expected first tree number 'C10.597.606.360.320.322', got %q", record.TreeNumbers[0])
	}
	if len(record.EntryTerms) == 0 {
		t.Error("expected at least one entry term")
	}

	// Check known entry terms
	found := false
	for _, e := range record.EntryTerms {
		if e == "FXS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected entry term 'FXS' in entry terms")
	}

	if record.Annotation == "" {
		t.Error("expected non-empty annotation")
	}
}

func TestLookup_NotFound(t *testing.T) {
	emptySearch := `{"header":{"type":"esearch","version":"0.3"},"esearchresult":{"count":"0","retmax":"20","retstart":"0","idlist":[]}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(emptySearch))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "pubmed-cli", "test@example.com")
	_, err := c.Lookup(context.Background(), "nonexistent_mesh_term_xyz")
	if err == nil {
		t.Error("expected error for not found term, got nil")
	}
}

func TestLookup_EmptyTerm(t *testing.T) {
	c := NewClient("http://example.com", "key", "tool", "email")
	_, err := c.Lookup(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty term, got nil")
	}
}

func TestParseMeSHRecord(t *testing.T) {
	data := loadTestdata(t, "mesh_fetch.txt")
	record := parseMeSHRecord(string(data))

	if record.UI != "D005600" {
		t.Errorf("expected UI 'D005600', got %q", record.UI)
	}
	if record.Name != "Fragile X Syndrome" {
		t.Errorf("expected name 'Fragile X Syndrome', got %q", record.Name)
	}
	if len(record.TreeNumbers) != 3 {
		t.Errorf("expected 3 tree numbers, got %d", len(record.TreeNumbers))
	}
	if len(record.EntryTerms) != 5 {
		t.Errorf("expected 5 entry terms, got %d", len(record.EntryTerms))
	}
}
