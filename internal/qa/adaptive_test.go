package qa

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/henrybloomingdale/pubmed-cli/internal/eutils"
)

// mockLLM implements LLMClient for testing.
type mockLLM struct {
	responses []string
	calls     []string
	callIndex int
}

func (m *mockLLM) Complete(ctx context.Context, prompt string, maxTokens int) (string, error) {
	m.calls = append(m.calls, prompt)
	if m.callIndex >= len(m.responses) {
		return "CONFIDENCE: 5\nANSWER: no", nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func TestDetectNovelty(t *testing.T) {
	tests := []struct {
		name     string
		question string
		want     bool
	}{
		// Year pattern tests
		{
			name:     "question with 2024 year",
			question: "What are the effects of the 2024 COVID variant?",
			want:     true,
		},
		{
			name:     "question with 2025 year",
			question: "According to a 2025 meta-analysis, does ketamine help depression?",
			want:     true,
		},
		{
			name:     "question with 2029 year",
			question: "What does the 2029 guideline recommend?",
			want:     true,
		},
		{
			name:     "question with 2030 year",
			question: "Is the 2030 target achievable?",
			want:     true,
		},
		{
			name:     "question with old year 2020",
			question: "What was the 2020 pandemic response?",
			want:     false,
		},
		{
			name:     "question with old year 2023",
			question: "What was published in 2023?",
			want:     false,
		},
		// Recency keyword tests
		{
			name:     "question with 'recent'",
			question: "What are recent advances in CRISPR therapy?",
			want:     true,
		},
		{
			name:     "question with 'latest'",
			question: "What is the latest treatment for migraines?",
			want:     true,
		},
		{
			name:     "question with 'new study'",
			question: "Did a new study show benefits of meditation?",
			want:     true,
		},
		{
			name:     "question with 'new research'",
			question: "What does new research say about sleep?",
			want:     true,
		},
		{
			name:     "question with 'newly published'",
			question: "According to newly published evidence, does X work?",
			want:     true,
		},
		{
			name:     "question with 'this year'",
			question: "What studies were published this year on autism?",
			want:     true,
		},
		{
			name:     "question with 'last month'",
			question: "What was discovered last month?",
			want:     true,
		},
		{
			name:     "question with 'just published'",
			question: "A just published paper claims X - is it true?",
			want:     true,
		},
		// Case insensitivity
		{
			name:     "RECENT uppercase",
			question: "What are RECENT findings?",
			want:     true,
		},
		{
			name:     "Latest mixed case",
			question: "The Latest research shows what?",
			want:     true,
		},
		// No novelty - established knowledge
		{
			name:     "simple established question",
			question: "Does aspirin reduce inflammation?",
			want:     false,
		},
		{
			name:     "mechanism question",
			question: "How does metformin work?",
			want:     false,
		},
		{
			name:     "historical question",
			question: "When was penicillin discovered?",
			want:     false,
		},
		{
			name:     "definition question",
			question: "What is hypertension?",
			want:     false,
		},
		// Edge cases
		{
			name:     "empty string",
			question: "",
			want:     false,
		},
		{
			name:     "number that looks like year but isn't",
			question: "Should I take 2024 mg of vitamin D?",
			want:     true, // matches year pattern even though semantic is different
		},
		{
			name:     "year in middle of sentence",
			question: "The trial NCT2024 showed positive results",
			want:     false, // NCT2024 shouldn't match \b2024\b due to preceding letters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectNovelty(tt.question)
			if got != tt.want {
				t.Errorf("DetectNovelty(%q) = %v, want %v", tt.question, got, tt.want)
			}
		})
	}
}

func TestExpandQuery(t *testing.T) {
	tests := []struct {
		name     string
		question string
		want     string
	}{
		// Preamble removal
		{
			name:     "removes 2025 meta-analysis preamble",
			question: "According to a 2025 meta-analysis, does ketamine help depression?",
			want:     "ketamine help depression",
		},
		{
			name:     "removes 2025 RCT preamble",
			question: "Based on a 2025 RCT, is drug X effective?",
			want:     "drug X effective",
		},
		{
			name:     "removes 2025 study preamble",
			question: "According to a 2025 study, does exercise help anxiety?",
			want:     "exercise help anxiety",
		},
		// Question word removal
		{
			name:     "removes Does at start",
			question: "Does metformin reduce glucose?",
			want:     "metformin reduce glucose",
		},
		{
			name:     "removes Is at start",
			question: "Is aspirin safe for children?",
			want:     "aspirin safe for children",
		},
		{
			name:     "removes Can at start",
			question: "Can exercise prevent diabetes?",
			want:     "exercise prevent diabetes",
		},
		{
			name:     "removes Do at start",
			question: "Do statins cause muscle pain?",
			want:     "statins cause muscle pain",
		},
		// Question mark removal
		{
			name:     "removes trailing question mark",
			question: "What causes headaches?",
			want:     "What causes headaches",
		},
		// Whitespace normalization
		{
			name:     "normalizes multiple spaces",
			question: "Does   metformin    reduce   glucose?",
			want:     "metformin reduce glucose",
		},
		// Combined transformations
		{
			name:     "preamble + question word + question mark",
			question: "According to a 2025 systematic review, Does vitamin D help?",
			want:     "vitamin D help",
		},
		// Length truncation
		{
			name: "truncates long query",
			question: "Does this extremely long question about various medical conditions including " +
				"hypertension diabetes cardiovascular disease and numerous other chronic conditions " +
				"that might affect patient outcomes in clinical trials get truncated properly?",
			want: "this extremely long question about various medical conditions including hypertension " +
				"diabetes cardiovascular disease and numerous other chronic condit",
		},
		// Edge cases
		{
			name:     "empty string",
			question: "",
			want:     "",
		},
		{
			name:     "only question mark",
			question: "?",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandQuery(tt.question)
			if got != tt.want {
				t.Errorf("ExpandQuery(%q) =\n  %q\nwant:\n  %q", tt.question, got, tt.want)
			}
		})
	}
}

func TestMinifyAbstract(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxChars int
		wantLen  int // approximate expected length (0 means check exact match)
		contains []string
		want     string // exact match if provided
	}{
		{
			name:     "empty string",
			text:     "",
			maxChars: 500,
			want:     "",
		},
		{
			name:     "short text under limit",
			text:     "This is a short abstract.",
			maxChars: 500,
			want:     "This is a short abstract.",
		},
		// Structured abstract with labeled sections
		{
			name: "extracts RESULTS section",
			text: "BACKGROUND: We studied depression treatment. " +
				"METHODS: Randomized controlled trial with 200 patients. " +
				"RESULTS: Treatment showed 45% improvement (p<0.001). " +
				"CONCLUSION: The treatment is effective.",
			maxChars: 200,
			contains: []string{"RESULTS:", "45%", "p<0.001"},
		},
		{
			name: "extracts CONCLUSION section",
			text: "Introduction paragraph here. " +
				"Methods were standard. " +
				"CONCLUSIONS: We found significant improvement with 95% CI [1.2-2.4].",
			maxChars: 200,
			contains: []string{"CONCLUSIONS:", "95% CI"},
		},
		// Key term scoring
		{
			name: "prioritizes sentences with key terms",
			text: "The study was conducted. " +
				"Results demonstrated significant improvement in outcomes. " +
				"Data was collected. " +
				"The conclusion showed effective treatment.",
			maxChars: 150,
			contains: []string{"demonstrated", "significant"},
		},
		// Statistics boost
		{
			name: "prioritizes sentences with statistics",
			text: "We conducted a trial. " +
				"The primary outcome showed 78% response rate with p=0.003. " +
				"Patients were enrolled. " +
				"Side effects were mild.",
			maxChars: 150,
			contains: []string{"78%", "p=0.003"},
		},
		{
			name: "prioritizes 95% CI",
			text: "Background information here. " +
				"The pooled effect size was 0.45 (95% CI 0.32-0.58). " +
				"More background. " +
				"Discussion of implications.",
			maxChars: 150,
			contains: []string{"95% CI", "0.45"},
		},
		// Fallback behavior
		{
			name:     "truncates when no good sentences",
			text:     strings.Repeat("x", 500),
			maxChars: 100,
			wantLen:  100,
		},
		// Combined scoring
		{
			name: "meta-analysis term boosts score",
			text: "Study design details. " +
				"This meta-analysis pooled data from 20 trials. " +
				"Other information. " +
				"Final thoughts.",
			maxChars: 100,
			contains: []string{"meta-analysis"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MinifyAbstract(tt.text, tt.maxChars)

			// Check exact match if specified
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("MinifyAbstract() = %q, want %q", got, tt.want)
				}
				return
			}

			// Check length constraint
			if len(got) > tt.maxChars+50 { // allow some flexibility for sentence endings
				t.Errorf("MinifyAbstract() length = %d, want <= %d", len(got), tt.maxChars)
			}

			// Check expected length if specified
			if tt.wantLen > 0 && len(got) < tt.wantLen-10 {
				t.Errorf("MinifyAbstract() length = %d, want ~%d", len(got), tt.wantLen)
			}

			// Check required substrings
			for _, substr := range tt.contains {
				if !strings.Contains(got, substr) {
					t.Errorf("MinifyAbstract() = %q, want to contain %q", got, substr)
				}
			}
		})
	}
}

func TestMinifyAbstract_TokenSavings(t *testing.T) {
	// Simulate a typical long abstract
	longAbstract := `BACKGROUND: Depression is a major public health concern affecting millions worldwide. 
	Current treatments have variable efficacy. New therapeutic approaches are needed.
	METHODS: We conducted a randomized, double-blind, placebo-controlled trial with 500 participants.
	Patients received either the experimental drug or placebo for 12 weeks.
	RESULTS: The treatment group showed significant improvement with a mean reduction of 8.5 points on the HAM-D scale (95% CI 7.2-9.8, p<0.001). 
	Response rate was 67% vs 32% for placebo.
	CONCLUSIONS: The experimental treatment demonstrated significant efficacy for major depression.
	Future studies should examine long-term outcomes.`

	original := len(longAbstract)
	minified := MinifyAbstract(longAbstract, 300)
	savings := float64(original-len(minified)) / float64(original) * 100

	t.Logf("Original: %d chars, Minified: %d chars, Savings: %.1f%%", original, len(minified), savings)

	if savings < 30 {
		t.Errorf("Expected at least 30%% token savings, got %.1f%%", savings)
	}

	// Should preserve key findings
	if !strings.Contains(minified, "95% CI") && !strings.Contains(minified, "p<0.001") {
		t.Error("Minified abstract should preserve statistical findings")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ConfidenceThreshold != 7 {
		t.Errorf("ConfidenceThreshold = %d, want 7", cfg.ConfidenceThreshold)
	}
	if cfg.MaxResults != 3 {
		t.Errorf("MaxResults = %d, want 3", cfg.MaxResults)
	}
	if cfg.ForceRetrieval {
		t.Error("ForceRetrieval should be false by default")
	}
	if cfg.ForceParametric {
		t.Error("ForceParametric should be false by default")
	}
}

func TestNewEngine(t *testing.T) {
	llm := &mockLLM{}
	client := eutils.NewClient()
	cfg := DefaultConfig()

	engine := NewEngine(llm, client, cfg)

	if engine.llm != llm {
		t.Error("Engine.llm not set correctly")
	}
	if engine.eutils != client {
		t.Error("Engine.eutils not set correctly")
	}
	if engine.cfg != cfg {
		t.Error("Engine.cfg not set correctly")
	}
}

func TestEngine_Answer_ForceParametric(t *testing.T) {
	llm := &mockLLM{
		responses: []string{"ANSWER: yes"},
	}
	client := eutils.NewClient()
	cfg := Config{
		ForceParametric:     true,
		ConfidenceThreshold: 7,
	}

	engine := NewEngine(llm, client, cfg)
	result, err := engine.Answer(context.Background(), "Does aspirin reduce inflammation?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Strategy != StrategyParametric {
		t.Errorf("Strategy = %v, want %v", result.Strategy, StrategyParametric)
	}
	if result.Answer != "yes" {
		t.Errorf("Answer = %q, want 'yes'", result.Answer)
	}
}

func TestEngine_Answer_ForceRetrieval(t *testing.T) {
	// Create a mock server for eutils (Search uses JSON, Fetch uses XML)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "esearch") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"esearchresult":{"count":"1","idlist":["12345678"]}}`))
		} else if strings.Contains(r.URL.Path, "efetch") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0"?>
<PubmedArticleSet>
	<PubmedArticle>
		<MedlineCitation>
			<PMID>12345678</PMID>
			<Article>
				<ArticleTitle>Test Article</ArticleTitle>
				<Abstract><AbstractText>RESULTS: Treatment showed 50% improvement.</AbstractText></Abstract>
				<Journal><Title>Test Journal</Title><ISOAbbreviation>Test J</ISOAbbreviation></Journal>
				<AuthorList><Author><LastName>Smith</LastName><ForeName>John</ForeName></Author></AuthorList>
			</Article>
		</MedlineCitation>
		<PubmedData><ArticleIdList><ArticleId IdType="pubmed">12345678</ArticleId></ArticleIdList></PubmedData>
	</PubmedArticle>
</PubmedArticleSet>`))
		}
	}))
	defer server.Close()

	llm := &mockLLM{
		responses: []string{"ANSWER: yes"},
	}
	client := eutils.NewClient(eutils.WithBaseURL(server.URL))
	cfg := Config{
		ForceRetrieval: true,
		MaxResults:     3,
	}

	engine := NewEngine(llm, client, cfg)
	result, err := engine.Answer(context.Background(), "Does treatment X work?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Strategy != StrategyRetrieval {
		t.Errorf("Strategy = %v, want %v", result.Strategy, StrategyRetrieval)
	}
	if len(result.SourcePMIDs) == 0 {
		t.Error("Expected source PMIDs to be populated")
	}
	if result.MinifiedContext == "" {
		t.Error("Expected minified context to be populated")
	}
}

func TestEngine_Answer_NoveltyTriggersRetrieval(t *testing.T) {
	// Create a mock server for eutils (Search uses JSON, Fetch uses XML)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "esearch") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"esearchresult":{"count":"1","idlist":["99999999"]}}`))
		} else if strings.Contains(r.URL.Path, "efetch") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0"?>
<PubmedArticleSet>
	<PubmedArticle>
		<MedlineCitation>
			<PMID>99999999</PMID>
			<Article>
				<ArticleTitle>2025 Study</ArticleTitle>
				<Abstract><AbstractText>Recent findings show effectiveness.</AbstractText></Abstract>
				<Journal><Title>New Journal</Title><ISOAbbreviation>New J</ISOAbbreviation></Journal>
				<AuthorList><Author><LastName>Doe</LastName><ForeName>Jane</ForeName></Author></AuthorList>
			</Article>
		</MedlineCitation>
		<PubmedData><ArticleIdList><ArticleId IdType="pubmed">99999999</ArticleId></ArticleIdList></PubmedData>
	</PubmedArticle>
</PubmedArticleSet>`))
		}
	}))
	defer server.Close()

	llm := &mockLLM{
		responses: []string{"ANSWER: yes"},
	}
	client := eutils.NewClient(eutils.WithBaseURL(server.URL))
	cfg := DefaultConfig()

	engine := NewEngine(llm, client, cfg)

	// Question with 2025 year should trigger novelty detection
	result, err := engine.Answer(context.Background(), "According to a 2025 study, does X help?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.NovelDetected {
		t.Error("Expected NovelDetected to be true for 2025 question")
	}
	if result.Strategy != StrategyRetrieval {
		t.Errorf("Strategy = %v, want %v", result.Strategy, StrategyRetrieval)
	}
}

func TestEngine_Answer_HighConfidenceUsesParametric(t *testing.T) {
	llm := &mockLLM{
		responses: []string{"CONFIDENCE: 9\nANSWER: yes"},
	}
	client := eutils.NewClient()
	cfg := Config{
		ConfidenceThreshold: 7,
		MaxResults:          3,
	}

	engine := NewEngine(llm, client, cfg)
	result, err := engine.Answer(context.Background(), "Does aspirin reduce pain?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Strategy != StrategyParametric {
		t.Errorf("Strategy = %v, want %v", result.Strategy, StrategyParametric)
	}
	if result.Confidence != 9 {
		t.Errorf("Confidence = %d, want 9", result.Confidence)
	}
	if result.Answer != "yes" {
		t.Errorf("Answer = %q, want 'yes'", result.Answer)
	}
}

func TestEngine_Answer_LowConfidenceTriggersRetrieval(t *testing.T) {
	// Create a mock server for eutils (Search uses JSON, Fetch uses XML)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "esearch") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"esearchresult":{"count":"1","idlist":["11111111"]}}`))
		} else if strings.Contains(r.URL.Path, "efetch") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0"?>
<PubmedArticleSet>
	<PubmedArticle>
		<MedlineCitation>
			<PMID>11111111</PMID>
			<Article>
				<ArticleTitle>Obscure Treatment Study</ArticleTitle>
				<Abstract><AbstractText>RESULTS: The obscure treatment showed 30% efficacy.</AbstractText></Abstract>
				<Journal><Title>Obscure Journal</Title><ISOAbbreviation>Obscure J</ISOAbbreviation></Journal>
				<AuthorList><Author><LastName>Unknown</LastName><ForeName>Author</ForeName></Author></AuthorList>
			</Article>
		</MedlineCitation>
		<PubmedData><ArticleIdList><ArticleId IdType="pubmed">11111111</ArticleId></ArticleIdList></PubmedData>
	</PubmedArticle>
</PubmedArticleSet>`))
		}
	}))
	defer server.Close()

	llm := &mockLLM{
		responses: []string{
			"CONFIDENCE: 3\nANSWER: unsure", // First call - low confidence
			"ANSWER: no",                    // Second call - final answer with retrieval
		},
	}
	client := eutils.NewClient(eutils.WithBaseURL(server.URL))
	cfg := Config{
		ConfidenceThreshold: 7,
		MaxResults:          3,
	}

	engine := NewEngine(llm, client, cfg)
	result, err := engine.Answer(context.Background(), "Does obscure-treatment-xyz help condition-abc?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Strategy != StrategyRetrieval {
		t.Errorf("Strategy = %v, want %v", result.Strategy, StrategyRetrieval)
	}
	if result.Confidence != 3 {
		t.Errorf("Confidence = %d, want 3", result.Confidence)
	}
}

func TestEngine_Answer_EmptySearchFallsBackToParametric(t *testing.T) {
	// Create a mock server that returns empty results (Search uses JSON)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "esearch") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"esearchresult":{"count":"0","idlist":[]}}`))
		}
	}))
	defer server.Close()

	llm := &mockLLM{
		responses: []string{
			"CONFIDENCE: 3\nANSWER: unsure", // Low confidence
			"ANSWER: no",                    // Fallback parametric answer
		},
	}
	client := eutils.NewClient(eutils.WithBaseURL(server.URL))
	cfg := Config{
		ConfidenceThreshold: 7,
		MaxResults:          3,
	}

	engine := NewEngine(llm, client, cfg)
	result, err := engine.Answer(context.Background(), "Does made-up-drug-xyz treat fake-disease-abc?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Strategy is still retrieval (that was the decision), but it fell back to parametric for the answer
	if result.Answer != "no" {
		t.Errorf("Answer = %q, want 'no'", result.Answer)
	}
}

func TestResult_FieldsPopulated(t *testing.T) {
	result := &Result{
		Question:        "Test question?",
		Answer:          "yes",
		Confidence:      8,
		Strategy:        StrategyParametric,
		NovelDetected:   false,
		SourcePMIDs:     []string{"12345678"},
		MinifiedContext: "Test context",
	}

	if result.Question != "Test question?" {
		t.Error("Question field mismatch")
	}
	if result.Answer != "yes" {
		t.Error("Answer field mismatch")
	}
	if result.Confidence != 8 {
		t.Error("Confidence field mismatch")
	}
	if result.Strategy != StrategyParametric {
		t.Error("Strategy field mismatch")
	}
	if result.NovelDetected {
		t.Error("NovelDetected should be false")
	}
	if len(result.SourcePMIDs) != 1 || result.SourcePMIDs[0] != "12345678" {
		t.Error("SourcePMIDs field mismatch")
	}
	if result.MinifiedContext != "Test context" {
		t.Error("MinifiedContext field mismatch")
	}
}

func TestStrategy_Constants(t *testing.T) {
	if StrategyParametric != "parametric" {
		t.Errorf("StrategyParametric = %q, want 'parametric'", StrategyParametric)
	}
	if StrategyRetrieval != "retrieval" {
		t.Errorf("StrategyRetrieval = %q, want 'retrieval'", StrategyRetrieval)
	}
}

// Benchmark tests
func BenchmarkDetectNovelty(b *testing.B) {
	questions := []string{
		"Does aspirin reduce inflammation?",
		"According to a 2025 study, does X work?",
		"What are the latest findings on COVID treatment?",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, q := range questions {
			DetectNovelty(q)
		}
	}
}

func BenchmarkMinifyAbstract(b *testing.B) {
	abstract := `BACKGROUND: Depression is a major public health concern affecting millions worldwide. 
	Current treatments have variable efficacy. New therapeutic approaches are needed.
	METHODS: We conducted a randomized, double-blind, placebo-controlled trial with 500 participants.
	RESULTS: The treatment group showed significant improvement with a mean reduction of 8.5 points.
	CONCLUSIONS: The experimental treatment demonstrated significant efficacy for major depression.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MinifyAbstract(abstract, 200)
	}
}

func BenchmarkExpandQuery(b *testing.B) {
	question := "According to a 2025 meta-analysis, does ketamine help treatment-resistant depression?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExpandQuery(question)
	}
}

// Example tests for documentation
func ExampleDetectNovelty() {
	// Recent year triggers novelty
	fmt.Println(DetectNovelty("What does the 2025 study say?"))

	// Recency keyword triggers novelty
	fmt.Println(DetectNovelty("What are the latest findings?"))

	// Established knowledge - no novelty
	fmt.Println(DetectNovelty("Does aspirin reduce inflammation?"))

	// Output:
	// true
	// true
	// false
}

func ExampleMinifyAbstract() {
	abstract := `BACKGROUND: General context information.
	METHODS: Study design details.
	RESULTS: Treatment showed 75% improvement (p<0.001).
	CONCLUSIONS: Treatment is effective.`

	minified := MinifyAbstract(abstract, 100)
	fmt.Println(strings.Contains(minified, "75%"))

	// Output:
	// true
}

func ExampleExpandQuery() {
	// Removes preamble and question words
	query := ExpandQuery("According to a 2025 study, Does treatment X help?")
	fmt.Println(query)

	// Output:
	// treatment X help
}
