package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:               "open <number>",
	Aliases:           []string{"o"},
	Short:             "Move issue to open state",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueNumberExcluding(issue.StateOpen),
	RunE:              makeMoveFunc(issue.StateOpen),
}

var startCmd = &cobra.Command{
	Use:               "start <number>",
	Aliases:           []string{"wip"},
	Short:             "Move issue to in-progress state",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueNumberExcluding(issue.StateInProgress),
	RunE:              makeMoveFunc(issue.StateInProgress),
}

var doneCmd = &cobra.Command{
	Use:               "done <number>",
	Aliases:           []string{"d"},
	Short:             "Move issue to done state",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueNumberExcluding(issue.StateDone),
	RunE:              makeMoveFunc(issue.StateDone),
}

var closeCmd = &cobra.Command{
	Use:               "close <number>",
	Aliases:           []string{"c"},
	Short:             "Move issue to closed state (cancelled/on-hold)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueNumberExcluding(issue.StateClosed),
	RunE:              makeMoveFunc(issue.StateClosed),
}

var (
	moveProject string
)

func init() {
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(closeCmd)

	// Add --project flag to all move commands
	for _, cmd := range []*cobra.Command{openCmd, startCmd, doneCmd, closeCmd} {
		cmd.Flags().StringVarP(&moveProject, "project", "p", "", "Project alias (for multi-project mode)")
	}
}

func makeMoveFunc(targetState issue.State) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Check for multi-project mode
		if isMultiProjectMode(cmd) {
			return runMultiProjectMove(cmd, args, targetState)
		}

		// Single project mode (existing behavior)
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}

		dir, err := getIssuesDir(cmd)
		if err != nil {
			return err
		}
		store := issue.NewStore(dir)

		// 먼저 이슈 정보 가져오기
		iss, err := store.Get(number)
		if err != nil {
			return err
		}

		if iss.State == targetState {
			fmt.Printf("Issue #%d is already in %s state.\n", number, targetState)
			return nil
		}

		oldState := iss.State

		if err := store.Move(number, targetState); err != nil {
			return fmt.Errorf("failed to move issue: %w", err)
		}

		fmt.Printf("Issue #%d: %s → %s\n", number, oldState, targetState)
		return nil
	}
}

// runMultiProjectMove handles move commands for multiple projects
func runMultiProjectMove(cmd *cobra.Command, args []string, targetState issue.State) error {
	multiStore, err := getMultiStore(cmd)
	if err != nil {
		return err
	}

	// Parse the issue argument - could be "project/#number" or just "number"
	arg := args[0]

	var projectAlias string
	var number int

	// Check if it's a project reference (e.g., "zap/#1")
	if project.IsProjectRef(arg) {
		ref, err := project.ParseRef(arg)
		if err != nil {
			return err
		}
		projectAlias = ref.Project
		number = ref.Number
	} else {
		// It's just a number - need to find it
		var err error
		number, err = strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("invalid issue reference: %s (expected: number or project/#number)", arg)
		}

		// If --project flag is specified, use it
		projectFlag, _ := cmd.Flags().GetString("project")
		if projectFlag != "" {
			projectAlias = projectFlag
		} else {
			// Search across all projects
			matches := multiStore.FindByNumber(number)
			if len(matches) == 0 {
				return fmt.Errorf("issue #%d not found in any project", number)
			}
			if len(matches) > 1 {
				// Ambiguous - show all matches
				fmt.Fprintf(os.Stderr, "Issue #%d exists in multiple projects:\n", number)
				for _, m := range matches {
					fmt.Fprintf(os.Stderr, "  - %s (%s)\n", m.Ref(), m.Title)
				}
				return fmt.Errorf("please specify project with --project or use project/#number format")
			}
			projectAlias = matches[0].Project
		}
	}

	// Get the issue to check current state
	pIss, err := multiStore.Get(projectAlias, number)
	if err != nil {
		return err
	}

	if pIss.State == targetState {
		fmt.Printf("%s is already in %s state.\n", pIss.Ref(), targetState)
		return nil
	}

	oldState := pIss.State

	if err := multiStore.Move(projectAlias, number, targetState); err != nil {
		return fmt.Errorf("failed to move issue: %w", err)
	}

	fmt.Printf("%s: %s → %s\n", pIss.Ref(), oldState, targetState)
	return nil
}
