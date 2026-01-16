package cli

import (
	"fmt"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long:  `List issues from the .issues directory. By default shows active issues (open + in-progress).`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listAll      bool
	listState    string
	listLabel    string
	listAssignee string
	listQuiet    bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all issues including done and closed")
	listCmd.Flags().StringVarP(&listState, "state", "s", "", "Filter by state (open, in-progress, done, closed)")
	listCmd.Flags().StringVarP(&listLabel, "label", "l", "", "Filter by label")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "Filter by assignee")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Suppress parse failure warnings")
}

func runList(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	var states []issue.State

	if listState != "" {
		state, ok := issue.ParseState(listState)
		if !ok {
			return fmt.Errorf("invalid state: %s", listState)
		}
		states = []issue.State{state}
	} else if listAll {
		states = issue.AllStates()
	} else {
		states = issue.ActiveStates()
	}

	var issues []*issue.Issue

	if listLabel != "" {
		issues, err = store.FilterByLabel(listLabel, states...)
	} else if listAssignee != "" {
		issues, err = store.FilterByAssignee(listAssignee, states...)
	} else {
		issues, err = store.List(states...)
	}

	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Get warnings from store
	warnings := store.Warnings()

	if len(issues) == 0 && len(warnings) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	if len(issues) > 0 {
		printIssueList(issues, len(warnings))
	}

	// Print warnings unless --quiet is set
	if !listQuiet && len(warnings) > 0 {
		printParseWarnings(warnings)
	}

	return nil
}

func printIssueList(issues []*issue.Issue, skippedCount int) {
	// 상태별 색상/기호
	stateStyle := map[issue.State]struct {
		symbol string
		color  string
	}{
		issue.StateOpen:       {"○", ""},
		issue.StateInProgress: {"◐", colorYellow},
		issue.StateDone:       {"●", colorGreen},
		issue.StateClosed:     {"✕", colorGray},
	}

	for _, iss := range issues {
		style := stateStyle[iss.State]
		labels := ""
		if len(iss.Labels) > 0 {
			labels = fmt.Sprintf(" [%s]", strings.Join(iss.Labels, ", "))
		}

		line := fmt.Sprintf("%s #%-4d %s%s", style.symbol, iss.Number, iss.Title, labels)
		fmt.Println(colorize(line, style.color))
	}

	if skippedCount > 0 {
		fmt.Printf("\nTotal: %d issues (%d skipped)\n", len(issues), skippedCount)
	} else {
		fmt.Printf("\nTotal: %d issues\n", len(issues))
	}
}

func printParseWarnings(warnings []issue.ParseFailure) {
	fmt.Println(colorize(fmt.Sprintf("\n⚠️  Parse failures (%d files):", len(warnings)), colorYellow))
	for _, w := range warnings {
		// Truncate filename if too long
		name := w.FileName
		if len(name) > 50 {
			name = name[:47] + "..."
		}
		// Truncate error message
		errMsg := w.Error
		if len(errMsg) > 60 {
			errMsg = errMsg[:57] + "..."
		}
		fmt.Printf("  %s: %s\n", colorize("- "+name, colorGray), errMsg)
	}
	fmt.Println(colorize("\nRun 'zap repair --auto' to auto-fix with AI (requires claude/codex/gemini CLI)", colorGray))
}
