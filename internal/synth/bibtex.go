package synth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// GenerateBibTeX creates a BibTeX format string from references.
//
// Output is a sequence of @article entries separated by blank lines.
func GenerateBibTeX(refs []Reference) string {
	if len(refs) == 0 {
		return ""
	}

	keys := generateBibTeXCitationKeys(refs)
	parts := make([]string, 0, len(refs))
	for i, ref := range refs {
		key := keys[i]
		if key == "" {
			key = fmt.Sprintf("Ref%d", i+1)
		}
		parts = append(parts, generateBibTeXEntry(key, ref))
	}
	return strings.Join(parts, "\n\n") + "\n"
}

func generateBibTeXEntry(key string, ref Reference) string {
	lines := make([]string, 0, 10)
	lines = append(lines, fmt.Sprintf("@article{%s,", key))

	authors := bibtexAuthors(ref)
	if authors != "" {
		lines = append(lines, fmt.Sprintf("  author = {%s},", latexEscapeBibTeX(authors)))
	}
	if title := strings.TrimSpace(ref.Title); title != "" {
		lines = append(lines, fmt.Sprintf("  title = {%s},", latexEscapeBibTeX(title)))
	}
	if journal := strings.TrimSpace(ref.Journal); journal != "" {
		lines = append(lines, fmt.Sprintf("  journal = {%s},", latexEscapeBibTeX(journal)))
	}
	if year := strings.TrimSpace(ref.Year); year != "" {
		lines = append(lines, fmt.Sprintf("  year = {%s},", latexEscapeBibTeX(year)))
	}
	if doi := strings.TrimSpace(ref.DOI); doi != "" {
		lines = append(lines, fmt.Sprintf("  doi = {%s},", latexEscapeBibTeX(doi)))
	}
	if pmid := strings.TrimSpace(ref.PMID); pmid != "" {
		lines = append(lines, fmt.Sprintf("  pmid = {%s},", latexEscapeBibTeX(pmid)))
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func bibtexAuthors(ref Reference) string {
	if len(ref.AuthorsList) > 0 {
		return strings.Join(ref.AuthorsList, " and ")
	}
	authors := parseAuthorsForBibTeX(ref.Authors)
	if len(authors) == 0 {
		return ""
	}
	return strings.Join(authors, " and ")
}

// (intentionally no bibtexAuthorName helper; use bibtexAuthorFromName)

// parseAuthorsForBibTeX converts the human-readable author string into a list of BibTeX authors.
// It handles:
//   - "Smith et al." (returns just the first author)
//   - "John Smith & Jane Jones" (returns two authors)
func parseAuthorsForBibTeX(authorStr string) []string {
	authorStr = strings.TrimSpace(authorStr)
	if authorStr == "" {
		return nil
	}

	// If it contains "et al.", we only have the first author.
	if strings.Contains(authorStr, "et al.") {
		parts := strings.Split(authorStr, " et al.")
		if len(parts) > 0 {
			first := strings.TrimSpace(parts[0])
			if first != "" {
				return []string{bibtexAuthorFromName(first)}
			}
		}
	}

	// If it contains " & ", split on that.
	if strings.Contains(authorStr, " & ") {
		authors := strings.Split(authorStr, " & ")
		out := make([]string, 0, len(authors))
		for _, a := range authors {
			a = strings.TrimSpace(a)
			if a == "" {
				continue
			}
			out = append(out, bibtexAuthorFromName(a))
		}
		return out
	}

	return []string{bibtexAuthorFromName(authorStr)}
}

func bibtexAuthorFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Unknown"
	}
	// Already in "Last, First" form.
	if strings.Contains(name, ",") {
		return name
	}
	fields := strings.Fields(name)
	if len(fields) == 1 {
		return fields[0]
	}
	last := fields[len(fields)-1]
	fore := strings.Join(fields[:len(fields)-1], " ")
	return fmt.Sprintf("%s, %s", last, fore)
}

func generateBibTeXCitationKeys(refs []Reference) []string {
	keys := make([]string, len(refs))
	seen := make(map[string]int, len(refs))
	for i, ref := range refs {
		base := bibtexCitationKeyBase(ref)
		if base == "" {
			base = fmt.Sprintf("Ref%d", i+1)
		}
		dup := seen[base]
		seen[base] = dup + 1
		if dup == 0 {
			keys[i] = base
			continue
		}
		keys[i] = base + alphaSuffix(dup)
	}
	return keys
}

func bibtexCitationKeyBase(ref Reference) string {
	firstAuthor := ""
	if len(ref.AuthorsList) > 0 {
		firstAuthor = ref.AuthorsList[0]
	} else {
		// Try to pull the first author from the human-readable author string.
		authorStr := strings.TrimSpace(ref.Authors)
		firstAuthor = authorStr
		if strings.Contains(authorStr, " & ") {
			firstAuthor = strings.TrimSpace(strings.Split(authorStr, " & ")[0])
		} else if strings.Contains(authorStr, "et al.") {
			firstAuthor = strings.TrimSpace(strings.Split(authorStr, " et al.")[0])
		}
	}

	authorToken := bibtexKeyAuthorToken(firstAuthor)
	year := yearForBibTeXKey(ref.Year)
	base := sanitizeBibTeXKey(authorToken + year)
	if base == "" {
		base = "Ref" + year
	}
	return base
}

func bibtexKeyAuthorToken(author string) string {
	author = strings.TrimSpace(author)
	if author == "" {
		return "Unknown"
	}
	// If "Last, First" use the part before the comma.
	if idx := strings.Index(author, ","); idx >= 0 {
		author = author[:idx]
	}
	fields := strings.Fields(author)
	if len(fields) == 0 {
		return "Unknown"
	}
	return fields[len(fields)-1]
}

func yearForBibTeXKey(year string) string {
	year = strings.TrimSpace(year)
	if year == "" {
		return "nd"
	}
	// Find first 4 consecutive digits.
	r := []rune(year)
	for i := 0; i+3 < len(r); i++ {
		if unicode.IsDigit(r[i]) && unicode.IsDigit(r[i+1]) && unicode.IsDigit(r[i+2]) && unicode.IsDigit(r[i+3]) {
			return string(r[i : i+4])
		}
	}
	return "nd"
}

func sanitizeBibTeXKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		// Restrict to ASCII letters/digits to avoid surprising BibTeX behavior.
		if r > 127 {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return ""
	}
	// BibTeX keys should not start with a digit.
	if out[0] >= '0' && out[0] <= '9' {
		out = "Ref" + out
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

// alphaSuffix returns a, b, ..., z, aa, ab, ... for 1, 2, ..., 26, 27, 28, ...
func alphaSuffix(n int) string {
	if n <= 0 {
		return ""
	}
	n-- // 1 => a
	var out []byte
	for {
		out = append([]byte{byte('a' + (n % 26))}, out...)
		n = n/26 - 1
		if n < 0 {
			break
		}
	}
	return string(out)
}

// latexEscapeBibTeX escapes a minimal set of LaTeX-special characters.
//
// This is intentionally conservative; BibTeX consumers vary in strictness.
func latexEscapeBibTeX(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)

	// Order matters: escape backslash first.
	repl := strings.NewReplacer(
		"\\", "\\\\",
		"{", "\\{",
		"}", "\\}",
		"%", "\\%",
		"&", "\\&",
		"$", "\\$",
		"#", "\\#",
		"_", "\\_",
		"~", "\\~{}",
		"^", "\\^{}",
	)
	return repl.Replace(s)
}

// WriteBibTeXFile writes references to a BibTeX file.
func WriteBibTeXFile(filename string, refs []Reference) error {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return fmt.Errorf("filename is required")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create BibTeX output dir: %w", err)
	}
	return os.WriteFile(filename, []byte(GenerateBibTeX(refs)), 0o644)
}
