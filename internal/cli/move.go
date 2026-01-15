package cli

import (
	"fmt"
	"strconv"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <number>",
	Short: "Move issue to open state",
	Args:  cobra.ExactArgs(1),
	RunE:  makeMoveFunc(issue.StateOpen),
}

var startCmd = &cobra.Command{
	Use:   "start <number>",
	Short: "Move issue to in-progress state",
	Args:  cobra.ExactArgs(1),
	RunE:  makeMoveFunc(issue.StateInProgress),
}

var doneCmd = &cobra.Command{
	Use:   "done <number>",
	Short: "Move issue to done state",
	Args:  cobra.ExactArgs(1),
	RunE:  makeMoveFunc(issue.StateDone),
}

func init() {
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
}

func makeMoveFunc(targetState issue.State) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}

		dir, _ := cmd.Flags().GetString("dir")
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
