// Package ai provides a unified interface for AI CLI tools (claude, codex, gemini).
package ai

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrNoProvider     = errors.New("no AI CLI tool available")
	ErrProviderFailed = errors.New("AI CLI execution failed")
	ErrTimeout        = errors.New("AI request timed out")
	ErrRateLimit      = errors.New("rate limit exceeded")
	ErrAuthFailed     = errors.New("authentication failed - please run the CLI tool manually to authenticate")
)

// Client is the interface for AI CLI tools.
type Client interface {
	// Name returns the provider name (e.g., "claude", "codex", "gemini")
	Name() string

	// IsAvailable checks if the CLI tool is installed
	IsAvailable() bool

	// Complete sends a prompt and returns the response
	Complete(ctx context.Context, req *Request) (*Response, error)
}

// Request represents an AI completion request.
type Request struct {
	// Prompt is the user message/prompt
	Prompt string

	// System is the system prompt (optional)
	System string

	// MaxTokens limits the response length (0 = provider default)
	MaxTokens int

	// Model overrides the default model (optional)
	Model string
}

// Response represents an AI completion response.
type Response struct {
	// Content is the generated text
	Content string

	// Model is the model that was used
	Model string

	// Duration is how long the request took
	Duration time.Duration
}

// Provider represents the type of AI CLI provider.
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
	ProviderGemini Provider = "gemini"
)

// AllProviders returns all supported providers in priority order.
func AllProviders() []Provider {
	return []Provider{
		ProviderClaude,
		ProviderCodex,
		ProviderGemini,
	}
}

// String returns the string representation of the provider.
func (p Provider) String() string {
	return string(p)
}

// ParseProvider converts a string to Provider.
func ParseProvider(s string) (Provider, bool) {
	switch s {
	case "claude":
		return ProviderClaude, true
	case "codex":
		return ProviderCodex, true
	case "gemini":
		return ProviderGemini, true
	default:
		return "", false
	}
}
