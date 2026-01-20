package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long:  `List issues from the .issues directory. By default shows active issues (open + wip).`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listAll        bool
	listState      string
	listLabel      string
	listAssignee   string
	listQuiet      bool
	listSearch     string
	listTitleOnly  bool
	listDateFilter DateFilter
	listRefs       bool
	listNoDate     bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all issues including done and closed")
	listCmd.Flags().StringVarP(&listState, "state", "s", "", "Filter by state (open, wip, done, closed)")
	listCmd.Flags().StringVarP(&listLabel, "label", "l", "", "Filter by label")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "Filter by assignee")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Suppress parse failure warnings")
	listCmd.Flags().StringVarP(&listSearch, "search", "S", "", "Search in title and body")
	listCmd.Flags().BoolVar(&listTitleOnly, "title-only", false, "Search in title only (use with --search)")

	// Date filter options
	listCmd.Flags().BoolVar(&listDateFilter.Today, "today", false, "Show issues created/updated today")
	listCmd.Flags().StringVar(&listDateFilter.Since, "since", "", "Show issues since date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listDateFilter.Until, "until", "", "Show issues until date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listDateFilter.Year, "year", "", "Show issues from year (YYYY)")
	listCmd.Flags().StringVar(&listDateFilter.Month, "month", "", "Show issues from month (YYYY-MM)")
	listCmd.Flags().StringVar(&listDateFilter.Date, "date", "", "Show issues from specific date (YYYY-MM-DD)")
	listCmd.Flags().IntVar(&listDateFilter.Days, "days", 0, "Show issues from last N days")
	listCmd.Flags().IntVar(&listDateFilter.Weeks, "weeks", 0, "Show issues from last N weeks")

	// Reference options
	listCmd.Flags().BoolVar(&listRefs, "refs", false, "Show reference count for each issue")

	// Date display options
	listCmd.Flags().BoolVar(&listNoDate, "no-date", false, "Hide updated time from output")
}

func runList(cmd *cobra.Command, args []string) error {
	// Check for multi-project mode
	if isMultiProjectMode(cmd) {
		return runMultiProjectList(cmd, args)
	}

	// Single project mode (existing behavior)
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
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

	// Include recently closed issues if not showing all and not filtering by specific state
	recentClosedDuration := getRecentClosedDuration()
	if !listAll && listState == "" && recentClosedDuration > 0 {
		recentIssues, err := getRecentlyClosedIssues(store, recentClosedDuration, listLabel, listAssignee)
		if err == nil && len(recentIssues) > 0 {
			issues = mergeIssues(issues, recentIssues)
		}
	}

	// Apply search filter if specified
	if listSearch != "" {
		issues = filterBySearch(issues, listSearch, listTitleOnly)
	}

	// Apply date filter if specified
	if !listDateFilter.IsEmpty() {
		issues, err = FilterIssuesByDate(issues, &listDateFilter)
		if err != nil {
			return err
		}
	}

	// Get warnings from store
	warnings := store.Warnings()

	if len(issues) == 0 && len(warnings) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	// Build ref graph if --refs is specified
	var refGraph *issue.RefGraph
	if listRefs {
		refGraph, err = store.BuildRefGraph()
		if err != nil {
			return fmt.Errorf("failed to build reference graph: %w", err)
		}
	}

	if len(issues) > 0 {
		// Sort by state priority (done → closed → wip → open), then by UpdatedAt descending
		sortIssuesByStateAndTime(issues)
		printIssueList(issues, len(warnings), listSearch, refGraph, recentClosedDuration)
	}

	// Print warnings unless --quiet is set
	if !listQuiet && len(warnings) > 0 {
		printParseWarnings(warnings)
	}

	return nil
}

// runMultiProjectList handles listing for multiple projects
func runMultiProjectList(cmd *cobra.Command, args []string) error {
	multiStore, err := getMultiStore(cmd)
	if err != nil {
		return err
	}

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

	var projectIssues []*project.ProjectIssue

	if listLabel != "" {
		projectIssues, err = multiStore.FilterByLabel(listLabel, states...)
	} else if listAssignee != "" {
		projectIssues, err = multiStore.FilterByAssignee(listAssignee, states...)
	} else {
		projectIssues, err = multiStore.ListAll(states...)
	}

	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Apply search filter
	if listSearch != "" {
		projectIssues = filterProjectIssuesBySearch(projectIssues, listSearch, listTitleOnly)
	}

	// Apply date filter
	if !listDateFilter.IsEmpty() {
		projectIssues, err = filterProjectIssuesByDate(projectIssues, &listDateFilter)
		if err != nil {
			return err
		}
	}

	// Get warnings from all projects
	warnings := multiStore.Warnings()

	if len(projectIssues) == 0 && len(warnings) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	if len(projectIssues) > 0 {
		// Sort by state priority (done → closed → wip → open), then by UpdatedAt descending
		sortProjectIssuesByStateAndTime(projectIssues)
		printMultiProjectIssueList(projectIssues, len(warnings), listSearch)
	}

	// Print warnings unless --quiet is set
	if !listQuiet && len(warnings) > 0 {
		printMultiProjectWarnings(warnings)
	}

	return nil
}

func printIssueList(issues []*issue.Issue, skippedCount int, keyword string, refGraph *issue.RefGraph, recentClosedDuration time.Duration) {
	// 상태별 텍스트 태그와 색상
	stateStyle := map[issue.State]struct {
		tag        string
		color      string
		titleColor string
	}{
		issue.StateOpen:   {"[open]", "", ""},
		issue.StateWip:    {"[wip]", colorBrightYellow, colorBrightYellow},
		issue.StateDone:   {"[done]", colorBrightGreen, colorBrightGreen},
		issue.StateClosed: {"[closed]", colorGray, colorLightGray},
	}

	for _, iss := range issues {
		style := stateStyle[iss.State]
		labels := ""
		if len(iss.Labels) > 0 {
			labels = fmt.Sprintf(" [%s]", strings.Join(iss.Labels, ", "))
		}

		// Reference count suffix
		refSuffix := ""
		if refGraph != nil {
			count := refGraph.GetRefCount(iss.Number)
			if count > 0 {
				refSuffix = fmt.Sprintf(" %s", colorize(fmt.Sprintf("(refs: %d)", count), colorGray))
			}
		}

		// Updated time suffix
		dateSuffix := ""
		if !listNoDate {
			dateSuffix = fmt.Sprintf(" %s", colorize(formatRelativeTime(iss.UpdatedAt), colorGray))
		}

		// Check if this is a recently closed issue
		recentlyClosed := isRecentlyClosed(iss.UpdatedAt, string(iss.State), recentClosedDuration)

		// 제목에 키워드 하이라이트 적용
		title := highlightKeyword(iss.Title, keyword)

		if recentlyClosed {
			// Apply background color for entire row of recently closed issues
			tag := colorizeWithBg(fmt.Sprintf("%-8s", style.tag), style.color, bgGray)
			titlePart := colorizeWithBg(title, style.titleColor, bgGray)
			labelsPart := colorizeWithBg(labels, "", bgGray)
			refPart := colorizeWithBg(strings.TrimPrefix(refSuffix, " "), colorGray, bgGray)
			datePart := colorizeWithBg(strings.TrimPrefix(dateSuffix, " "), colorGray, bgGray)

			// Build the line with consistent background
			line := fmt.Sprintf("%s #%-4d %s", tag, iss.Number, titlePart)
			if labels != "" {
				line += " " + labelsPart
			}
			if refSuffix != "" {
				line += " " + refPart
			}
			if dateSuffix != "" {
				line += " " + datePart
			}
			fmt.Println(line)
		} else {
			// 상태별 밝은 색상을 제목에 적용
			title = colorize(title, style.titleColor)
			// 태그를 색상 적용 후 출력
			tag := colorize(fmt.Sprintf("%-8s", style.tag), style.color)
			fmt.Printf("%s #%-4d %s%s%s%s\n", tag, iss.Number, title, labels, refSuffix, dateSuffix)
		}
	}

	if skippedCount > 0 {
		fmt.Printf("\nTotal: %d issues (%d skipped)\n", len(issues), skippedCount)
	} else {
		fmt.Printf("\nTotal: %d issues\n", len(issues))
	}
}

// printMultiProjectIssueList prints issues with project prefixes
func printMultiProjectIssueList(issues []*project.ProjectIssue, skippedCount int, keyword string) {
	// 상태별 텍스트 태그와 색상
	stateStyle := map[issue.State]struct {
		tag        string
		color      string
		titleColor string
	}{
		issue.StateOpen:   {"[open]", "", ""},
		issue.StateWip:    {"[wip]", colorBrightYellow, colorBrightYellow},
		issue.StateDone:   {"[done]", colorBrightGreen, colorBrightGreen},
		issue.StateClosed: {"[closed]", colorGray, colorLightGray},
	}

	for _, pIss := range issues {
		style := stateStyle[pIss.State]
		labels := ""
		if len(pIss.Labels) > 0 {
			labels = fmt.Sprintf(" [%s]", strings.Join(pIss.Labels, ", "))
		}

		// Updated time suffix
		dateSuffix := ""
		if !listNoDate {
			dateSuffix = fmt.Sprintf(" %s", colorize(formatRelativeTime(pIss.UpdatedAt), colorGray))
		}

		// 제목에 키워드 하이라이트 적용
		title := highlightKeyword(pIss.Title, keyword)
		// 상태별 밝은 색상을 제목에 적용
		title = colorize(title, style.titleColor)

		// 태그를 색상 적용 후 출력
		tag := colorize(fmt.Sprintf("%-8s", style.tag), style.color)
		// Use project/# format for multi-project mode
		ref := colorize(fmt.Sprintf("%-12s", pIss.Ref()), colorCyan)
		fmt.Printf("%s %s %s%s%s\n", tag, ref, title, labels, dateSuffix)
	}

	if skippedCount > 0 {
		fmt.Printf("\nTotal: %d issues (%d skipped)\n", len(issues), skippedCount)
	} else {
		fmt.Printf("\nTotal: %d issues\n", len(issues))
	}
}

// filterProjectIssuesBySearch filters project issues by keyword
func filterProjectIssuesBySearch(issues []*project.ProjectIssue, keyword string, titleOnly bool) []*project.ProjectIssue {
	keyword = strings.ToLower(keyword)
	var results []*project.ProjectIssue

	for _, pIss := range issues {
		if strings.Contains(strings.ToLower(pIss.Title), keyword) {
			results = append(results, pIss)
			continue
		}

		if !titleOnly && strings.Contains(strings.ToLower(pIss.Body), keyword) {
			results = append(results, pIss)
		}
	}

	return results
}

// filterProjectIssuesByDate filters project issues by date
func filterProjectIssuesByDate(issues []*project.ProjectIssue, filter *DateFilter) ([]*project.ProjectIssue, error) {
	// Convert to regular issues for filtering, then convert back
	regularIssues := make([]*issue.Issue, len(issues))
	issueMap := make(map[*issue.Issue]*project.ProjectIssue)
	for i, pIss := range issues {
		regularIssues[i] = pIss.Issue
		issueMap[pIss.Issue] = pIss
	}

	filtered, err := FilterIssuesByDate(regularIssues, filter)
	if err != nil {
		return nil, err
	}

	results := make([]*project.ProjectIssue, len(filtered))
	for i, iss := range filtered {
		results[i] = issueMap[iss]
	}
	return results, nil
}

// printMultiProjectWarnings prints warnings with project prefix
func printMultiProjectWarnings(warnings []project.ProjectWarning) {
	fmt.Println(colorize(fmt.Sprintf("\n⚠️  Parse failures (%d files):", len(warnings)), colorYellow))
	for _, w := range warnings {
		// Truncate filename if too long
		name := w.FileName
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		// Truncate error message
		errMsg := w.Error
		if len(errMsg) > 50 {
			errMsg = errMsg[:47] + "..."
		}
		projPrefix := colorize(fmt.Sprintf("[%s]", w.Project), colorCyan)
		fmt.Printf("  %s %s: %s\n", projPrefix, colorize("- "+name, colorGray), errMsg)
	}
	fmt.Println(colorize("\nRun 'zap repair --auto' to auto-fix with AI (requires claude/codex/gemini CLI)", colorGray))
}

func printParseWarnings(warnings []issue.ParseFailure) {
	fmt.Println(colorize(fmt.Sprintf("\n⚠️  Parse failures (%d files):", len(warnings)), colorYellow))
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
		fmt.Printf("  %s: %s\n", colorize("- "+name, colorGray), errMsg)
	}
	fmt.Println(colorize("\nRun 'zap repair --auto' to auto-fix with AI (requires claude/codex/gemini CLI)", colorGray))
}

// filterBySearch filters issues by keyword in title and/or body
func filterBySearch(issues []*issue.Issue, keyword string, titleOnly bool) []*issue.Issue {
	keyword = strings.ToLower(keyword)
	var results []*issue.Issue

	for _, iss := range issues {
		if strings.Contains(strings.ToLower(iss.Title), keyword) {
			results = append(results, iss)
			continue
		}

		if !titleOnly && strings.Contains(strings.ToLower(iss.Body), keyword) {
			results = append(results, iss)
		}
	}

	return results
}

// highlightKeyword highlights the keyword in text with ANSI bold
func highlightKeyword(text, keyword string) string {
	if !colorEnabled || keyword == "" {
		return text
	}

	lower := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)

	idx := strings.Index(lower, lowerKeyword)
	if idx == -1 {
		return text
	}

	before := text[:idx]
	match := text[idx : idx+len(keyword)]
	after := text[idx+len(keyword):]

	return before + "\033[1m" + match + colorReset + after
}

// getRecentlyClosedIssues returns done/closed issues that were updated within the given duration
func getRecentlyClosedIssues(store *issue.Store, duration time.Duration, labelFilter, assigneeFilter string) ([]*issue.Issue, error) {
	closedStates := []issue.State{issue.StateDone, issue.StateClosed}

	var issues []*issue.Issue
	var err error

	if labelFilter != "" {
		issues, err = store.FilterByLabel(labelFilter, closedStates...)
	} else if assigneeFilter != "" {
		issues, err = store.FilterByAssignee(assigneeFilter, closedStates...)
	} else {
		issues, err = store.List(closedStates...)
	}

	if err != nil {
		return nil, err
	}

	// Filter to only recently closed issues
	var recentIssues []*issue.Issue
	for _, iss := range issues {
		if isRecentlyClosed(iss.UpdatedAt, string(iss.State), duration) {
			recentIssues = append(recentIssues, iss)
		}
	}

	return recentIssues, nil
}

// mergeIssues merges two issue slices, avoiding duplicates based on issue number
func mergeIssues(base, additional []*issue.Issue) []*issue.Issue {
	seen := make(map[int]bool)
	for _, iss := range base {
		seen[iss.Number] = true
	}

	result := make([]*issue.Issue, len(base))
	copy(result, base)

	for _, iss := range additional {
		if !seen[iss.Number] {
			result = append(result, iss)
			seen[iss.Number] = true
		}
	}

	return result
}
