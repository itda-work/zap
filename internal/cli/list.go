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
	dir, _ := cmd.Flags().GetString("dir")
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
	var err error

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

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorGray   = "\033[90m"
	colorRed    = "\033[31m"
)

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

		if style.color != "" {
			fmt.Printf("%s%s%s #%-4d %s%s\n", style.color, style.symbol, colorReset, iss.Number, iss.Title, labels)
		} else {
			fmt.Printf("%s #%-4d %s%s\n", style.symbol, iss.Number, iss.Title, labels)
		}
	}

	if skippedCount > 0 {
		fmt.Printf("\nTotal: %d issues (%d skipped)\n", len(issues), skippedCount)
	} else {
		fmt.Printf("\nTotal: %d issues\n", len(issues))
	}
}

func printParseWarnings(warnings []issue.ParseFailure) {
	fmt.Printf("\n%s⚠️  Parse failures (%d files):%s\n", colorYellow, len(warnings), colorReset)
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
		fmt.Printf("  %s- %s%s: %s\n", colorGray, name, colorReset, errMsg)
	}
	fmt.Printf("\n%sRun 'zap repair --all' to fix with AI (requires claude/codex/gemini CLI)%s\n", colorGray, colorReset)
}
