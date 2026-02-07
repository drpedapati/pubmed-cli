// Input sanitization for LLM CLI integrations.
//
// Security Model:
//
// This package shells out to Claude and Codex CLIs via exec.Command. While Go's
// exec.Command uses execve() directly (bypassing shell interpretation), we still
// implement defense-in-depth sanitization for several reasons:
//
//  1. Prompt Injection Defense: LLMs can be manipulated via crafted inputs that
//     attempt to override their instructions. We block common injection patterns.
//
//  2. Defense in Depth: Even though shell metacharacters won't execute, blocking
//     suspicious patterns provides an additional safety layer.
//
// 3. DoS Prevention: Length limits prevent resource exhaustion attacks.
//
// 4. Unicode Normalization: Prevents homoglyph attacks and encoding confusion.
//
// 5. Audit Trail: Rejected prompts are logged for security review.
//
// This is NOT a complete solution to prompt injection (an unsolved problem), but
// it raises the bar significantly and catches obvious attack attempts.
package llm

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Exported constants for prompt validation.
const (
	// MaxPromptLength is the maximum allowed prompt length in characters.
	// 10,000 chars is generous for research questions while preventing abuse.
	MaxPromptLength = 10000

	// MinPromptLength is the minimum prompt length to reject trivial inputs.
	MinPromptLength = 5
)

// Validation error types
var (
	ErrPromptTooShort = fmt.Errorf("prompt too short")
	ErrPromptTooLong  = fmt.Errorf("prompt too long")
	ErrNullByte       = fmt.Errorf("prompt contains null bytes")
	ErrDisallowedURL  = fmt.Errorf("prompt contains disallowed URL")

	// ErrUnsafeContent is the parent error for security-related rejections.
	// Use errors.Is(err, ErrUnsafeContent) to catch shell metachars AND prompt injection.
	ErrUnsafeContent   = fmt.Errorf("prompt contains unsafe content")
	ErrShellMetachar   = fmt.Errorf("%w: shell metacharacters detected", ErrUnsafeContent)
	ErrPromptInjection = fmt.Errorf("%w: injection pattern detected", ErrUnsafeContent)
)

// Shell metacharacter patterns that could indicate injection attempts.
var shellMetacharPattern = regexp.MustCompile(`[;|&$` + "`" + `]|\$\(|\)\s*[|&;]`)

// Patterns that suggest prompt injection attempts.
var promptInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|context)`),
	regexp.MustCompile(`(?i)disregard\s+(all\s+)?(previous|prior|above)`),
	regexp.MustCompile(`(?i)forget\s+(everything|all|what)\s+(you|i)\s+(told|said)`),
	regexp.MustCompile(`(?i)new\s+instructions?:\s*`),
	regexp.MustCompile(`(?i)system\s*:\s*you\s+are`),
	regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an|my)`),
	regexp.MustCompile(`(?i)pretend\s+(you\s+are|to\s+be)\s+(a|an)`),
	regexp.MustCompile(`(?i)act\s+as\s+(if|though|a|an)`),
	regexp.MustCompile(`(?i)override\s+(previous|system|safety)`),
	regexp.MustCompile(`(?i)\[\[.*?(system|admin|root).*?\]\]`),
	regexp.MustCompile(`(?i)<\|?(system|endoftext|im_start|im_end)\|?>`),
	regexp.MustCompile(`(?i)jailbreak`),
}

// URL pattern for domain validation
var urlPattern = regexp.MustCompile(`https?://([^/\s]+)`)

// SanitizePrompt validates and sanitizes user input before passing to LLM CLIs.
// Returns the sanitized prompt and an error if the input is rejected.
//
// Uses default validation rules:
//   - Min length: 5 chars
//   - Max length: 10,000 chars
//   - Shell metacharacters blocked
//   - Prompt injection patterns blocked
//   - Control characters stripped
func SanitizePrompt(prompt string) (string, error) {
	cfg := SecurityConfig{
		MaxPromptLength:      MaxPromptLength,
		AllowShellMetachars:  false,
		BlockPromptInjection: true,
		AllowedDomains:       nil,
	}
	return SanitizePromptWithConfig(prompt, cfg)
}

// SanitizePromptWithConfig validates and sanitizes with custom SecurityConfig.
func SanitizePromptWithConfig(prompt string, cfg SecurityConfig) (string, error) {
	// Step 1: Trim whitespace first
	prompt = strings.TrimSpace(prompt)

	// Step 2: Handle null bytes - always strip them
	if strings.ContainsRune(prompt, '\x00') {
		prompt = strings.ReplaceAll(prompt, "\x00", "")
	}

	// Step 3: Strip control characters (except common whitespace)
	prompt = stripControlChars(prompt)

	// Step 4: Normalize unicode to NFC form
	// This prevents homoglyph attacks and encoding confusion
	prompt = norm.NFC.String(prompt)

	// Step 5: Check minimum length
	if len(prompt) < MinPromptLength {
		return "", ErrPromptTooShort
	}

	// Step 6: Check maximum length
	maxLen := cfg.MaxPromptLength
	if maxLen <= 0 {
		maxLen = MaxPromptLength
	}
	if len(prompt) > maxLen {
		logRejection("length_exceeded", len(prompt))
		return "", ErrPromptTooLong
	}

	// Step 7: Check for shell metacharacters
	if !cfg.AllowShellMetachars && containsShellMetachars(prompt) {
		logRejection("shell_metachars", "detected")
		return "", ErrShellMetachar
	}

	// Step 8: Check for prompt injection patterns
	if cfg.BlockPromptInjection && containsPromptInjection(prompt) {
		logRejection("prompt_injection", "detected")
		return "", ErrPromptInjection
	}

	// Step 9: Validate URL domains if configured
	if len(cfg.AllowedDomains) > 0 {
		if err := validateURLDomains(prompt, cfg.AllowedDomains); err != nil {
			logRejection("disallowed_url", err.Error())
			return "", err
		}
	}

	return prompt, nil
}

// stripControlChars removes non-printable control characters except common whitespace.
func stripControlChars(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		// Keep common whitespace (space, tab, newline, carriage return)
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
			continue
		}
		// Keep printable characters and valid Unicode
		if !unicode.IsControl(r) {
			result.WriteRune(r)
		}
		// Drop everything else (control characters, etc.)
	}

	return result.String()
}

// containsShellMetachars checks for shell metacharacters.
func containsShellMetachars(s string) bool {
	return shellMetacharPattern.MatchString(s)
}

// containsPromptInjection checks for prompt injection patterns.
func containsPromptInjection(s string) bool {
	for _, pattern := range promptInjectionPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}

// validateURLDomains checks that all URLs in the prompt are from allowed domains.
func validateURLDomains(prompt string, allowedDomains []string) error {
	matches := urlPattern.FindAllStringSubmatch(prompt, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		domain := strings.ToLower(match[1])
		allowed := false
		for _, d := range allowedDomains {
			if strings.EqualFold(domain, d) || strings.HasSuffix(domain, "."+d) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: %s", ErrDisallowedURL, domain)
		}
	}
	return nil
}

// logRejection logs rejected prompts for security auditing.
// The actual prompt content is NOT logged to avoid leaking sensitive data.
func logRejection(reason string, detail interface{}) {
	// Use standard log package - in production this would go to a security audit log
	log.Printf("[SECURITY] Prompt rejected: reason=%s detail=%v", reason, detail)
}

// ValidatePrompt is a compatibility wrapper that performs basic validation.
// Deprecated: Use SanitizePrompt for new code.
func ValidatePrompt(prompt string) error {
	_, err := SanitizePrompt(prompt)
	return err
}
