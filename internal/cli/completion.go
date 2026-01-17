package cli

import (
	"fmt"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

// completeIssueNumber provides shell completion for issue numbers.
// It returns all issue numbers with their titles as descriptions.
func completeIssueNumber(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete the first argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	dir, err := getIssuesDir(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	store := issue.NewStore(dir)
	issues, err := store.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, iss := range issues {
		numStr := fmt.Sprintf("%d", iss.Number)
		// Only include if it matches the prefix being typed
		if strings.HasPrefix(numStr, toComplete) {
			// Format: "number\tdescription" - tab separates completion from description
			completion := fmt.Sprintf("%d\t#%d: %s [%s]", iss.Number, iss.Number, iss.Title, iss.State)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeIssueNumberExcluding provides shell completion excluding issues in specified states.
func completeIssueNumberExcluding(excludeStates ...issue.State) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	excludeMap := make(map[issue.State]bool)
	for _, s := range excludeStates {
		excludeMap[s] = true
	}

	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		dir, err := getIssuesDir(cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		store := issue.NewStore(dir)
		issues, err := store.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, iss := range issues {
			// Skip excluded states
			if excludeMap[iss.State] {
				continue
			}

			numStr := fmt.Sprintf("%d", iss.Number)
			if strings.HasPrefix(numStr, toComplete) {
				completion := fmt.Sprintf("%d\t#%d: %s [%s]", iss.Number, iss.Number, iss.Title, iss.State)
				completions = append(completions, completion)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
