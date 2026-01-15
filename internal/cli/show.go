package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/charmbracelet/glamour"
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
		rendered, err := renderMarkdown(iss.Body)
		if err != nil {
			fmt.Printf("\n%s\n", iss.Body)
		} else {
			fmt.Print(rendered)
		}
	}
}

func renderMarkdown(content string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
		glamour.WithStylesFromJSONBytes([]byte(compactStyle)),
	)
	if err != nil {
		return "", err
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return "", err
	}
	// 빈 줄 모두 제거
	rendered = removeBlankLines(rendered)
	// glamour는 끝에 개행을 추가하므로 제거
	return strings.TrimSuffix(rendered, "\n"), nil
}

// removeBlankLines removes all blank lines (lines with only whitespace)
func removeBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var result []string

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// 리스트 항목 사이 여백을 제거한 컴팩트 스타일
const compactStyle = `{
	"list": {
		"level_indent": 2,
		"margin": 0
	},
	"item": {
		"block_prefix": "",
		"margin": 0
	},
	"paragraph": {
		"margin": 0
	},
	"code_block": {
		"margin": 0
	}
}`

func printRawIssue(iss *issue.Issue) {
	data, err := issue.Serialize(iss)
	if err != nil {
		fmt.Printf("Error serializing issue: %v\n", err)
		return
	}
	fmt.Print(string(data))
}
