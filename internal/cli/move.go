package cli

import (
	"fmt"
	"strconv"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:     "open <number>",
	Aliases: []string{"o"},
	Short:   "Move issue to open state",
	Args:    cobra.ExactArgs(1),
	RunE:    makeMoveFunc(issue.StateOpen),
}

var startCmd = &cobra.Command{
	Use:     "start <number>",
	Aliases: []string{"wip"},
	Short:   "Move issue to in-progress state",
	Args:    cobra.ExactArgs(1),
	RunE:    makeMoveFunc(issue.StateInProgress),
}

var doneCmd = &cobra.Command{
	Use:     "done <number>",
	Aliases: []string{"d"},
	Short:   "Move issue to done state",
	Args:    cobra.ExactArgs(1),
	RunE:    makeMoveFunc(issue.StateDone),
}

var closeCmd = &cobra.Command{
	Use:     "close <number>",
	Aliases: []string{"c"},
	Short:   "Move issue to closed state (cancelled/on-hold)",
	Args:    cobra.ExactArgs(1),
	RunE:    makeMoveFunc(issue.StateClosed),
}

func init() {
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(closeCmd)
}

func makeMoveFunc(targetState issue.State) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
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
