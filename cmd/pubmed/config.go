package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage wizard configuration",
	Long: `View and modify wizard defaults.

Commands:
  pubmed config show    - Show current configuration
  pubmed config set     - Interactive configuration editor
  pubmed config reset   - Reset to defaults`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadWizardConfig()

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2)

		configPath := getConfigPath()

		content := fmt.Sprintf(`📁 Config file: %s

📊 Defaults:
   Papers to include:    %d
   Target word count:    %d
   Relevance threshold:  %d

📄 Output:
   Output folder:  %s
   Prefer DOCX:    %v
   Include RIS:    %v

🤖 LLM:
   Use Claude CLI: %v
   Model:          %s`,
			configPath,
			cfg.DefaultPapers,
			cfg.DefaultWords,
			cfg.DefaultRelevance,
			cfg.OutputFolder,
			cfg.PreferDocx,
			cfg.PreferRIS,
			cfg.UseClaude,
			valueOrDefault(cfg.LLMModel, "(auto)"))

		fmt.Println(style.Render(content))
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Interactive configuration editor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadWizardConfig()

		var (
			papersStr    = fmt.Sprintf("%d", cfg.DefaultPapers)
			wordsStr     = fmt.Sprintf("%d", cfg.DefaultWords)
			relevanceStr = fmt.Sprintf("%d", cfg.DefaultRelevance)
			outputFolder = cfg.OutputFolder
			preferDocx   = cfg.PreferDocx
			preferRIS    = cfg.PreferRIS
			useClaude    = cfg.UseClaude
			llmModel     = cfg.LLMModel
		)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Default papers to include").
					Value(&papersStr).
					Validate(validatePositiveInt),
				huh.NewInput().
					Title("Default word count").
					Value(&wordsStr).
					Validate(validatePositiveInt),
				huh.NewInput().
					Title("Relevance threshold (1-10)").
					Value(&relevanceStr).
					Validate(func(s string) error {
						n, err := strconv.Atoi(s)
						if err != nil {
							return fmt.Errorf("enter a number")
						}
						if n < 1 || n > 10 {
							return fmt.Errorf("must be 1-10")
						}
						return nil
					}),
			).Title("Synthesis Defaults"),

			huh.NewGroup(
				huh.NewInput().
					Title("Output folder").
					Value(&outputFolder).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("output folder is required")
						}
						return nil
					}),
				huh.NewConfirm().
					Title("Generate Word documents by default?").
					Value(&preferDocx),
				huh.NewConfirm().
					Title("Generate RIS files by default?").
					Value(&preferRIS),
			).Title("Output Settings"),

			huh.NewGroup(
				huh.NewConfirm().
					Title("Use Claude CLI instead of OpenAI API?").
					Description("Claude CLI uses OAuth, no API key needed").
					Value(&useClaude),
				huh.NewInput().
					Title("LLM Model (leave empty for auto)").
					Description("e.g., gpt-4o, claude-3-opus, gemini-pro").
					Value(&llmModel),
			).Title("LLM Settings"),
		).WithTheme(huh.ThemeCatppuccin())

		if err := form.Run(); err != nil {
			return err
		}

		p, err := strconv.Atoi(papersStr)
		if err != nil {
			return fmt.Errorf("parse papers: %w", err)
		}
		w, err := strconv.Atoi(wordsStr)
		if err != nil {
			return fmt.Errorf("parse words: %w", err)
		}
		r, err := strconv.Atoi(relevanceStr)
		if err != nil {
			return fmt.Errorf("parse relevance: %w", err)
		}

		cfg.DefaultPapers = p
		cfg.DefaultWords = w
		cfg.DefaultRelevance = r
		cfg.OutputFolder = strings.TrimSpace(outputFolder)
		cfg.PreferDocx = preferDocx
		cfg.PreferRIS = preferRIS
		cfg.UseClaude = useClaude
		cfg.LLMModel = strings.TrimSpace(llmModel)

		if err := saveWizardConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("✓ Configuration saved!"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("  %s", getConfigPath())))
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := DefaultWizardConfig()
		if err := saveWizardConfig(cfg); err != nil {
			return err
		}
		fmt.Println(successStyle.Render("✓ Configuration reset to defaults"))
		return nil
	},
}

func valueOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
