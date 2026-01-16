package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClaudeClient implements Client for Claude CLI.
type ClaudeClient struct {
	bin   string
	model string
}

// NewClaudeClient creates a new Claude CLI client.
func NewClaudeClient(cfg ProviderConfig) *ClaudeClient {
	bin := cfg.Bin
	if bin == "" {
		bin = "claude"
	}
	return &ClaudeClient{
		bin:   bin,
		model: cfg.Model,
	}
}

// Name returns the provider name.
func (c *ClaudeClient) Name() string {
	return "claude"
}

// IsAvailable checks if Claude CLI is installed.
func (c *ClaudeClient) IsAvailable() bool {
	_, err := exec.LookPath(c.bin)
	return err == nil
}

// Complete sends a completion request to Claude CLI.
func (c *ClaudeClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	args := []string{"-p", req.Prompt, "--output-format", "text"}

	if req.System != "" {
		args = append(args, "--system-prompt", req.System)
	}

	model := req.Model
	if model == "" {
		model = c.model
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	if req.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
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
			strings.Contains(errMsg, "login") {
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
