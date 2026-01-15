package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/allieus/lim/internal/issue"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <number>",
	Short: "Show issue details",
	Long:  `Show detailed information about a specific issue.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

var showRaw bool

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Show raw markdown content")
}

func runShow(cmd *cobra.Command, args []string) error {
	number, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	dir, _ := cmd.Flags().GetString("dir")
	store := issue.NewStore(dir)

	iss, err := store.Get(number)
	if err != nil {
		return err
	}

	if showRaw {
		printRawIssue(iss)
	} else {
		printIssueDetail(iss)
	}

	return nil
}

func printIssueDetail(iss *issue.Issue) {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Issue #%d: %s\n", iss.Number, iss.Title)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("State:    %s\n", iss.State)

	if len(iss.Labels) > 0 {
		fmt.Printf("Labels:   %s\n", strings.Join(iss.Labels, ", "))
	}

	if len(iss.Assignees) > 0 {
		fmt.Printf("Assignee: %s\n", strings.Join(iss.Assignees, ", "))
	}

	fmt.Printf("Created:  %s\n", iss.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Updated:  %s\n", iss.UpdatedAt.Format("2006-01-02 15:04"))

	if iss.ClosedAt != nil {
		fmt.Printf("Closed:   %s\n", iss.ClosedAt.Format("2006-01-02 15:04"))
	}

	fmt.Printf("File:     %s\n", iss.FilePath)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if iss.Body != "" {
		fmt.Printf("\n%s\n", iss.Body)
	}
}

func printRawIssue(iss *issue.Issue) {
	data, err := issue.Serialize(iss)
	if err != nil {
		fmt.Printf("Error serializing issue: %v\n", err)
		return
	}
	fmt.Print(string(data))
}
