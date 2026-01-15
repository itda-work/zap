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
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all issues including done and closed")
	listCmd.Flags().StringVarP(&listState, "state", "s", "", "Filter by state (open, in-progress, done, closed)")
	listCmd.Flags().StringVarP(&listLabel, "label", "l", "", "Filter by label")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "Filter by assignee")
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

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	printIssueList(issues)
	return nil
}

func printIssueList(issues []*issue.Issue) {
	// 상태별 색상/기호
	stateSymbol := map[issue.State]string{
		issue.StateOpen:       "○",
		issue.StateInProgress: "◐",
		issue.StateDone:       "●",
		issue.StateClosed:     "✕",
	}

	for _, iss := range issues {
		symbol := stateSymbol[iss.State]
		labels := ""
		if len(iss.Labels) > 0 {
			labels = fmt.Sprintf(" [%s]", strings.Join(iss.Labels, ", "))
		}

		fmt.Printf("%s #%-4d %s%s\n", symbol, iss.Number, iss.Title, labels)
	}

	fmt.Printf("\nTotal: %d issues\n", len(issues))
}
