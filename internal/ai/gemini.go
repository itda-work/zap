package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GeminiClient implements Client for Gemini CLI.
type GeminiClient struct {
	bin   string
	model string
}

// NewGeminiClient creates a new Gemini CLI client.
func NewGeminiClient(cfg ProviderConfig) *GeminiClient {
	bin := cfg.Bin
	if bin == "" {
		bin = "gemini"
	}
	return &GeminiClient{
		bin:   bin,
		model: cfg.Model,
	}
}

// Name returns the provider name.
func (c *GeminiClient) Name() string {
	return "gemini"
}

// IsAvailable checks if Gemini CLI is installed.
func (c *GeminiClient) IsAvailable() bool {
	_, err := exec.LookPath(c.bin)
	return err == nil
}

// Complete sends a completion request to Gemini CLI.
func (c *GeminiClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Build the full prompt
	prompt := req.Prompt
	if req.System != "" {
		prompt = req.System + "\n\n" + prompt
	}

	args := []string{"-p", prompt}

	model := req.Model
	if model == "" {
		model = c.model
	}
	if model != "" {
		args = append(args, "-m", model)
	}

	cmd := exec.CommandContext(ctx, c.bin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}

		errMsg := stderr.String()

		// Check for auth errors
		if strings.Contains(errMsg, "authentication") ||
			strings.Contains(errMsg, "unauthorized") ||
			strings.Contains(errMsg, "API key") ||
			strings.Contains(errMsg, "GOOGLE_API_KEY") {
			return nil, fmt.Errorf("%w: %s", ErrAuthFailed, errMsg)
		}

		// Check for rate limits
		if strings.Contains(errMsg, "rate limit") ||
			strings.Contains(errMsg, "RESOURCE_EXHAUSTED") ||
			strings.Contains(errMsg, "429") {
			return nil, ErrRateLimit
		}

		return nil, fmt.Errorf("%w: %s", ErrProviderFailed, errMsg)
	}

	return &Response{
		Content:  strings.TrimSpace(stdout.String()),
		Model:    model,
		Duration: time.Since(start),
	}, nil
}
