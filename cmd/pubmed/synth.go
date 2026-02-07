package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	synthCmd.Flags().BoolVar(&synthFlagMd, "md", false, "Output markdown (default if no --docx)")

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
	Args: cobra.MaximumNArgs(1),
	RunE: runSynth,
}

func runSynth(cmd *cobra.Command, args []string) error {
	// Validate args
	if synthFlagPMID == "" && len(args) == 0 {
		return fmt.Errorf("provide a question or use --pmid for single paper")
	}

	// Build LLM client
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

	// Build config
	cfg := synth.DefaultConfig()
	cfg.PapersToUse = synthFlagPapers
	cfg.PapersToSearch = synthFlagSearch
	cfg.RelevanceThreshold = synthFlagRelevance
	cfg.TargetWords = synthFlagWords

	// Build engine
	engine := synth.NewEngine(llmClient, newEutilsClient(), cfg)

	// Run synthesis
	var result *synth.Result
	ctx := cmd.Context()

	if synthFlagPMID != "" {
		result, err = engine.SynthesizePMID(ctx, synthFlagPMID)
	} else {
		question := strings.Join(args, " ")
		result, err = engine.Synthesize(ctx, question)
	}

	if err != nil {
		return err
	}

	// Write RIS file if requested
	if synthFlagRIS != "" {
		if err := os.WriteFile(synthFlagRIS, []byte(result.RIS), 0644); err != nil {
			return fmt.Errorf("write RIS file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Wrote %s (%d references)\n", synthFlagRIS, len(result.References))
	}

	// Write DOCX if requested
	if synthFlagDocx != "" {
		if err := writeDocx(synthFlagDocx, result); err != nil {
			return fmt.Errorf("write DOCX: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Wrote %s\n", synthFlagDocx)
	}

	// Output
	if flagJSON {
		return outputJSON(result)
	}

	// Default to markdown
	return outputMarkdown(result)
}

func outputJSON(result *synth.Result) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputMarkdown(result *synth.Result) error {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# %s\n\n", result.Question))

	// Stats
	sb.WriteString(fmt.Sprintf("*Searched %d papers, scored %d, used %d (relevance ≥ threshold)*\n\n",
		result.PapersSearched, result.PapersScored, result.PapersUsed))

	// Synthesis
	sb.WriteString("## Synthesis\n\n")
	sb.WriteString(result.Synthesis)
	sb.WriteString("\n\n")

	// References
	sb.WriteString("## References\n\n")
	for i, ref := range result.References {
		sb.WriteString(fmt.Sprintf("%d. %s (relevance: %d/10) [PMID: %s]\n",
			i+1, ref.CitationAPA, ref.RelevanceScore, ref.PMID))
	}

	// Token usage
	sb.WriteString(fmt.Sprintf("\n---\n*Tokens: ~%d input, ~%d output, ~%d total*\n",
		result.Tokens.Input, result.Tokens.Output, result.Tokens.Total))

	fmt.Println(sb.String())
	return nil
}

// writeDocx creates a Word document with synthesis and references.
func writeDocx(filename string, result *synth.Result) error {
	// For now, we'll use a simple approach with the docx package
	// This is a placeholder - we'll implement properly
	
	// Create simple DOCX content
	content := fmt.Sprintf(`%s

%s

References

%s`,
		result.Question,
		result.Synthesis,
		formatReferencesForDocx(result.References))

	// Write as markdown for now (proper DOCX needs external package)
	// TODO: Use proper DOCX library
	mdFile := strings.TrimSuffix(filename, ".docx") + ".md"
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		return err
	}
	
	// Try to convert with pandoc if available
	if _, err := os.Stat("/opt/homebrew/bin/pandoc"); err == nil {
		ctx := context.Background()
		return convertWithPandoc(ctx, mdFile, filename)
	}

	fmt.Fprintf(os.Stderr, "Note: pandoc not found, wrote %s instead\n", mdFile)
	return nil
}

func formatReferencesForDocx(refs []synth.Reference) string {
	var lines []string
	for i, ref := range refs {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, ref.CitationAPA))
	}
	return strings.Join(lines, "\n\n")
}

func convertWithPandoc(ctx context.Context, mdFile, docxFile string) error {
	cmd := exec.CommandContext(ctx, "pandoc", mdFile, "-o", docxFile)
	return cmd.Run()
}
