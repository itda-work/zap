package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var fixDatetimeCmd = &cobra.Command{
	Use:   "fix-datetime-format [number]",
	Short: "Fix datetime format in issue files",
	Long: `Standardize datetime format to RFC3339 UTC in all issue files.

This command converts all datetime fields (created_at, updated_at, closed_at)
to RFC3339 UTC format (e.g., 2026-01-17T06:30:00Z).

Options:
  --dry-run     Preview changes without modifying files
  --git-dates   Use git history to fill in missing/zero datetime values

Examples:
  zap fix-datetime-format --dry-run    # Preview what would change
  zap fix-datetime-format              # Apply to all issues
  zap fix-datetime-format --git-dates  # Also fix zero values from git
  zap fix-datetime-format 1            # Fix only issue #1`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueNumber,
	RunE:              runFixDatetime,
}

var (
	fixDryRun   bool
	fixGitDates bool
)

func init() {
	rootCmd.AddCommand(fixDatetimeCmd)
	fixDatetimeCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "Preview changes only")
	fixDatetimeCmd.Flags().BoolVar(&fixGitDates, "git-dates", false, "Use git dates for zero values")
}

func runFixDatetime(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	// Load all issues
	issues, err := store.List(issue.AllStates()...)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	// Filter by number if specified as argument
	if len(args) > 0 {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}
		var filtered []*issue.Issue
		for _, iss := range issues {
			if iss.Number == number {
				filtered = append(filtered, iss)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("issue #%d not found", number)
		}
		issues = filtered
	}

	updatedCount := 0
	skippedCount := 0

	for _, iss := range issues {
		needsUpdate := false
		changes := []string{}

		// Check created_at
		originalCreated := iss.CreatedAt
		if iss.CreatedAt.IsZero() && fixGitDates {
			gitTime := getGitCreatedTime(iss.FilePath)
			if !gitTime.IsZero() {
				iss.CreatedAt = gitTime.UTC()
				changes = append(changes, fmt.Sprintf("created_at: (zero) → %s", iss.CreatedAt.Format(time.RFC3339)))
				needsUpdate = true
			}
		} else if !iss.CreatedAt.IsZero() {
			utcTime := iss.CreatedAt.UTC()
			if !timeEqualRFC3339(originalCreated, utcTime) {
				iss.CreatedAt = utcTime
				changes = append(changes, fmt.Sprintf("created_at: %s → %s", originalCreated.Format(time.RFC3339), utcTime.Format(time.RFC3339)))
				needsUpdate = true
			}
		}

		// Check updated_at
		originalUpdated := iss.UpdatedAt
		if iss.UpdatedAt.IsZero() && fixGitDates {
			gitTime := getGitModifiedTime(iss.FilePath)
			if !gitTime.IsZero() {
				iss.UpdatedAt = gitTime.UTC()
				changes = append(changes, fmt.Sprintf("updated_at: (zero) → %s", iss.UpdatedAt.Format(time.RFC3339)))
				needsUpdate = true
			}
		} else if !iss.UpdatedAt.IsZero() {
			utcTime := iss.UpdatedAt.UTC()
			if !timeEqualRFC3339(originalUpdated, utcTime) {
				iss.UpdatedAt = utcTime
				changes = append(changes, fmt.Sprintf("updated_at: %s → %s", originalUpdated.Format(time.RFC3339), utcTime.Format(time.RFC3339)))
				needsUpdate = true
			}
		}

		// Check closed_at
		if iss.ClosedAt != nil {
			originalClosed := *iss.ClosedAt
			utcTime := iss.ClosedAt.UTC()
			if !timeEqualRFC3339(originalClosed, utcTime) {
				iss.ClosedAt = &utcTime
				changes = append(changes, fmt.Sprintf("closed_at: %s → %s", originalClosed.Format(time.RFC3339), utcTime.Format(time.RFC3339)))
				needsUpdate = true
			}
		}

		if !needsUpdate {
			skippedCount++
			continue
		}

		// Print changes
		fmt.Printf("Issue #%d (%s):\n", iss.Number, iss.Title)
		for _, change := range changes {
			fmt.Printf("  %s\n", change)
		}

		if !fixDryRun {
			// Serialize and write
			data, err := issue.Serialize(iss)
			if err != nil {
				fmt.Printf("  ❌ Failed to serialize: %v\n", err)
				continue
			}

			if err := os.WriteFile(iss.FilePath, data, 0644); err != nil {
				fmt.Printf("  ❌ Failed to write: %v\n", err)
				continue
			}
			fmt.Printf("  ✅ Updated\n")
		}

		updatedCount++
	}

	fmt.Println()
	if fixDryRun {
		fmt.Printf("Dry run complete. Would update %d issues (%d already correct).\n", updatedCount, skippedCount)
	} else {
		fmt.Printf("Updated %d issues (%d already correct).\n", updatedCount, skippedCount)
	}

	return nil
}

// timeEqualRFC3339 checks if two times are equal when formatted as RFC3339
// This accounts for timezone differences - we only care about the resulting string
func timeEqualRFC3339(t1, t2 time.Time) bool {
	return t1.UTC().Format(time.RFC3339) == t2.UTC().Format(time.RFC3339)
}

// getGitCreatedTime gets the creation time of a file from git history
func getGitCreatedTime(filePath string) time.Time {
	// Get the first commit that added this file
	cmd := exec.Command("git", "log", "--diff-filter=A", "--follow", "--format=%aI", "-1", "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	timeStr := strings.TrimSpace(string(output))
	if timeStr == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}
	}

	return t
}

// getGitModifiedTime gets the last modification time of a file from git history
func getGitModifiedTime(filePath string) time.Time {
	// Get the most recent commit that modified this file
	cmd := exec.Command("git", "log", "--format=%aI", "-1", "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	timeStr := strings.TrimSpace(string(output))
	if timeStr == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}
	}

	return t
}
