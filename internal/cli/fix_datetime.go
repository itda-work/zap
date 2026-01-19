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
  --analyze     Analyze current datetime formats without making changes

Examples:
  zap fix-datetime-format --dry-run    # Preview what would change
  zap fix-datetime-format              # Apply to all issues
  zap fix-datetime-format --analyze    # Show format distribution statistics
  zap fix-datetime-format 1            # Fix only issue #1`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueNumber,
	RunE:              runFixDatetime,
}

var (
	fixDryRun  bool
	fixAnalyze bool
)

func init() {
	rootCmd.AddCommand(fixDatetimeCmd)
	fixDatetimeCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "Preview changes only")
	fixDatetimeCmd.Flags().BoolVar(&fixAnalyze, "analyze", false, "Analyze datetime formats")
}

func runFixDatetime(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	// Handle --analyze mode
	if fixAnalyze {
		return runAnalyzeDatetime(dir)
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

		// Get raw datetime strings to detect original format
		rawInfo, err := issue.GetRawDatetimeInfo(iss.FilePath)
		if err != nil {
			fmt.Printf("Warning: failed to read raw datetime for issue #%d: %v\n", iss.Number, err)
			continue
		}

		// Check created_at
		createdFmt := issue.DetectDatetimeFormat(rawInfo.CreatedAt)
		if iss.CreatedAt.IsZero() {
			// Zero value: always use git time
			gitTime := getGitCreatedTime(iss.FilePath)
			if !gitTime.IsZero() {
				iss.CreatedAt = gitTime.UTC()
				changes = append(changes, fmt.Sprintf("created_at: (zero) → %s", iss.CreatedAt.Format(time.RFC3339)))
				needsUpdate = true
			}
		} else if createdFmt != issue.FormatRFC3339 {
			// Original format is not RFC3339, needs conversion
			if isDateOnlyFormat(createdFmt) {
				// Always use git time for date-only formats
				gitTime := getGitCreatedTime(iss.FilePath)
				if !gitTime.IsZero() {
					iss.CreatedAt = gitTime.UTC()
				} else {
					iss.CreatedAt = iss.CreatedAt.UTC()
				}
			} else {
				iss.CreatedAt = iss.CreatedAt.UTC()
			}
			changes = append(changes, fmt.Sprintf("created_at: %s → %s", rawInfo.CreatedAt, iss.CreatedAt.Format(time.RFC3339)))
			needsUpdate = true
		}

		// Check updated_at
		updatedFmt := issue.DetectDatetimeFormat(rawInfo.UpdatedAt)
		if iss.UpdatedAt.IsZero() {
			// Zero value: always use git time
			gitTime := getGitModifiedTime(iss.FilePath)
			if !gitTime.IsZero() {
				iss.UpdatedAt = gitTime.UTC()
				changes = append(changes, fmt.Sprintf("updated_at: (zero) → %s", iss.UpdatedAt.Format(time.RFC3339)))
				needsUpdate = true
			}
		} else if updatedFmt != issue.FormatRFC3339 {
			// Original format is not RFC3339, needs conversion
			if isDateOnlyFormat(updatedFmt) {
				// Always use git time for date-only formats
				gitTime := getGitModifiedTime(iss.FilePath)
				if !gitTime.IsZero() {
					iss.UpdatedAt = gitTime.UTC()
				} else {
					iss.UpdatedAt = iss.UpdatedAt.UTC()
				}
			} else {
				iss.UpdatedAt = iss.UpdatedAt.UTC()
			}
			changes = append(changes, fmt.Sprintf("updated_at: %s → %s", rawInfo.UpdatedAt, iss.UpdatedAt.Format(time.RFC3339)))
			needsUpdate = true
		}

		// Check closed_at
		if rawInfo.ClosedAt != "" {
			closedFmt := issue.DetectDatetimeFormat(rawInfo.ClosedAt)
			if iss.ClosedAt != nil && closedFmt != issue.FormatRFC3339 {
				iss.ClosedAt = timePtr(iss.ClosedAt.UTC())
				changes = append(changes, fmt.Sprintf("closed_at: %s → %s", rawInfo.ClosedAt, iss.ClosedAt.Format(time.RFC3339)))
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

// formatStats holds statistics for a datetime format
type formatStats struct {
	count   int
	issues  []int // issue numbers
}

// runAnalyzeDatetime analyzes datetime formats across all issues
func runAnalyzeDatetime(dir string) error {
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

	// Stats by field and format
	createdStats := make(map[issue.DatetimeFormat]*formatStats)
	updatedStats := make(map[issue.DatetimeFormat]*formatStats)
	closedStats := make(map[issue.DatetimeFormat]*formatStats)

	for _, iss := range issues {
		raw, err := issue.GetRawDatetimeInfo(iss.FilePath)
		if err != nil {
			fmt.Printf("Warning: failed to read raw datetime for issue #%d: %v\n", iss.Number, err)
			continue
		}

		// Analyze created_at
		createdFmt := issue.DetectDatetimeFormat(raw.CreatedAt)
		if createdStats[createdFmt] == nil {
			createdStats[createdFmt] = &formatStats{}
		}
		createdStats[createdFmt].count++
		createdStats[createdFmt].issues = append(createdStats[createdFmt].issues, iss.Number)

		// Analyze updated_at
		updatedFmt := issue.DetectDatetimeFormat(raw.UpdatedAt)
		if updatedStats[updatedFmt] == nil {
			updatedStats[updatedFmt] = &formatStats{}
		}
		updatedStats[updatedFmt].count++
		updatedStats[updatedFmt].issues = append(updatedStats[updatedFmt].issues, iss.Number)

		// Analyze closed_at (only if present)
		if raw.ClosedAt != "" {
			closedFmt := issue.DetectDatetimeFormat(raw.ClosedAt)
			if closedStats[closedFmt] == nil {
				closedStats[closedFmt] = &formatStats{}
			}
			closedStats[closedFmt].count++
			closedStats[closedFmt].issues = append(closedStats[closedFmt].issues, iss.Number)
		}
	}

	// Print results
	fmt.Println("DateTime Format Analysis")
	fmt.Println("========================")
	fmt.Println()

	printFieldStats("created_at", createdStats)
	printFieldStats("updated_at", updatedStats)
	if len(closedStats) > 0 {
		printFieldStats("closed_at", closedStats)
	}

	// Summary
	totalFields := 0
	rfc3339Fields := 0
	needConversion := 0

	for fmt, stats := range createdStats {
		totalFields += stats.count
		if fmt == issue.FormatRFC3339 {
			rfc3339Fields += stats.count
		} else if fmt != issue.FormatEmpty {
			needConversion += stats.count
		}
	}
	for fmt, stats := range updatedStats {
		totalFields += stats.count
		if fmt == issue.FormatRFC3339 {
			rfc3339Fields += stats.count
		} else if fmt != issue.FormatEmpty {
			needConversion += stats.count
		}
	}
	for fmt, stats := range closedStats {
		totalFields += stats.count
		if fmt == issue.FormatRFC3339 {
			rfc3339Fields += stats.count
		} else if fmt != issue.FormatEmpty {
			needConversion += stats.count
		}
	}

	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Printf("  Total issues:      %d\n", len(issues))
	fmt.Printf("  Total fields:      %d\n", totalFields)
	fmt.Printf("  Already RFC3339:   %d\n", rfc3339Fields)
	fmt.Printf("  Need conversion:   %d\n", needConversion)

	return nil
}

// formatExamples maps DatetimeFormat to example strings
var formatExamples = map[issue.DatetimeFormat]string{
	issue.FormatRFC3339:       "2026-01-17T15:47:00Z",
	issue.FormatISO8601:       "2026-01-17T15:47:00",
	issue.FormatDatetimeSpace: "2026-01-17 15:47:00",
	issue.FormatDatetimeShort: "2026-01-17 15:47",
	issue.FormatDateOnly:      "2026-01-17",
	issue.FormatEmpty:         "",
	issue.FormatUnknown:       "",
}

// printFieldStats prints statistics for a datetime field
func printFieldStats(fieldName string, stats map[issue.DatetimeFormat]*formatStats) {
	fmt.Printf("%s:\n", fieldName)

	// Define order for formats
	formatOrder := []issue.DatetimeFormat{
		issue.FormatRFC3339,
		issue.FormatISO8601,
		issue.FormatDatetimeSpace,
		issue.FormatDatetimeShort,
		issue.FormatDateOnly,
		issue.FormatEmpty,
		issue.FormatUnknown,
	}

	for _, format := range formatOrder {
		s, ok := stats[format]
		if !ok || s.count == 0 {
			continue
		}

		// Format issue numbers for display (max 5)
		issueStr := formatIssueNumbers(s.issues, 5)

		// Pluralize "issue"
		issueWord := "issue"
		if s.count > 1 {
			issueWord = "issues"
		}

		// Build format label with example
		label := string(format)
		if example, ok := formatExamples[format]; ok && example != "" {
			label = fmt.Sprintf("%s (%s)", format, example)
		}

		fmt.Printf("  %-42s %3d %s %s\n", label, s.count, issueWord, issueStr)
	}
	fmt.Println()
}

// formatIssueNumbers formats issue numbers for display
func formatIssueNumbers(numbers []int, max int) string {
	if len(numbers) == 0 {
		return ""
	}

	var parts []string
	for i, n := range numbers {
		if i >= max {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("#%d", n))
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

// isDateOnlyFormat checks if the format is date-only (no time component)
func isDateOnlyFormat(fmt issue.DatetimeFormat) bool {
	return fmt == issue.FormatDateOnly
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}
