package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
