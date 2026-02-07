package synth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateRIS creates an RIS format string from references.
// RIS is a standardized format for importing into reference managers
// like EndNote, Zotero, Mendeley, etc.
func GenerateRIS(refs []Reference) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		parts = append(parts, generateRISEntry(ref))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n") + "\n"
}

func generateRISEntry(ref Reference) string {
	var lines []string

	// Type: Journal Article
	lines = append(lines, "TY  - JOUR")

	// Authors (AU tag for each)
	for _, author := range parseAuthorsForRIS(ref.Authors) {
		lines = append(lines, fmt.Sprintf("AU  - %s", sanitizeRIS(author)))
	}

	// Title
	lines = append(lines, fmt.Sprintf("TI  - %s", sanitizeRIS(ref.Title)))

	// Journal
	if ref.Journal != "" {
		lines = append(lines, fmt.Sprintf("JO  - %s", sanitizeRIS(ref.Journal)))
	}

	// Year
	if ref.Year != "" {
		lines = append(lines, fmt.Sprintf("PY  - %s", sanitizeRIS(ref.Year)))
	}

	// DOI
	if ref.DOI != "" {
		lines = append(lines, fmt.Sprintf("DO  - %s", sanitizeRIS(ref.DOI)))
	}

	// PMID as accession number
	if ref.PMID != "" {
		lines = append(lines, fmt.Sprintf("AN  - %s", sanitizeRIS(ref.PMID)))
	}

	// Abstract
	if ref.Abstract != "" {
		abstract := ref.Abstract
		if len([]rune(abstract)) > 5000 {
			abstract = string([]rune(abstract)[:5000]) + "..."
		}
		lines = append(lines, fmt.Sprintf("AB  - %s", sanitizeRIS(abstract)))
	}

	// Database
	lines = append(lines, "DB  - PubMed")

	// URL to PubMed
	if ref.PMID != "" {
		lines = append(lines, fmt.Sprintf("UR  - https://pubmed.ncbi.nlm.nih.gov/%s/", sanitizeRIS(ref.PMID)))
	}

	// End of record
	lines = append(lines, "ER  -")
	return strings.Join(lines, "\n")
}

// sanitizeRIS replaces newlines/tabs with spaces so we don't accidentally break the RIS line format.
func sanitizeRIS(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.TrimSpace(s)
}

// parseAuthorsForRIS converts "Smith, J. et al." back to individual authors
// if possible, or returns the string as-is.
func parseAuthorsForRIS(authorStr string) []string {
	authorStr = strings.TrimSpace(authorStr)
	if authorStr == "" {
		return []string{"Unknown"}
	}

	// If it contains "et al.", we only have the first author.
	if strings.Contains(authorStr, "et al.") {
		parts := strings.Split(authorStr, " et al.")
		if len(parts) > 0 {
			first := strings.TrimSpace(parts[0])
			if first != "" {
				return []string{first}
			}
		}
	}

	// If it contains " & ", split on that.
	if strings.Contains(authorStr, " & ") {
		authors := strings.Split(authorStr, " & ")
		result := make([]string, 0, len(authors))
		for _, a := range authors {
			a = strings.TrimSpace(a)
			if a != "" {
				result = append(result, a)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// Single author or unknown formatting.
	return []string{authorStr}
}

// WriteRISFile writes references to an RIS file.
func WriteRISFile(filename string, refs []Reference) error {
	if filename == "" {
		return fmt.Errorf("filename is required")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create RIS output dir: %w", err)
	}
	return os.WriteFile(filename, []byte(GenerateRIS(refs)), 0o644)
}
