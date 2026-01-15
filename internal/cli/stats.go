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

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	store := issue.NewStore(dir)

	stats, err := store.Stats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	printStats(stats)
	return nil
}

func printStats(stats *issue.Stats) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                    Issue Statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("\n📊 Total Issues: %d\n", stats.Total)

	// 상태별 통계
	fmt.Println("\n📁 By State:")
	stateOrder := []issue.State{issue.StateOpen, issue.StateInProgress, issue.StateDone}
	stateEmoji := map[issue.State]string{
		issue.StateOpen:       "○",
		issue.StateInProgress: "◐",
		issue.StateDone:       "●",
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
