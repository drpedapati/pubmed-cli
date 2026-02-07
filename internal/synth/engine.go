// Package synth provides literature synthesis from PubMed searches.
package synth

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/henrybloomingdale/pubmed-cli/internal/eutils"
)

// LLMClient is the interface for LLM completions.
type LLMClient interface {
	Complete(ctx context.Context, prompt string, maxTokens int) (string, error)
}

// Config controls synthesis behavior.
type Config struct {
	PapersToUse       int    // How many papers to include (default: 5)
	PapersToSearch    int    // How many to search before filtering (default: 30)
	RelevanceThreshold int   // Minimum relevance score 1-10 (default: 7)
	TargetWords       int    // Target word count (default: 250)
	CitationStyle     string // Citation style (default: apa)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PapersToUse:       5,
		PapersToSearch:    30,
		RelevanceThreshold: 7,
		TargetWords:       250,
		CitationStyle:     "apa",
	}
}

// ScoredPaper holds a paper with its relevance score.
type ScoredPaper struct {
	Article        eutils.Article
	RelevanceScore int
}

// Reference holds citation information.
type Reference struct {
	Key            string `json:"key"`
	PMID           string `json:"pmid"`
	CitationAPA    string `json:"citation_apa"`
	RelevanceScore int    `json:"relevance_score"`
	DOI            string `json:"doi,omitempty"`
	Title          string `json:"title"`
	Abstract       string `json:"abstract,omitempty"`
	Year           string `json:"year"`
	Authors        string `json:"authors"`
	Journal        string `json:"journal"`
}

// Result contains the synthesis output.
type Result struct {
	Question       string      `json:"question"`
	Synthesis      string      `json:"synthesis"`
	PapersSearched int         `json:"papers_searched"`
	PapersScored   int         `json:"papers_scored"`
	PapersUsed     int         `json:"papers_used"`
	References     []Reference `json:"references"`
	RIS            string      `json:"ris,omitempty"`
	Tokens         TokenUsage  `json:"tokens"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

// Engine performs literature synthesis.
type Engine struct {
	llm    LLMClient
	eutils *eutils.Client
	cfg    Config
}

// NewEngine creates a new synthesis engine.
func NewEngine(llmClient LLMClient, eutilsClient *eutils.Client, cfg Config) *Engine {
	return &Engine{
		llm:    llmClient,
		eutils: eutilsClient,
		cfg:    cfg,
	}
}

// Synthesize performs the full synthesis workflow.
func (e *Engine) Synthesize(ctx context.Context, question string) (*Result, error) {
	result := &Result{
		Question: question,
	}

	// Step 1: Search PubMed
	searchResult, err := e.eutils.Search(ctx, question, &eutils.SearchOptions{
		Limit: e.cfg.PapersToSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	result.PapersSearched = len(searchResult.IDs)

	if len(searchResult.IDs) == 0 {
		return nil, fmt.Errorf("no papers found for query: %s", question)
	}

	// Step 2: Fetch articles
	articles, err := e.eutils.Fetch(ctx, searchResult.IDs)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// Step 3: Score relevance
	scored, tokensUsed, err := e.scoreRelevance(ctx, question, articles)
	if err != nil {
		return nil, fmt.Errorf("relevance scoring: %w", err)
	}
	result.PapersScored = len(scored)
	result.Tokens.Input += tokensUsed

	// Step 4: Filter and sort by relevance
	var relevant []ScoredPaper
	for _, sp := range scored {
		if sp.RelevanceScore >= e.cfg.RelevanceThreshold {
			relevant = append(relevant, sp)
		}
	}

	// Sort by relevance descending
	sort.Slice(relevant, func(i, j int) bool {
		return relevant[i].RelevanceScore > relevant[j].RelevanceScore
	})

	// Take top N
	if len(relevant) > e.cfg.PapersToUse {
		relevant = relevant[:e.cfg.PapersToUse]
	}

	if len(relevant) == 0 {
		return nil, fmt.Errorf("no papers met relevance threshold (%d) for: %s", e.cfg.RelevanceThreshold, question)
	}

	result.PapersUsed = len(relevant)

	// Step 5: Build references
	for i, sp := range relevant {
		ref := buildReference(sp.Article, i+1, sp.RelevanceScore)
		result.References = append(result.References, ref)
	}

	// Step 6: Generate synthesis
	synthesis, tokensUsed, err := e.generateSynthesis(ctx, question, relevant)
	if err != nil {
		return nil, fmt.Errorf("synthesis: %w", err)
	}
	result.Synthesis = synthesis
	result.Tokens.Output += tokensUsed

	// Step 7: Generate RIS
	result.RIS = GenerateRIS(result.References)

	// Estimate total tokens (rough)
	result.Tokens.Total = result.Tokens.Input + result.Tokens.Output

	return result, nil
}

// SynthesizePMID performs deep dive on a single paper.
func (e *Engine) SynthesizePMID(ctx context.Context, pmid string) (*Result, error) {
	result := &Result{
		Question:       fmt.Sprintf("Deep dive: PMID %s", pmid),
		PapersSearched: 1,
		PapersScored:   1,
		PapersUsed:     1,
	}

	// Fetch the article
	articles, err := e.eutils.Fetch(ctx, []string{pmid})
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	if len(articles) == 0 {
		return nil, fmt.Errorf("article not found: %s", pmid)
	}

	article := articles[0]
	ref := buildReference(article, 1, 10)
	result.References = []Reference{ref}

	// Generate deep dive summary
	prompt := fmt.Sprintf(`Summarize this research paper in approximately %d words. Include:
- Main objective/question
- Key methods
- Primary findings
- Implications/conclusions

Title: %s

Abstract:
%s

Write a cohesive summary paragraph. Cite as (Author et al., %s).`,
		e.cfg.TargetWords, article.Title, article.Abstract, article.Year)

	synthesis, err := e.llm.Complete(ctx, prompt, e.cfg.TargetWords*2)
	if err != nil {
		return nil, fmt.Errorf("synthesis: %w", err)
	}
	result.Synthesis = strings.TrimSpace(synthesis)

	// Estimate tokens
	result.Tokens.Input = len(prompt) / 4
	result.Tokens.Output = len(synthesis) / 4
	result.Tokens.Total = result.Tokens.Input + result.Tokens.Output

	// Generate RIS
	result.RIS = GenerateRIS(result.References)

	return result, nil
}

func (e *Engine) scoreRelevance(ctx context.Context, question string, articles []eutils.Article) ([]ScoredPaper, int, error) {
	var scored []ScoredPaper
	totalTokens := 0

	for _, article := range articles {
		score, tokens, err := scoreArticleRelevance(ctx, e.llm, question, &article)
		if err != nil {
			// Log but continue - don't fail entire synthesis for one bad score
			score = 5 // neutral score
		}
		totalTokens += tokens
		scored = append(scored, ScoredPaper{
			Article:        article,
			RelevanceScore: score,
		})
	}

	return scored, totalTokens, nil
}

func (e *Engine) generateSynthesis(ctx context.Context, question string, papers []ScoredPaper) (string, int, error) {
	// Build context from papers
	var contextParts []string
	var citeKeys []string

	for i, sp := range papers {
		// Create citation key
		firstAuthor := "Unknown"
		if len(sp.Article.Authors) > 0 {
			parts := strings.Split(sp.Article.Authors[0].FullName(), " ")
			if len(parts) > 0 {
				firstAuthor = parts[len(parts)-1] // Last name
			}
		}
		citeKey := fmt.Sprintf("%s et al., %s", firstAuthor, sp.Article.Year)
		citeKeys = append(citeKeys, citeKey)

		contextParts = append(contextParts, fmt.Sprintf(`[%d] %s (%s)
Title: %s
Abstract: %s
`, i+1, citeKey, sp.Article.PMID, sp.Article.Title, sp.Article.Abstract))
	}

	prompt := fmt.Sprintf(`You are a scientific writer. Synthesize the following research papers to answer this question:

Question: %s

Papers:
%s

Write a synthesis of approximately %d words that:
1. Directly addresses the question
2. Integrates findings across papers
3. Uses inline citations like (Smith et al., 2024)
4. Maintains academic tone
5. Notes any conflicting findings

Available citations: %s

Write the synthesis:`,
		question,
		strings.Join(contextParts, "\n---\n"),
		e.cfg.TargetWords,
		strings.Join(citeKeys, "; "))

	synthesis, err := e.llm.Complete(ctx, prompt, e.cfg.TargetWords*3)
	if err != nil {
		return "", 0, err
	}

	// Estimate tokens
	tokensUsed := len(synthesis) / 4

	return strings.TrimSpace(synthesis), tokensUsed, nil
}

func buildReference(article eutils.Article, num int, relevance int) Reference {
	// Build author string
	var authorStr string
	if len(article.Authors) > 0 {
		if len(article.Authors) == 1 {
			authorStr = article.Authors[0].FullName()
		} else if len(article.Authors) == 2 {
			authorStr = article.Authors[0].FullName() + " & " + article.Authors[1].FullName()
		} else {
			authorStr = article.Authors[0].FullName() + " et al."
		}
	}

	// Build APA citation
	apa := formatAPA(article)

	// Build citation key
	key := fmt.Sprintf("%d", num)
	if len(article.Authors) > 0 {
		parts := strings.Split(article.Authors[0].FullName(), " ")
		lastName := parts[len(parts)-1]
		key = fmt.Sprintf("%s %s", lastName, article.Year)
	}

	return Reference{
		Key:            key,
		PMID:           article.PMID,
		CitationAPA:    apa,
		RelevanceScore: relevance,
		DOI:            article.DOI,
		Title:          article.Title,
		Abstract:       article.Abstract,
		Year:           article.Year,
		Authors:        authorStr,
		Journal:        article.Journal,
	}
}

func formatAPA(article eutils.Article) string {
	// Build author list for APA
	var authors string
	if len(article.Authors) == 0 {
		authors = "Unknown"
	} else if len(article.Authors) == 1 {
		a := article.Authors[0]
		authors = fmt.Sprintf("%s, %s.", a.LastName, initials(a.ForeName))
	} else if len(article.Authors) <= 7 {
		var parts []string
		for i, a := range article.Authors {
			if i == len(article.Authors)-1 {
				parts = append(parts, fmt.Sprintf("& %s, %s.", a.LastName, initials(a.ForeName)))
			} else {
				parts = append(parts, fmt.Sprintf("%s, %s.", a.LastName, initials(a.ForeName)))
			}
		}
		authors = strings.Join(parts, ", ")
	} else {
		// More than 7 authors: first 6, ..., last
		var parts []string
		for i := 0; i < 6; i++ {
			a := article.Authors[i]
			parts = append(parts, fmt.Sprintf("%s, %s.", a.LastName, initials(a.ForeName)))
		}
		last := article.Authors[len(article.Authors)-1]
		parts = append(parts, "...")
		parts = append(parts, fmt.Sprintf("& %s, %s.", last.LastName, initials(last.ForeName)))
		authors = strings.Join(parts, ", ")
	}

	// Format: Authors (Year). Title. Journal.
	citation := fmt.Sprintf("%s (%s). %s. %s.",
		authors,
		article.Year,
		article.Title,
		article.Journal)

	if article.DOI != "" {
		citation += fmt.Sprintf(" https://doi.org/%s", article.DOI)
	}

	return citation
}

func initials(foreName string) string {
	parts := strings.Fields(foreName)
	var inits []string
	for _, p := range parts {
		if len(p) > 0 {
			inits = append(inits, string(p[0]))
		}
	}
	return strings.Join(inits, ". ")
}
