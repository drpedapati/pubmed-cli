package synth

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/henrybloomingdale/pubmed-cli/internal/eutils"
)

// scoreArticleRelevance asks the LLM to rate relevance of an article to the question.
func scoreArticleRelevance(ctx context.Context, llm LLMClient, question string, article *eutils.Article) (int, int, error) {
	prompt := fmt.Sprintf(`Rate how relevant this paper is to the research question.

Question: %s

Paper Title: %s
Abstract: %s

Rate relevance from 1-10 where:
1-3 = Not relevant (different topic, population, or scope)
4-6 = Somewhat relevant (related but not directly addressing the question)
7-9 = Highly relevant (directly addresses the question)
10 = Perfect match (exactly what the question asks about)

Respond with only the number (1-10):`, question, article.Title, truncate(article.Abstract, 500))

	resp, err := llm.Complete(ctx, prompt, 10)
	if err != nil {
		return 0, 0, err
	}

	// Parse score from response
	score := parseScore(resp)

	// Estimate tokens used
	tokensUsed := len(prompt)/4 + 5

	return score, tokensUsed, nil
}

func parseScore(resp string) int {
	resp = strings.TrimSpace(resp)

	// Try to find a number 1-10
	re := regexp.MustCompile(`\b(10|[1-9])\b`)
	match := re.FindString(resp)
	if match != "" {
		score, err := strconv.Atoi(match)
		if err == nil && score >= 1 && score <= 10 {
			return score
		}
	}

	// Default to neutral if parsing fails
	return 5
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
