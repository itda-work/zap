package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var fixStateCmd = &cobra.Command{
	Use:   "fix-state",
	Short: "Fix issues with invalid state values",
	Long: `Scan for and fix issues with invalid state values.

This command finds issues with deprecated or invalid states (e.g., "in-progress")
and helps convert them to valid states (e.g., "wip").

Valid states: open, wip, check, review, done, closed

Examples:
  zap fix-state              # Interactive mode - asks before fixing
  zap fix-state --dry-run    # Show what would be fixed without changing
  zap fix-state --yes        # Fix all without asking`,
	RunE: runFixState,
}

var (
	fixStateDryRun bool
	fixStateYes    bool
)

func init() {
	rootCmd.AddCommand(fixStateCmd)
	fixStateCmd.Flags().BoolVar(&fixStateDryRun, "dry-run", false, "Show what would be fixed without making changes")
	fixStateCmd.Flags().BoolVarP(&fixStateYes, "yes", "y", false, "Fix all without asking")
}

// knownStateMappings maps deprecated/invalid states to valid ones
var knownStateMappings = map[string]issue.State{
	"in-progress": issue.StateWip,
	"progress":    issue.StateWip,
	"working":     issue.StateWip,
	"started":     issue.StateWip,
	"checking":    issue.StateCheck,
	"verify":      issue.StateCheck,
	"verified":    issue.StateCheck,
	"reviewing":   issue.StateReview,
	"reviewed":    issue.StateReview,
	"complete":    issue.StateDone,
	"completed":   issue.StateDone,
	"finished":    issue.StateDone,
	"cancelled":   issue.StateClosed,
	"canceled":    issue.StateClosed,
	"archived":    issue.StateClosed,
}

func runFixState(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	// Find all issue files
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No issue files found.")
		return nil
	}

	var invalidIssues []invalidIssue
	reader := bufio.NewReader(os.Stdin)

	// Scan for invalid states
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Extract state from frontmatter
		state := extractStateFromContent(string(content))
		if state == "" {
			continue
		}

		// Check if state is valid
		_, valid := issue.ParseState(state)
		if valid {
			continue
		}

		// Found invalid state
		filename := filepath.Base(file)
		suggestion := suggestState(state)
		invalidIssues = append(invalidIssues, invalidIssue{
			path:       file,
			filename:   filename,
			state:      state,
			suggestion: suggestion,
		})
	}

	if len(invalidIssues) == 0 {
		fmt.Println("✅ All issues have valid states.")
		return nil
	}

	fmt.Printf("Found %d issue(s) with invalid state:\n\n", len(invalidIssues))

	fixedCount := 0
	for _, inv := range invalidIssues {
		fmt.Printf("  %s: state \"%s\"", inv.filename, inv.state)
		if inv.suggestion != "" {
			fmt.Printf(" → suggested: \"%s\"", inv.suggestion)
		}
		fmt.Println()

		if fixStateDryRun {
			continue
		}

		if inv.suggestion == "" {
			fmt.Printf("    ⚠️  No suggestion available. Please fix manually.\n")
			continue
		}

		// Ask user or auto-fix
		shouldFix := fixStateYes
		if !shouldFix {
			fmt.Printf("    Fix \"%s\" → \"%s\"? [y/N]: ", inv.state, inv.suggestion)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			shouldFix = answer == "y" || answer == "yes"
		}

		if shouldFix {
			if err := fixIssueState(inv.path, inv.state, inv.suggestion); err != nil {
				fmt.Printf("    ❌ Failed to fix: %v\n", err)
			} else {
				fmt.Printf("    ✅ Fixed: %s → %s\n", inv.state, inv.suggestion)
				fixedCount++
			}
		}
	}

	if fixStateDryRun {
		fmt.Printf("\n(dry-run mode: no changes made)\n")
	} else if fixedCount > 0 {
		fmt.Printf("\n✅ Fixed %d issue(s).\n", fixedCount)
	}

	return nil
}

type invalidIssue struct {
	path       string
	filename   string
	state      string
	suggestion string
}

func extractStateFromContent(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break // End of frontmatter
		}

		if inFrontmatter && strings.HasPrefix(line, "state:") {
			state := strings.TrimPrefix(line, "state:")
			state = strings.TrimSpace(state)
			state = strings.Trim(state, "\"'")
			return state
		}
	}

	return ""
}

func suggestState(invalidState string) string {
	lower := strings.ToLower(invalidState)

	// Check known mappings
	if suggestion, ok := knownStateMappings[lower]; ok {
		return string(suggestion)
	}

	return ""
}

func fixIssueState(filepath, oldState, newState string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	// Replace state in frontmatter
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "state:") {
			// Preserve formatting (quoted or unquoted)
			if strings.Contains(line, "\"") {
				lines[i] = fmt.Sprintf("state: \"%s\"", newState)
			} else if strings.Contains(line, "'") {
				lines[i] = fmt.Sprintf("state: '%s'", newState)
			} else {
				lines[i] = fmt.Sprintf("state: %s", newState)
			}
			break
		}
	}

	newContent := strings.Join(lines, "\n")
	return os.WriteFile(filepath, []byte(newContent), 0644)
}
