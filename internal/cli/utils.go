package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/ai"
	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	// DefaultRecentClosedMinutes is the default duration (in minutes) to show recently closed issues
	DefaultRecentClosedMinutes = 5
	// EnvRecentClosedMinutes is the environment variable name for configuring recent closed duration
	EnvRecentClosedMinutes = "ZAP_RECENT_CLOSED_MINUTES"
)

// getRecentClosedDuration returns the duration for which recently closed/done issues should be displayed.
// It reads from ZAP_RECENT_CLOSED_MINUTES environment variable, defaulting to 5 minutes.
func getRecentClosedDuration() time.Duration {
	if val := os.Getenv(EnvRecentClosedMinutes); val != "" {
		if minutes, err := strconv.Atoi(val); err == nil && minutes >= 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return DefaultRecentClosedMinutes * time.Minute
}

// isRecentlyClosed checks if an issue was recently closed (done or closed state) within the given duration.
func isRecentlyClosed(updatedAt time.Time, state string, duration time.Duration) bool {
	if state != "done" && state != "closed" {
		return false
	}
	return time.Since(updatedAt) <= duration
}

// confirm prompts the user for yes/no confirmation.
func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// On error (EOF, closed stdin), default to No for safety
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// confirmYesDefault prompts with [Y/n] - Enter defaults to Yes
func confirmYesDefault(prompt string) bool {
	fmt.Printf("%s [Y/n]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// On error (EOF, closed stdin), default to No for safety
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))

	// Empty or "y" or "yes" = true
	if response == "" || response == "y" || response == "yes" {
		return true
	}
	return false
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

// formatRelativeTime formats a time as relative time string (e.g., "2 hr ago", "3 days ago")
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	// Future time
	if diff < 0 {
		return "just now"
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
		return "just now"
	case minutes < 60:
		return fmt.Sprintf("%d min ago", minutes)
	case hours < 24:
		return fmt.Sprintf("%d hr ago", hours)
	case days < 7:
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case weeks < 4:
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case months < 12:
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// statePriority returns the priority for sorting issues by state.
// Lower value = appears first in the list.
// Order: done(0) → closed(1) → wip(2) → open(3)
func statePriority(state issue.State) int {
	switch state {
	case issue.StateDone:
		return 0
	case issue.StateClosed:
		return 1
	case issue.StateWip:
		return 2
	case issue.StateOpen:
		return 3
	default:
		return 4
	}
}

// sortIssuesByStateAndTime sorts issues by state priority, then by UpdatedAt descending.
// State order: done → closed → wip → open
// Within each state group: most recently updated first
func sortIssuesByStateAndTime(issues []*issue.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		pi, pj := statePriority(issues[i].State), statePriority(issues[j].State)
		if pi != pj {
			return pi < pj
		}
		// Same state: sort by UpdatedAt descending
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})
}

// sortProjectIssuesByStateAndTime sorts project issues by state priority, then by UpdatedAt descending.
// State order: done → closed → wip → open
// Within each state group: most recently updated first
func sortProjectIssuesByStateAndTime(issues []*project.ProjectIssue) {
	sort.Slice(issues, func(i, j int) bool {
		pi, pj := statePriority(issues[i].State), statePriority(issues[j].State)
		if pi != pj {
			return pi < pj
		}
		// Same state: sort by UpdatedAt descending
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})
}

// getTerminalWidth returns the current terminal width.
// Falls back to 80 columns if detection fails.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

// truncateLine truncates a string containing ANSI escape codes to fit within
// maxWidth visible characters. If truncation occurs, an ellipsis (…) is appended.
// Handles CJK wide characters correctly via go-runewidth.
func truncateLine(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	visibleWidth := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		visibleWidth += runewidth.RuneWidth(r)
		if visibleWidth > maxWidth {
			break
		}
	}

	if visibleWidth <= maxWidth {
		return s
	}

	limit := maxWidth - 1
	if limit < 0 {
		limit = 0
	}

	visibleWidth = 0
	inEscape = false
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if r == '\033' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}

		w := runewidth.RuneWidth(r)
		if visibleWidth+w > limit {
			result.WriteString(colorReset)
			result.WriteRune('…')
			return result.String()
		}
		visibleWidth += w
		result.WriteRune(r)
	}

	return result.String()
}
