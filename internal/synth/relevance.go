package synth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/henrybloomingdale/pubmed-cli/internal/eutils"
)

var scoreRe = regexp.MustCompile(`\b(10|[1-9])\b`)

// scoreArticleRelevance asks the LLM to rate relevance of an article to the question.
func scoreArticleRelevance(ctx context.Context, llm LLMClient, question string, article *eutils.Article) (int, int, error) {
	if llm == nil {
		return 0, 0, errors.New("LLM client is nil")
	}
	if article == nil {
		return 0, 0, errors.New("article is nil")
	}

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

	// Parse score from response.
	score := parseScore(resp)

	// Very rough token estimate.
	tokensUsed := len(prompt)/4 + 5
	return score, tokensUsed, nil
}

func parseScore(resp string) int {
	resp = strings.TrimSpace(resp)

	match := scoreRe.FindString(resp)
	if match != "" {
		score, err := strconv.Atoi(match)
		if err == nil && score >= 1 && score <= 10 {
			return score
		}
	}
	return 5
}

// truncate returns s truncated to at most maxLen runes.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}
