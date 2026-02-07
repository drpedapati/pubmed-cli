package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/henrybloomingdale/pubmed-cli/internal/llm"
	"github.com/henrybloomingdale/pubmed-cli/internal/synth"
	"github.com/spf13/cobra"
)

var (
	synthFlagPapers    int
	synthFlagSearch    int
	synthFlagRelevance int
	synthFlagWords     int
	synthFlagDocx      string
	synthFlagRIS       string
	synthFlagPMID      string
	synthFlagModel     string
	synthFlagBaseURL   string
	synthFlagClaude    bool
	synthFlagMd        bool
)

func init() {
	synthCmd.Flags().IntVar(&synthFlagPapers, "papers", 5, "Number of papers to include in synthesis")
	synthCmd.Flags().IntVar(&synthFlagSearch, "search", 30, "Number of papers to search before filtering")
	synthCmd.Flags().IntVar(&synthFlagRelevance, "relevance", 7, "Minimum relevance score (1-10)")
	synthCmd.Flags().IntVar(&synthFlagWords, "words", 250, "Target word count")
	synthCmd.Flags().StringVar(&synthFlagDocx, "docx", "", "Output Word document")
	synthCmd.Flags().StringVar(&synthFlagRIS, "ris", "", "Output RIS file for reference managers")
	synthCmd.Flags().StringVar(&synthFlagPMID, "pmid", "", "Deep dive on single paper by PMID")
	synthCmd.Flags().StringVar(&synthFlagModel, "model", "", "LLM model (default: gpt-4o or LLM_MODEL env)")
	synthCmd.Flags().StringVar(&synthFlagBaseURL, "llm-url", "", "LLM API base URL")
	synthCmd.Flags().BoolVar(&synthFlagClaude, "claude", false, "Use Claude CLI (no API key needed)")
	synthCmd.Flags().BoolVar(&synthFlagMd, "md", false, "Output markdown to stdout (default if no --docx)")

	rootCmd.AddCommand(synthCmd)
}

var synthCmd = &cobra.Command{
	Use:   "synth <question>",
	Short: "Synthesize literature on a topic with citations",
	Long: `Search PubMed, filter by relevance, and synthesize findings into paragraphs with citations.

Examples:
  # Basic synthesis (markdown output)
  pubmed synth "SGLT-2 inhibitors in liver fibrosis"

  # Word document + RIS file
  pubmed synth "CBT for pediatric anxiety" --docx review.docx --ris refs.ris

  # More papers, longer output
  pubmed synth "autism biomarkers" --papers 10 --words 500

  # Single paper deep dive
  pubmed synth --pmid 41234567 --words 400

  # JSON for agents
  pubmed synth "treatments for fragile x" --json

Environment:
  LLM_API_KEY   - API key for LLM
  LLM_BASE_URL  - Base URL for OpenAI-compatible API
  LLM_MODEL     - Model name (default: gpt-4o)`,
	Args: cobra.ArbitraryArgs,
	RunE: runSynth,
}

func runSynth(cmd *cobra.Command, args []string) error {
	// Validate args.
	pmid := strings.TrimSpace(synthFlagPMID)
	if pmid == "" && len(args) == 0 {
		return fmt.Errorf("provide a question or use --pmid for single paper")
	}
	if pmid != "" && len(args) > 0 {
		return fmt.Errorf("provide either a question or --pmid, not both")
	}

	if synthFlagPapers < 1 {
		return fmt.Errorf("--papers must be >= 1")
	}
	if synthFlagSearch < 1 {
		return fmt.Errorf("--search must be >= 1")
	}
	if synthFlagWords < 1 {
		return fmt.Errorf("--words must be >= 1")
	}
	if synthFlagRelevance < 1 || synthFlagRelevance > 10 {
		return fmt.Errorf("--relevance must be 1-10")
	}
	if synthFlagPapers > synthFlagSearch {
		// Avoid accidentally filtering down to fewer than requested.
		synthFlagSearch = synthFlagPapers
	}

	// Build LLM client.
	var llmClient synth.LLMClient
	var err error
	if synthFlagClaude {
		llmClient, err = llm.NewClaudeClient(synthFlagModel)
		if err != nil {
			return fmt.Errorf("claude setup: %w", err)
		}
	} else {
		var llmOpts []llm.Option
		if synthFlagModel != "" {
			llmOpts = append(llmOpts, llm.WithModel(synthFlagModel))
		}
		if synthFlagBaseURL != "" {
			llmOpts = append(llmOpts, llm.WithBaseURL(synthFlagBaseURL))
		}
		llmClient = llm.NewClient(llmOpts...)
	}

	// Build config.
	cfg := synth.DefaultConfig()
	cfg.PapersToUse = synthFlagPapers
	cfg.PapersToSearch = synthFlagSearch
	cfg.RelevanceThreshold = synthFlagRelevance
	cfg.TargetWords = synthFlagWords

	// Build engine.
	engine := synth.NewEngine(llmClient, newEutilsClient(), cfg)

	// Run synthesis.
	var result *synth.Result
	ctx := cmd.Context()
	if pmid != "" {
		result, err = engine.SynthesizePMID(ctx, pmid)
	} else {
		question := strings.TrimSpace(strings.Join(args, " "))
		result, err = engine.Synthesize(ctx, question)
	}
	if err != nil {
		return fmt.Errorf("synthesize: %w", err)
	}
	if result == nil {
		return errors.New("synthesis returned nil result")
	}

	// Write RIS file if requested.
	if synthFlagRIS != "" {
		if err := os.MkdirAll(filepath.Dir(synthFlagRIS), 0o755); err != nil {
			return fmt.Errorf("create RIS dir: %w", err)
		}
		if err := os.WriteFile(synthFlagRIS, []byte(result.RIS), 0o644); err != nil {
			return fmt.Errorf("write RIS file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Wrote %s (%d references)\n", synthFlagRIS, len(result.References))
	}

	// Write DOCX if requested.
	if synthFlagDocx != "" {
		if err := writeDocx(ctx, synthFlagDocx, result); err != nil {
			var w *docxFallbackWarning
			if errors.As(err, &w) {
				fmt.Fprintln(os.Stderr, w.Error())
			} else {
				return fmt.Errorf("write DOCX: %w", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "✓ Wrote %s\n", synthFlagDocx)
		}
	}

	// Output.
	if flagJSON {
		return outputJSON(result)
	}
	// If the user requested a file output, default to being quiet unless --md is set.
	if synthFlagDocx != "" && !synthFlagMd {
		return nil
	}
	return outputMarkdown(result)
}

func outputJSON(result *synth.Result) error {
	if result == nil {
		return errors.New("result is nil")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputMarkdown(result *synth.Result) error {
	if result == nil {
		return errors.New("result is nil")
	}

	var sb strings.Builder

	// Header.
	sb.WriteString(fmt.Sprintf("# %s\n\n", result.Question))

	// Stats.
	sb.WriteString(fmt.Sprintf("*Searched %d papers, scored %d, used %d*\n\n",
		result.PapersSearched, result.PapersScored, result.PapersUsed))

	// Synthesis.
	sb.WriteString("## Synthesis\n\n")
	sb.WriteString(result.Synthesis)
	sb.WriteString("\n\n")

	// References.
	sb.WriteString("## References\n\n")
	for i, ref := range result.References {
		sb.WriteString(fmt.Sprintf("%d. %s (relevance: %d/10) [PMID: %s]\n",
			i+1, ref.CitationAPA, ref.RelevanceScore, ref.PMID))
	}

	// Token usage.
	sb.WriteString(fmt.Sprintf("\n---\n*Tokens: ~%d input, ~%d output, ~%d total*\n",
		result.Tokens.Input, result.Tokens.Output, result.Tokens.Total))

	_, err := fmt.Fprint(os.Stdout, sb.String())
	return err
}

type docxFallbackWarning struct {
	DocxPath     string
	MarkdownPath string
	Cause        error
}

func (w *docxFallbackWarning) Error() string {
	return fmt.Sprintf("DOCX conversion failed; wrote markdown instead: %s (requested DOCX: %s): %v", w.MarkdownPath, w.DocxPath, w.Cause)
}

func (w *docxFallbackWarning) Unwrap() error { return w.Cause }

// writeDocx creates a Word document with synthesis and references.
// Implementation strategy: write a temporary markdown file and convert via pandoc.
func writeDocx(ctx context.Context, filename string, result *synth.Result) error {
	// convertToDocx accepts a context for cancellation.
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return errors.New("filename is required")
	}
	if strings.HasSuffix(filename, "/") || strings.HasSuffix(filename, "\\") {
		return errors.New("filename must be a file path, not a directory")
	}
	if result == nil {
		return errors.New("result is nil")
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.CreateTemp("", "pubmed-synth-*.md")
	if err != nil {
		return fmt.Errorf("create temp markdown: %w", err)
	}
	tmpMD := f.Name()
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp markdown: %w", err)
	}
	defer os.Remove(tmpMD) // best-effort cleanup

	if err := saveMarkdownFile(tmpMD, result); err != nil {
		return fmt.Errorf("write temp markdown: %w", err)
	}
	if err := convertToDocxContext(ctx, tmpMD, filename); err != nil {
		mdOut := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".md"
		if err2 := saveMarkdownFile(mdOut, result); err2 != nil {
			return fmt.Errorf("pandoc conversion failed (%w); additionally failed to write markdown fallback %q: %w", err, mdOut, err2)
		}
		return &docxFallbackWarning{DocxPath: filename, MarkdownPath: mdOut, Cause: err}
	}
	return nil
}
