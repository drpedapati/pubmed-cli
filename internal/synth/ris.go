package synth

import (
	"fmt"
	"strings"
)

// GenerateRIS creates an RIS format string from references.
// RIS is a standardized format for importing into reference managers
// like EndNote, Zotero, Mendeley, etc.
func GenerateRIS(refs []Reference) string {
	var parts []string

	for _, ref := range refs {
		entry := generateRISEntry(ref)
		parts = append(parts, entry)
	}

	return strings.Join(parts, "\n")
}

func generateRISEntry(ref Reference) string {
	var lines []string

	// Type: Journal Article
	lines = append(lines, "TY  - JOUR")

	// Authors (AU tag for each)
	authors := parseAuthorsForRIS(ref.Authors)
	for _, author := range authors {
		lines = append(lines, fmt.Sprintf("AU  - %s", author))
	}

	// Title
	lines = append(lines, fmt.Sprintf("TI  - %s", ref.Title))

	// Journal
	if ref.Journal != "" {
		lines = append(lines, fmt.Sprintf("JO  - %s", ref.Journal))
	}

	// Year
	if ref.Year != "" {
		lines = append(lines, fmt.Sprintf("PY  - %s", ref.Year))
	}

	// DOI
	if ref.DOI != "" {
		lines = append(lines, fmt.Sprintf("DO  - %s", ref.DOI))
	}

	// PMID as accession number
	if ref.PMID != "" {
		lines = append(lines, fmt.Sprintf("AN  - %s", ref.PMID))
	}

	// Abstract
	if ref.Abstract != "" {
		// RIS abstract can be long, but some systems truncate
		abstract := ref.Abstract
		if len(abstract) > 5000 {
			abstract = abstract[:5000] + "..."
		}
		lines = append(lines, fmt.Sprintf("AB  - %s", abstract))
	}

	// Database
	lines = append(lines, "DB  - PubMed")

	// URL to PubMed
	if ref.PMID != "" {
		lines = append(lines, fmt.Sprintf("UR  - https://pubmed.ncbi.nlm.nih.gov/%s/", ref.PMID))
	}

	// End of record
	lines = append(lines, "ER  -")

	return strings.Join(lines, "\n")
}

// parseAuthorsForRIS converts "Smith, J. et al." back to individual authors
// if possible, or returns the string as-is
func parseAuthorsForRIS(authorStr string) []string {
	if authorStr == "" {
		return []string{"Unknown"}
	}

	// If it contains "et al.", we only have the first author
	if strings.Contains(authorStr, "et al.") {
		// Extract first author
		parts := strings.Split(authorStr, " et al.")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return []string{strings.TrimSpace(parts[0])}
		}
	}

	// If it contains " & ", split on that
	if strings.Contains(authorStr, " & ") {
		authors := strings.Split(authorStr, " & ")
		var result []string
		for _, a := range authors {
			a = strings.TrimSpace(a)
			if a != "" {
				result = append(result, a)
			}
		}
		return result
	}

	// Single author
	return []string{authorStr}
}

// WriteRISFile writes references to an RIS file.
func WriteRISFile(filename string, refs []Reference) error {
	// Implementation would write to file
	// For now, we'll handle this in the command layer
	return nil
}
