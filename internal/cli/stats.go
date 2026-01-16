package cli

import (
	"fmt"
	"sort"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show issue statistics",
	Long:  `Display statistics about issues including counts by state, label, and assignee.`,
	RunE:  runStats,
}

var statsDateFilter DateFilter

func init() {
	rootCmd.AddCommand(statsCmd)

	// Date filter options
	statsCmd.Flags().BoolVar(&statsDateFilter.Today, "today", false, "Show statistics for issues created/updated today")
	statsCmd.Flags().StringVar(&statsDateFilter.Since, "since", "", "Show statistics since date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsDateFilter.Until, "until", "", "Show statistics until date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsDateFilter.Year, "year", "", "Show statistics for year (YYYY)")
	statsCmd.Flags().StringVar(&statsDateFilter.Month, "month", "", "Show statistics for month (YYYY-MM)")
	statsCmd.Flags().StringVar(&statsDateFilter.Date, "date", "", "Show statistics for specific date (YYYY-MM-DD)")
	statsCmd.Flags().IntVar(&statsDateFilter.Days, "days", 0, "Show statistics for last N days")
	statsCmd.Flags().IntVar(&statsDateFilter.Weeks, "weeks", 0, "Show statistics for last N weeks")
}

func runStats(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	// Get all issues first
	issues, err := store.List(issue.AllStates()...)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Apply date filter if specified
	filterDescription := ""
	if !statsDateFilter.IsEmpty() {
		issues, err = FilterIssuesByDate(issues, &statsDateFilter)
		if err != nil {
			return err
		}
		filterDescription = getFilterDescription(&statsDateFilter)
	}

	// Calculate stats from filtered issues
	stats := calculateStats(issues)

	printStats(stats, filterDescription)
	return nil
}

// calculateStats computes statistics from a list of issues
func calculateStats(issues []*issue.Issue) *issue.Stats {
	stats := &issue.Stats{
		Total:      len(issues),
		ByState:    make(map[issue.State]int),
		ByLabel:    make(map[string]int),
		ByAssignee: make(map[string]int),
	}

	for _, iss := range issues {
		stats.ByState[iss.State]++

		for _, label := range iss.Labels {
			stats.ByLabel[label]++
		}

		for _, assignee := range iss.Assignees {
			stats.ByAssignee[assignee]++
		}
	}

	return stats
}

// getFilterDescription returns a human-readable description of the filter
func getFilterDescription(filter *DateFilter) string {
	if filter.Today {
		return "today"
	}
	if filter.Days > 0 {
		return fmt.Sprintf("last %d days", filter.Days)
	}
	if filter.Weeks > 0 {
		return fmt.Sprintf("last %d weeks", filter.Weeks)
	}
	if filter.Date != "" {
		return filter.Date
	}
	if filter.Month != "" {
		return filter.Month
	}
	if filter.Year != "" {
		return filter.Year
	}
	if filter.Since != "" && filter.Until != "" {
		return fmt.Sprintf("%s ~ %s", filter.Since, filter.Until)
	}
	if filter.Since != "" {
		return fmt.Sprintf("since %s", filter.Since)
	}
	if filter.Until != "" {
		return fmt.Sprintf("until %s", filter.Until)
	}
	return ""
}

func printStats(stats *issue.Stats, filterDescription string) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if filterDescription != "" {
		fmt.Printf("            Issue Statistics (%s)\n", filterDescription)
	} else {
		fmt.Println("                    Issue Statistics")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("\n📊 Total Issues: %d\n", stats.Total)

	// 상태별 통계
	fmt.Println("\n📁 By State:")
	stateOrder := []issue.State{issue.StateOpen, issue.StateInProgress, issue.StateDone, issue.StateClosed}
	stateEmoji := map[issue.State]string{
		issue.StateOpen:       "○",
		issue.StateInProgress: "◐",
		issue.StateDone:       "●",
		issue.StateClosed:     "✕",
	}

	for _, state := range stateOrder {
		count := stats.ByState[state]
		bar := makeBar(count, stats.Total, 20)
		fmt.Printf("  %s %-12s %3d %s\n", stateEmoji[state], state, count, bar)
	}

	// 레이블별 통계
	if len(stats.ByLabel) > 0 {
		fmt.Println("\n🏷️  By Label:")
		labels := sortedMapKeys(stats.ByLabel)
		for _, label := range labels {
			count := stats.ByLabel[label]
			bar := makeBar(count, stats.Total, 20)
			fmt.Printf("  %-15s %3d %s\n", label, count, bar)
		}
	}

	// 담당자별 통계
	if len(stats.ByAssignee) > 0 {
		fmt.Println("\n👤 By Assignee:")
		assignees := sortedMapKeys(stats.ByAssignee)
		for _, assignee := range assignees {
			count := stats.ByAssignee[assignee]
			bar := makeBar(count, stats.Total, 20)
			fmt.Printf("  %-15s %3d %s\n", assignee, count, bar)
		}
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func makeBar(count, total, width int) string {
	if total == 0 {
		return ""
	}

	filled := (count * width) / total
	if count > 0 && filled == 0 {
		filled = 1 // 최소 1칸
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}

	percentage := float64(count) * 100 / float64(total)
	return fmt.Sprintf("%s %.0f%%", bar, percentage)
}

func sortedMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
