package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <state> <number>",
	Short: "Set issue state (open, wip, done, closed)",
	Long: `Set issue state to one of: open, wip, done, closed.

Examples:
  zap set done 1
  zap set wip 5
  zap set open 2
  zap set closed 3`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeSetArgs,
	RunE:              runSetCmd,
}

var (
	setProject string
)

func init() {
	rootCmd.AddCommand(setCmd)
	setCmd.Flags().StringVarP(&setProject, "project", "p", "", "Project alias (for multi-project mode)")
}

// completeSetArgs provides completion for the set command
func completeSetArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// Complete state names
		states := []string{
			"open\tReopen issue",
			"wip\tStart working on issue",
			"done\tMark issue as completed",
			"closed\tClose issue (cancelled/on-hold)",
		}
		var completions []string
		for _, s := range states {
			if strings.HasPrefix(s, toComplete) {
				completions = append(completions, s)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		// Complete issue numbers (excluding issues already in the target state)
		targetState, ok := issue.ParseState(args[0])
		if !ok {
			return nil, cobra.ShellCompDirectiveError
		}
		return completeIssueNumberExcluding(targetState)(cmd, args[1:], toComplete)
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func runSetCmd(cmd *cobra.Command, args []string) error {
	stateStr := args[0]
	targetState, ok := issue.ParseState(stateStr)
	if !ok {
		return fmt.Errorf("invalid state: %s (valid: open, wip, done, closed)", stateStr)
	}

	// Check for multi-project mode
	if isMultiProjectMode(cmd) {
		return runMultiProjectMove(cmd, args[1:], targetState)
	}

	// Single project mode
	number, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[1])
	}

	// Get issues directory with discovery info
	dir, wasDiscovered, err := getIssuesDirWithDiscovery(cmd)
	if err != nil {
		return err
	}

	// If discovered from parent directory
	if wasDiscovered {
		// Show info message
		fmt.Fprintf(os.Stderr, "info: Using .issues at %s\n", dir)

		// Check if TTY
		if !IsTTY() {
			return fmt.Errorf("cannot modify issues in parent directory from non-interactive session (use --project or -d flag to specify directory explicitly)")
		}

		// Confirm with user
		if !confirmYesDefault("Proceed with this .issues directory?") {
			return fmt.Errorf("operation cancelled")
		}
	}

	store := issue.NewStore(dir)

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
	printTransitionTip(targetState)
	return nil
}

// printTransitionTip prints a helpful tip after state transition
func printTransitionTip(state issue.State) {
	var tip string
	switch state {
	case issue.StateWip:
		tip = "Tip: 구현 내용을 이슈에 기록하세요."
	case issue.StateDone:
		tip = "Tip: 작업이 완료되었습니다."
	default:
		return
	}
	fmt.Println(colorize(tip, colorGray))
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
	printTransitionTip(targetState)
	return nil
}
