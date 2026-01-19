package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/ai"
	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report [commit-range | issue-numbers...]",
	Aliases: []string{"rep"},
	Short: "Generate work report for team sharing",
	Long: `Generate a work report by analyzing commits and issues within a specified period.

The report includes:
- Commit list with linked issues
- Issue progress status (done, wip, open)
- File change statistics
- AI-generated summary

Examples:
  # Report for last week
  zap report --since 2025-01-13 --until 2025-01-19

  # Report for today
  zap report --today

  # Report for last 7 days
  zap report --days 7

  # Report for commit range
  zap report v1.0..HEAD

  # Report for specific issues
  zap report 10 11 12

  # Output to file
  zap report --days 7 -o report.md

  # JSON format
  zap report --days 7 --format json`,
	RunE: runReport,
}

var (
	reportFormat     string
	reportOutput     string
	reportAI         string
	reportTimeout    time.Duration
	reportDateFilter DateFilter
	reportNoAI       bool
)

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&reportFormat, "format", "f", "markdown", "Output format (markdown, text, json)")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Write output to file instead of stdout")
	reportCmd.Flags().StringVar(&reportAI, "ai", "", "AI provider to use (claude, codex, gemini)")
	reportCmd.Flags().DurationVar(&reportTimeout, "timeout", 120*time.Second, "AI request timeout")
	reportCmd.Flags().BoolVar(&reportNoAI, "no-ai", false, "Skip AI summary generation")

	// Date filter options
	reportCmd.Flags().BoolVar(&reportDateFilter.Today, "today", false, "Report for today")
	reportCmd.Flags().StringVar(&reportDateFilter.Since, "since", "", "Report since date (YYYY-MM-DD)")
	reportCmd.Flags().StringVar(&reportDateFilter.Until, "until", "", "Report until date (YYYY-MM-DD)")
	reportCmd.Flags().IntVar(&reportDateFilter.Days, "days", 0, "Report for last N days")
	reportCmd.Flags().IntVar(&reportDateFilter.Weeks, "weeks", 0, "Report for last N weeks")
}

// ReportData holds all data for report generation.
type ReportData struct {
	Period     string
	Since      time.Time
	Until      time.Time
	Summary    string
	Commits    []CommitInfo
	Issues     []*issue.Issue
	IssueLinks map[int][]CommitInfo // issue number -> related commits
	FileStats  *FileStats
}

func runReport(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	// Determine report mode based on arguments
	var reportData *ReportData

	if len(args) > 0 {
		// Check if first arg looks like a commit range (contains "..")
		if strings.Contains(args[0], "..") {
			// Commit range mode
			reportData, err = buildReportFromCommitRange(store, args[0])
		} else if isNumeric(args[0]) {
			// Issue numbers mode
			reportData, err = buildReportFromIssueNumbers(store, args)
		} else {
			// Assume single ref to HEAD
			reportData, err = buildReportFromCommitRange(store, args[0]+"..HEAD")
		}
	} else if !reportDateFilter.IsEmpty() {
		// Date filter mode
		reportData, err = buildReportFromDateFilter(store, &reportDateFilter)
	} else {
		return fmt.Errorf("please specify a date range (--since, --days, etc.), commit range (v1.0..HEAD), or issue numbers")
	}

	if err != nil {
		return err
	}

	// Generate AI summary if not disabled and there's content to summarize
	if !reportNoAI && (len(reportData.Commits) > 0 || len(reportData.Issues) > 0) {
		fmt.Fprintf(os.Stderr, "🤖 Generating AI summary...\n")
		summary, aiErr := generateReportSummary(reportData)
		if aiErr != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to generate AI summary: %v\n", aiErr)
		} else {
			reportData.Summary = summary
		}
	}

	// Format output
	var output string
	switch reportFormat {
	case "json":
		data, err := formatReportJSON(reportData)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		output = string(data)
	case "text":
		output = formatReportText(reportData)
	default:
		output = formatReportMarkdown(reportData)
	}

	// Write output
	if reportOutput != "" {
		if err := os.WriteFile(reportOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✅ Report written to %s\n", reportOutput)
	} else {
		fmt.Println(output)
	}

	return nil
}

// isNumeric checks if a string is a number.
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// buildReportFromDateFilter builds report from date filter.
func buildReportFromDateFilter(store *issue.Store, filter *DateFilter) (*ReportData, error) {
	since, until, err := filter.GetDateRange()
	if err != nil {
		return nil, err
	}

	// If until is zero, use now
	if until.IsZero() {
		until = time.Now()
	}

	return buildReportForPeriod(store, since, until)
}

// buildReportFromCommitRange builds report from git commit range.
func buildReportFromCommitRange(store *issue.Store, commitRange string) (*ReportData, error) {
	parts := strings.Split(commitRange, "..")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid commit range format: %s (expected from..to)", commitRange)
	}

	fromRef, toRef := parts[0], parts[1]

	// Get commits in range
	commits, err := getCommitLogs(fromRef, toRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found in range %s", commitRange)
	}

	// Determine date range from commits
	var since, until time.Time
	for _, c := range commits {
		t, _ := time.Parse("2006-01-02", c.Date)
		if since.IsZero() || t.Before(since) {
			since = t
		}
		if until.IsZero() || t.After(until) {
			until = t
		}
	}

	// Get file stats
	stats, err := getFileStats(fromRef, toRef)
	if err != nil {
		stats = &FileStats{}
	}

	// Get all issues and link to commits
	allIssues, err := store.List(issue.AllStates()...)
	if err != nil {
		allIssues = nil
	}

	issueLinks := linkCommitsToIssues(commits, allIssues)
	relatedIssues := getRelatedIssues(allIssues, issueLinks)

	return &ReportData{
		Period:     fmt.Sprintf("%s ~ %s", since.Format("2006-01-02"), until.Format("2006-01-02")),
		Since:      since,
		Until:      until,
		Commits:    commits,
		Issues:     relatedIssues,
		IssueLinks: issueLinks,
		FileStats:  stats,
	}, nil
}

// buildReportFromIssueNumbers builds report from specific issue numbers.
func buildReportFromIssueNumbers(store *issue.Store, args []string) (*ReportData, error) {
	var issues []*issue.Issue

	for _, arg := range args {
		num, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid issue number: %s", arg)
		}
		iss, err := store.Get(num)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Issue #%d not found\n", num)
			continue
		}
		issues = append(issues, iss)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("no valid issues found")
	}

	// Get all commits and filter by issue references
	// Try to find commits from last 30 days
	filter := &DateFilter{Days: 30}
	since, until, _ := filter.GetDateRange()
	commits, _ := getCommitsInDateRange(since, until)

	// Filter commits that reference these issues
	issueNumbers := make(map[int]bool)
	for _, iss := range issues {
		issueNumbers[iss.Number] = true
	}

	var relatedCommits []CommitInfo
	for _, c := range commits {
		refs := extractIssueRefs(c.Subject + " " + c.Body)
		for _, ref := range refs {
			if issueNumbers[ref] {
				relatedCommits = append(relatedCommits, c)
				break
			}
		}
	}

	issueLinks := linkCommitsToIssues(relatedCommits, issues)

	return &ReportData{
		Period:     fmt.Sprintf("Issues: %s", strings.Join(args, ", ")),
		Since:      since,
		Until:      until,
		Commits:    relatedCommits,
		Issues:     issues,
		IssueLinks: issueLinks,
		FileStats:  &FileStats{},
	}, nil
}

// buildReportForPeriod builds report for a specific date range.
func buildReportForPeriod(store *issue.Store, since, until time.Time) (*ReportData, error) {
	// Get commits in date range
	commits, err := getCommitsInDateRange(since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	// Get file stats (use first and last commit if available)
	var stats *FileStats
	if len(commits) > 0 {
		firstHash := commits[len(commits)-1].Hash
		lastHash := commits[0].Hash
		stats, _ = getFileStats(firstHash+"^", lastHash)
	}
	if stats == nil {
		stats = &FileStats{}
	}

	// Get all issues and filter by date
	allIssues, err := store.List(issue.AllStates()...)
	if err != nil {
		allIssues = nil
	}

	// Filter issues updated in this period
	var periodIssues []*issue.Issue
	for _, iss := range allIssues {
		if matchesDateRange(iss.UpdatedAt, since, until) || matchesDateRange(iss.CreatedAt, since, until) {
			periodIssues = append(periodIssues, iss)
		}
	}

	issueLinks := linkCommitsToIssues(commits, allIssues)

	// Get issues that are either in the period OR linked from commits
	relatedFromCommits := getRelatedIssues(allIssues, issueLinks)
	issueMap := make(map[int]*issue.Issue)
	for _, iss := range periodIssues {
		issueMap[iss.Number] = iss
	}
	for _, iss := range relatedFromCommits {
		issueMap[iss.Number] = iss
	}

	var finalIssues []*issue.Issue
	for _, iss := range issueMap {
		finalIssues = append(finalIssues, iss)
	}
	// Sort by number
	sort.Slice(finalIssues, func(i, j int) bool {
		return finalIssues[i].Number < finalIssues[j].Number
	})

	return &ReportData{
		Period:     fmt.Sprintf("%s ~ %s", since.Format("2006-01-02"), until.Format("2006-01-02")),
		Since:      since,
		Until:      until,
		Commits:    commits,
		Issues:     finalIssues,
		IssueLinks: issueLinks,
		FileStats:  stats,
	}, nil
}

// getCommitsInDateRange gets commits within a date range.
func getCommitsInDateRange(since, until time.Time) ([]CommitInfo, error) {
	args := []string{"log", "--date=short", "--format=%H%x00%s%x00%b%x00%an%x00%ad%x00%x01"}

	if !since.IsZero() {
		args = append(args, "--since="+since.Format("2006-01-02"))
	}
	if !until.IsZero() {
		args = append(args, "--until="+until.Add(24*time.Hour).Format("2006-01-02"))
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseCommitOutput(string(output))
}

// parseCommitOutput parses git log output into CommitInfo slice.
func parseCommitOutput(output string) ([]CommitInfo, error) {
	var commits []CommitInfo
	entries := strings.Split(output, "\x01")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, "\x00")
		if len(parts) < 5 {
			continue
		}

		hash := parts[0]
		if len(hash) > 8 {
			hash = hash[:8]
		}

		commits = append(commits, CommitInfo{
			Hash:    hash,
			Subject: parts[1],
			Body:    strings.TrimSpace(parts[2]),
			Author:  parts[3],
			Date:    parts[4],
		})
	}

	return commits, nil
}

// extractIssueRefs extracts issue numbers from text (#N pattern).
func extractIssueRefs(text string) []int {
	pattern := regexp.MustCompile(`#(\d+)`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	var refs []int
	seen := make(map[int]bool)
	for _, match := range matches {
		if num, err := strconv.Atoi(match[1]); err == nil && !seen[num] {
			refs = append(refs, num)
			seen[num] = true
		}
	}
	return refs
}

// linkCommitsToIssues creates a mapping from issue numbers to related commits.
func linkCommitsToIssues(commits []CommitInfo, issues []*issue.Issue) map[int][]CommitInfo {
	result := make(map[int][]CommitInfo)

	// Build set of valid issue numbers
	validIssues := make(map[int]bool)
	for _, iss := range issues {
		validIssues[iss.Number] = true
	}

	for _, c := range commits {
		refs := extractIssueRefs(c.Subject + " " + c.Body)
		for _, ref := range refs {
			if validIssues[ref] {
				result[ref] = append(result[ref], c)
			}
		}
	}

	return result
}

// getRelatedIssues returns issues that are referenced by commits.
func getRelatedIssues(allIssues []*issue.Issue, links map[int][]CommitInfo) []*issue.Issue {
	var related []*issue.Issue
	for _, iss := range allIssues {
		if _, ok := links[iss.Number]; ok {
			related = append(related, iss)
		}
	}
	return related
}

// formatReportMarkdown formats report as Markdown.
func formatReportMarkdown(data *ReportData) string {
	var sb strings.Builder

	sb.WriteString("# 작업 보고서\n")
	sb.WriteString(fmt.Sprintf("> 기간: %s\n\n", data.Period))

	// Summary section
	if data.Summary != "" {
		sb.WriteString("## 요약\n")
		sb.WriteString(data.Summary + "\n\n")
	}

	// Commits section
	if len(data.Commits) > 0 {
		sb.WriteString(fmt.Sprintf("## 커밋 (%d건)\n", len(data.Commits)))
		sb.WriteString("| 해시 | 메시지 | 관련 이슈 |\n")
		sb.WriteString("|------|--------|----------|\n")

		for _, c := range data.Commits {
			refs := extractIssueRefs(c.Subject + " " + c.Body)
			refStr := "-"
			if len(refs) > 0 {
				var refStrs []string
				for _, r := range refs {
					refStrs = append(refStrs, fmt.Sprintf("#%d", r))
				}
				refStr = strings.Join(refStrs, ", ")
			}
			// Escape pipe characters in subject
			subject := strings.ReplaceAll(c.Subject, "|", "\\|")
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", c.Hash, subject, refStr))
		}
		sb.WriteString("\n")
	}

	// Issues section
	if len(data.Issues) > 0 {
		sb.WriteString("## 이슈 진행 상황\n")

		// Group by state
		byState := make(map[issue.State][]*issue.Issue)
		for _, iss := range data.Issues {
			byState[iss.State] = append(byState[iss.State], iss)
		}

		stateOrder := []issue.State{issue.StateDone, issue.StateWip, issue.StateOpen, issue.StateClosed}
		stateNames := map[issue.State]string{
			issue.StateDone:   "완료 (done)",
			issue.StateWip:    "진행 중 (wip)",
			issue.StateOpen:   "신규 (open)",
			issue.StateClosed: "취소 (closed)",
		}

		for _, state := range stateOrder {
			issues := byState[state]
			if len(issues) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("### %s\n", stateNames[state]))
			for _, iss := range issues {
				sb.WriteString(fmt.Sprintf("- #%d: %s\n", iss.Number, iss.Title))
			}
			sb.WriteString("\n")
		}
	}

	// File stats section
	if data.FileStats != nil && len(data.FileStats.Files) > 0 {
		sb.WriteString("## 파일 변경 통계\n")
		sb.WriteString(fmt.Sprintf("- 추가: %d개 파일\n", data.FileStats.Added))
		sb.WriteString(fmt.Sprintf("- 수정: %d개 파일\n", data.FileStats.Modified))
		sb.WriteString(fmt.Sprintf("- 삭제: %d개 파일\n", data.FileStats.Deleted))

		// Find major change area
		dirCounts := make(map[string]int)
		for _, f := range data.FileStats.Files {
			parts := strings.Split(f, "/")
			if len(parts) > 1 {
				dirCounts[parts[0]+"/"]++
			}
		}
		if len(dirCounts) > 0 {
			var maxDir string
			var maxCount int
			for dir, count := range dirCounts {
				if count > maxCount {
					maxDir = dir
					maxCount = count
				}
			}
			sb.WriteString(fmt.Sprintf("- 주요 변경 영역: %s\n", maxDir))
		}
	}

	return sb.String()
}

// formatReportText formats report as plain text.
func formatReportText(data *ReportData) string {
	var sb strings.Builder

	sb.WriteString("작업 보고서\n")
	sb.WriteString(fmt.Sprintf("기간: %s\n", data.Period))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	if data.Summary != "" {
		sb.WriteString("요약:\n")
		sb.WriteString(data.Summary + "\n\n")
	}

	if len(data.Commits) > 0 {
		sb.WriteString(fmt.Sprintf("커밋 (%d건):\n", len(data.Commits)))
		for _, c := range data.Commits {
			refs := extractIssueRefs(c.Subject + " " + c.Body)
			refStr := ""
			if len(refs) > 0 {
				var refStrs []string
				for _, r := range refs {
					refStrs = append(refStrs, fmt.Sprintf("#%d", r))
				}
				refStr = " [" + strings.Join(refStrs, ", ") + "]"
			}
			sb.WriteString(fmt.Sprintf("  %s: %s%s\n", c.Hash, c.Subject, refStr))
		}
		sb.WriteString("\n")
	}

	if len(data.Issues) > 0 {
		sb.WriteString("이슈 진행 상황:\n")
		for _, iss := range data.Issues {
			sb.WriteString(fmt.Sprintf("  [%s] #%d: %s\n", iss.State, iss.Number, iss.Title))
		}
		sb.WriteString("\n")
	}

	if data.FileStats != nil && len(data.FileStats.Files) > 0 {
		sb.WriteString("파일 변경 통계:\n")
		sb.WriteString(fmt.Sprintf("  추가: %d, 수정: %d, 삭제: %d\n",
			data.FileStats.Added, data.FileStats.Modified, data.FileStats.Deleted))
	}

	return sb.String()
}

// ReportJSON is the JSON output structure.
type ReportJSON struct {
	Period    string            `json:"period"`
	Since     string            `json:"since"`
	Until     string            `json:"until"`
	Summary   string            `json:"summary,omitempty"`
	Commits   []CommitJSON      `json:"commits"`
	Issues    []IssueJSON       `json:"issues"`
	FileStats FileStatsJSON     `json:"file_stats"`
}

// CommitJSON is the JSON structure for a commit.
type CommitJSON struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Issues  []int  `json:"issues,omitempty"`
}

// IssueJSON is the JSON structure for an issue.
type IssueJSON struct {
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	State   string   `json:"state"`
	Labels  []string `json:"labels,omitempty"`
	Commits []string `json:"commits,omitempty"`
}

// FileStatsJSON is the JSON structure for file stats.
type FileStatsJSON struct {
	Added    int      `json:"added"`
	Modified int      `json:"modified"`
	Deleted  int      `json:"deleted"`
	Files    []string `json:"files,omitempty"`
}

// formatReportJSON formats report as JSON.
func formatReportJSON(data *ReportData) ([]byte, error) {
	report := ReportJSON{
		Period:  data.Period,
		Since:   data.Since.Format("2006-01-02"),
		Until:   data.Until.Format("2006-01-02"),
		Summary: data.Summary,
	}

	// Commits
	for _, c := range data.Commits {
		cj := CommitJSON{
			Hash:    c.Hash,
			Subject: c.Subject,
			Author:  c.Author,
			Date:    c.Date,
			Issues:  extractIssueRefs(c.Subject + " " + c.Body),
		}
		report.Commits = append(report.Commits, cj)
	}

	// Issues
	for _, iss := range data.Issues {
		ij := IssueJSON{
			Number: iss.Number,
			Title:  iss.Title,
			State:  string(iss.State),
			Labels: iss.Labels,
		}
		// Add linked commits
		if commits, ok := data.IssueLinks[iss.Number]; ok {
			for _, c := range commits {
				ij.Commits = append(ij.Commits, c.Hash)
			}
		}
		report.Issues = append(report.Issues, ij)
	}

	// File stats
	if data.FileStats != nil {
		report.FileStats = FileStatsJSON{
			Added:    data.FileStats.Added,
			Modified: data.FileStats.Modified,
			Deleted:  data.FileStats.Deleted,
			Files:    data.FileStats.Files,
		}
	}

	return json.MarshalIndent(report, "", "  ")
}

// generateReportSummary generates an AI summary of the report.
func generateReportSummary(data *ReportData) (string, error) {
	client, err := getAIClient(reportAI)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "🤖 Using %s to generate summary...\n", client.Name())

	// Build context for AI
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("기간: %s\n\n", data.Period))

	if len(data.Commits) > 0 {
		sb.WriteString("## 커밋 목록\n")
		for _, c := range data.Commits {
			refs := extractIssueRefs(c.Subject + " " + c.Body)
			refStr := ""
			if len(refs) > 0 {
				var refStrs []string
				for _, r := range refs {
					refStrs = append(refStrs, fmt.Sprintf("#%d", r))
				}
				refStr = " (" + strings.Join(refStrs, ", ") + ")"
			}
			sb.WriteString(fmt.Sprintf("- %s: %s%s\n", c.Hash, c.Subject, refStr))
		}
		sb.WriteString("\n")
	}

	if len(data.Issues) > 0 {
		sb.WriteString("## 이슈 상태\n")
		for _, iss := range data.Issues {
			sb.WriteString(fmt.Sprintf("- #%d [%s]: %s\n", iss.Number, iss.State, iss.Title))
		}
	}

	systemPrompt := `당신은 개발팀의 작업 보고서를 작성하는 테크니컬 라이터입니다.
주어진 커밋과 이슈 정보를 바탕으로 팀 공유용 요약을 작성하세요.

규칙:
- 한국어로 작성
- 2-3문장으로 핵심 성과 요약
- 주요 변경 사항 강조
- 전문적이고 간결한 톤 유지
- 추가 설명이나 코멘트 없이 요약만 출력`

	userPrompt := fmt.Sprintf(`다음은 %s 동안의 작업 내역입니다.

%s

위 내용을 바탕으로 팀 공유용 보고서 요약을 작성해주세요.`, data.Period, sb.String())

	ctx, cancel := context.WithTimeout(context.Background(), reportTimeout)
	defer cancel()

	resp, err := client.Complete(ctx, &ai.Request{
		System: systemPrompt,
		Prompt: userPrompt,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}
