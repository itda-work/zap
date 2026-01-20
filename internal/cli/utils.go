package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/ai"
)

// confirm prompts the user for yes/no confirmation.
func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// getAIClient returns an AI client based on the provided flag or auto-detection.
func getAIClient(aiFlag string) (ai.Client, error) {
	cfg, err := ai.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load AI config: %w", err)
	}

	if aiFlag != "" {
		provider, ok := ai.ParseProvider(aiFlag)
		if !ok {
			return nil, fmt.Errorf("unknown AI provider: %s (supported: claude, codex, gemini)", aiFlag)
		}
		client := ai.NewClient(provider, cfg)
		if client == nil || !client.IsAvailable() {
			return nil, fmt.Errorf("%s CLI is not installed or not available", aiFlag)
		}
		return client, nil
	}

	// Auto-detect
	client, err := ai.AutoDetect(cfg)
	if err != nil {
		return nil, fmt.Errorf("no AI CLI available. Install one of: claude, codex, gemini")
	}
	return client, nil
}

// formatRelativeTime formats a time as relative time string (e.g., "2시간 전", "3일 전")
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	// Future time
	if diff < 0 {
		return "방금"
	}

	seconds := int(diff.Seconds())
	minutes := int(diff.Minutes())
	hours := int(diff.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 60:
		return "방금"
	case minutes < 60:
		return fmt.Sprintf("%d분 전", minutes)
	case hours < 24:
		return fmt.Sprintf("%d시간 전", hours)
	case days < 7:
		return fmt.Sprintf("%d일 전", days)
	case weeks < 4:
		return fmt.Sprintf("%d주 전", weeks)
	case months < 12:
		return fmt.Sprintf("%d개월 전", months)
	default:
		return fmt.Sprintf("%d년 전", years)
	}
}
