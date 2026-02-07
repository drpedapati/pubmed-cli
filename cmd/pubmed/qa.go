package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/henrybloomingdale/pubmed-cli/internal/llm"
	"github.com/henrybloomingdale/pubmed-cli/internal/qa"
	"github.com/spf13/cobra"
)

var (
	qaFlagConfidence int
	qaFlagRetrieval  bool
	qaFlagParametric bool
	qaFlagExplain    bool
	qaFlagModel      string
	qaFlagBaseURL    string
	qaFlagClaude     bool
	qaFlagCodex      bool
	qaFlagOpus       bool
	qaFlagUnsafe     bool
)

func init() {
	qaCmd.Flags().IntVar(&qaFlagConfidence, "confidence", 7, "Confidence threshold for parametric answers (1-10)")
	qaCmd.Flags().BoolVar(&qaFlagRetrieval, "retrieve", false, "Force retrieval (skip confidence check)")
	qaCmd.Flags().BoolVar(&qaFlagParametric, "parametric", false, "Force parametric (never retrieve)")
	qaCmd.Flags().BoolVarP(&qaFlagExplain, "explain", "e", false, "Show reasoning and sources")
	qaCmd.Flags().StringVar(&qaFlagModel, "model", "", "LLM model (default: gpt-4o or LLM_MODEL env)")
	qaCmd.Flags().StringVar(&qaFlagBaseURL, "llm-url", "", "LLM API base URL (default: LLM_BASE_URL env)")
	qaCmd.Flags().BoolVar(&qaFlagClaude, "claude", false, "Use Claude CLI (no API key needed)")
	qaCmd.Flags().BoolVar(&qaFlagCodex, "codex", false, "Use OpenAI Codex CLI (no API key needed)")
	qaCmd.Flags().BoolVar(&qaFlagOpus, "opus", false, "Use Claude Opus model (with --claude)")
	qaCmd.Flags().BoolVar(&qaFlagUnsafe, "unsafe", false, "Enable full LLM access (DANGEROUS: bypasses sandbox)")

	rootCmd.AddCommand(qaCmd)
}

var qaCmd = &cobra.Command{
	Use:   "qa <question>",
	Short: "Answer biomedical yes/no questions with adaptive retrieval",
	Long: `Answers biomedical questions using adaptive retrieval:

1. Detects if question requires novel (post-training) knowledge
2. Checks model confidence for established knowledge
3. Retrieves from PubMed only when necessary
4. Minifies abstracts to preserve key findings

Examples:
  pubmed qa "Does CBT help hypertension-related anxiety?"
  pubmed qa --explain "According to 2025 studies, does SGLT-2 reduce liver fibrosis?"
  pubmed qa --retrieve "Is metformin effective for PCOS?"

Environment variables:
  LLM_API_KEY   - API key for LLM (or OPENAI_API_KEY)
  LLM_BASE_URL  - Base URL for OpenAI-compatible API
  LLM_MODEL     - Model name (default: gpt-4o)`,
	Args: cobra.MinimumNArgs(1),
	RunE: runQA,
}

// LLMCompleter is the interface both OpenAI and Claude clients implement.
type LLMCompleter interface {
	Complete(ctx context.Context, prompt string, maxTokens int) (string, error)
}

func runQA(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	// Validate mutually exclusive flags.
	if qaFlagClaude && qaFlagCodex {
		return fmt.Errorf("--claude and --codex are mutually exclusive")
	}

	// Determine security config for LLM clients.
	// QA uses read-only by default (safest for question-answering).
	securityCfg := llm.ForQA()
	if qaFlagUnsafe {
		fmt.Fprintln(cmd.ErrOrStderr(), "⚠️  WARNING: --unsafe enables full LLM access. The model can execute arbitrary commands.")
		securityCfg = securityCfg.WithFullAccess()
	}

	// Build LLM client
	var llmClient LLMCompleter
	var err error

	if qaFlagCodex {
		// Use Codex via OAuth tokens from ChatGPT account
		codexOpts := []llm.CodexOption{
			llm.WithSecurityConfig(securityCfg),
		}
		if qaFlagModel != "" {
			codexOpts = append(codexOpts, llm.WithCodexModel(qaFlagModel))
		}
		llmClient, err = llm.NewCodexClient(codexOpts...)
		if err != nil {
			return fmt.Errorf("codex setup: %w", err)
		}
	} else if qaFlagClaude {
		// Use Claude via OAuth tokens from keychain
		claudeOpts := []llm.ClaudeOption{
			llm.WithClaudeSecurityConfig(securityCfg),
		}
		if qaFlagModel != "" {
			claudeOpts = append(claudeOpts, llm.WithClaudeModel(qaFlagModel))
		}
		if qaFlagOpus {
			claudeOpts = append(claudeOpts, llm.WithOpus(true))
		}
		llmClient, err = llm.NewClaudeClientWithOptions(claudeOpts...)
		if err != nil {
			return fmt.Errorf("claude setup: %w", err)
		}
	} else {
		// Use OpenAI-compatible API
		var llmOpts []llm.Option
		if qaFlagModel != "" {
			llmOpts = append(llmOpts, llm.WithModel(qaFlagModel))
		}
		if qaFlagBaseURL != "" {
			llmOpts = append(llmOpts, llm.WithBaseURL(qaFlagBaseURL))
		}
		llmClient = llm.NewClient(llmOpts...)
	}

	// Build QA engine
	cfg := qa.DefaultConfig()
	cfg.ConfidenceThreshold = qaFlagConfidence
	cfg.ForceRetrieval = qaFlagRetrieval
	cfg.ForceParametric = qaFlagParametric
	cfg.Verbose = qaFlagExplain

	engine := qa.NewEngine(llmClient, newEutilsClient(), cfg)

	// Get answer
	result, err := engine.Answer(cmd.Context(), question)
	if err != nil {
		return fmt.Errorf("qa failed: %w", err)
	}

	// Output
	if flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if qaFlagExplain || flagHuman {
		printExplainedResult(result)
	} else {
		fmt.Println(result.Answer)
	}

	return nil
}

func printExplainedResult(r *qa.Result) {
	// Strategy icon
	stratIcon := "🧠"
	if r.Strategy == qa.StrategyRetrieval {
		stratIcon = "🔍"
	}

	fmt.Printf("\n%s Answer: %s\n", stratIcon, strings.ToUpper(r.Answer))
	fmt.Printf("   Strategy: %s\n", r.Strategy)

	if r.NovelDetected {
		fmt.Println("   Novel knowledge detected: yes")
	}
	if r.Confidence > 0 {
		fmt.Printf("   Confidence: %d/10\n", r.Confidence)
	}
	if len(r.SourcePMIDs) > 0 {
		fmt.Printf("   Sources: %s\n", strings.Join(r.SourcePMIDs, ", "))
	}
	if r.MinifiedContext != "" && len(r.MinifiedContext) < 500 {
		fmt.Printf("\n   Context:\n   %s\n", strings.ReplaceAll(r.MinifiedContext, "\n", "\n   "))
	}
	fmt.Println()
}
