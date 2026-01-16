package cli

import (
	"fmt"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search issues by keyword",
	Long:  `Search issues by keyword in title and body.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

var searchTitleOnly bool

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolVarP(&searchTitleOnly, "title", "t", false, "Search in title only")
}

func runSearch(cmd *cobra.Command, args []string) error {
	keyword := args[0]

	dir, _ := cmd.Flags().GetString("dir")
	store := issue.NewStore(dir)

	issues, err := store.Search(keyword, searchTitleOnly)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(issues) == 0 {
		fmt.Printf("No issues found matching \"%s\"\n", keyword)
		return nil
	}

	fmt.Printf("Found %d issue(s) matching \"%s\":\n\n", len(issues), keyword)
	printSearchResults(issues, keyword)
	return nil
}

func printSearchResults(issues []*issue.Issue, keyword string) {
	// 상태별 색상/기호 (list.go와 동일)
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

		// 제목에서 키워드 하이라이트 (대소문자 무시)
		title := highlightKeyword(iss.Title, keyword)

		line := fmt.Sprintf("%s #%-4d %s", style.symbol, iss.Number, title)
		fmt.Println(colorize(line, style.color))
	}
}

func highlightKeyword(text, keyword string) string {
	// colorEnabled가 false면 하이라이트하지 않음
	if !colorEnabled {
		return text
	}

	// 간단한 하이라이트 (터미널 볼드)
	lower := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)

	idx := strings.Index(lower, lowerKeyword)
	if idx == -1 {
		return text
	}

	// ANSI 볼드로 하이라이트
	before := text[:idx]
	match := text[idx : idx+len(keyword)]
	after := text[idx+len(keyword):]

	return before + "\033[1m" + match + colorReset + after
}
