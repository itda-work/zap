package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CodexClient implements Client for Codex CLI (OpenAI).
type CodexClient struct {
	bin   string
	model string
}

// NewCodexClient creates a new Codex CLI client.
func NewCodexClient(cfg ProviderConfig) *CodexClient {
	bin := cfg.Bin
	if bin == "" {
		bin = "codex"
	}
	return &CodexClient{
		bin:   bin,
		model: cfg.Model,
	}
}

// Name returns the provider name.
func (c *CodexClient) Name() string {
	return "codex"
}

// IsAvailable checks if Codex CLI is installed.
func (c *CodexClient) IsAvailable() bool {
	_, err := exec.LookPath(c.bin)
	return err == nil
}

// Complete sends a completion request to Codex CLI.
func (c *CodexClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Build the full prompt (codex uses system prompt differently)
	prompt := req.Prompt
	if req.System != "" {
		prompt = req.System + "\n\n" + prompt
	}

	args := []string{"-q", prompt} // -q for quiet mode

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
			strings.Contains(errMsg, "OPENAI_API_KEY") {
			return nil, fmt.Errorf("%w: %s", ErrAuthFailed, errMsg)
		}

		// Check for rate limits
		if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429") {
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
