package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/ai"
	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var releaseNotesCmd = &cobra.Command{
	Use:    "release-notes [from-ref] [to-ref]",
	Short:  "Generate release notes from commit logs using AI",
	Hidden: true, // Development tool, not for end users
	Long:   `Generate release notes by analyzing commit logs between two git references.

The command collects:
- Commit messages between the specified refs
- File change statistics (added/modified/deleted)
- Related issues from .issues/ directory

Then uses an AI CLI tool (claude, codex, or gemini) to generate
well-formatted release notes.

Examples:
  zap release-notes                    # latest tag to HEAD
  zap release-notes v0.6.6             # v0.6.6 to HEAD
  zap release-notes v0.6.5 v0.6.6      # v0.6.5 to v0.6.6
  zap release-notes --output RELEASE.md`,
	Args: cobra.MaximumNArgs(2),
	RunE: runReleaseNotes,
}

var (
	releaseNotesOutput  string
	releaseNotesTimeout time.Duration
)

func init() {
	rootCmd.AddCommand(releaseNotesCmd)

	releaseNotesCmd.Flags().StringVarP(&releaseNotesOutput, "output", "o", "", "Write output to file instead of stdout")
	releaseNotesCmd.Flags().DurationVar(&releaseNotesTimeout, "timeout", 120*time.Second, "AI request timeout")
}

func runReleaseNotes(cmd *cobra.Command, args []string) error {
	// Determine refs
	fromRef, toRef, err := resolveRefs(args)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "📝 Generating release notes: %s...%s\n", fromRef, toRef)

	// Collect git information
	commits, err := getCommitLogs(fromRef, toRef)
	if err != nil {
		return fmt.Errorf("failed to get commit logs: %w", err)
	}

	if len(commits) == 0 {
		return fmt.Errorf("no commits found between %s and %s", fromRef, toRef)
	}

	stats, err := getFileStats(fromRef, toRef)
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}

	// Get related issues
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	relatedIssues := findRelatedIssues(dir, commits)

	// Build context for AI
	contextData := buildReleaseContext(fromRef, toRef, commits, stats, relatedIssues)

	// Generate release notes using AI
	notes, err := generateReleaseNotesWithAI(contextData)
	if err != nil {
		return fmt.Errorf("failed to generate release notes: %w", err)
	}

	// Output
	if releaseNotesOutput != "" {
		if err := os.WriteFile(releaseNotesOutput, []byte(notes), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✅ Release notes written to %s\n", releaseNotesOutput)
	} else {
		fmt.Println(notes)
	}

	return nil
}

// resolveRefs determines the from and to refs based on arguments.
func resolveRefs(args []string) (string, string, error) {
	toRef := "HEAD"

	switch len(args) {
	case 0:
		// Find latest tag
		latestTag, err := getLatestTag()
		if err != nil {
			return "", "", fmt.Errorf("failed to find latest tag: %w", err)
		}
		return latestTag, toRef, nil
	case 1:
		return args[0], toRef, nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", fmt.Errorf("too many arguments")
	}
}

// getLatestTag returns the most recent git tag.
func getLatestTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no tags found: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// CommitInfo holds information about a single commit.
type CommitInfo struct {
	Hash    string
	Subject string
	Body    string
	Author  string
	Date    string
}

// getCommitLogs retrieves commit information between two refs.
func getCommitLogs(fromRef, toRef string) ([]CommitInfo, error) {
	// Format: hash|subject|body|author|date
	// Using %x00 as separator to handle multiline bodies
	format := "%H%x00%s%x00%b%x00%an%x00%ad%x00%x01"
	cmd := exec.Command("git", "log", "--date=short", fmt.Sprintf("--format=%s", format), fmt.Sprintf("%s..%s", fromRef, toRef))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	entries := strings.Split(string(output), "\x01")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, "\x00")
		if len(parts) < 5 {
			continue
		}

		commits = append(commits, CommitInfo{
			Hash:    parts[0][:8], // Short hash
			Subject: parts[1],
			Body:    strings.TrimSpace(parts[2]),
			Author:  parts[3],
			Date:    parts[4],
		})
	}

	return commits, nil
}

// FileStats holds file change statistics.
type FileStats struct {
	Added    int
	Modified int
	Deleted  int
	Files    []string
}

// getFileStats retrieves file change statistics between two refs.
func getFileStats(fromRef, toRef string) (*FileStats, error) {
	// Get list of changed files with status
	cmd := exec.Command("git", "diff", "--name-status", fmt.Sprintf("%s..%s", fromRef, toRef))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	stats := &FileStats{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		file := parts[len(parts)-1]
		stats.Files = append(stats.Files, file)

		switch status[0] {
		case 'A':
			stats.Added++
		case 'D':
			stats.Deleted++
		case 'M', 'R':
			stats.Modified++
		}
	}

	return stats, nil
}

// findRelatedIssues finds issues that may be related to the commits.
func findRelatedIssues(issuesDir string, commits []CommitInfo) []*issue.Issue {
	store := issue.NewStore(issuesDir)
	allIssues, err := store.List(issue.AllStates()...)
	if err != nil {
		return nil
	}

	// Extract issue numbers mentioned in commits
	issueNumbers := make(map[int]bool)
	issuePattern := regexp.MustCompile(`#(\d+)`)

	for _, c := range commits {
		// Check subject and body for issue references
		for _, match := range issuePattern.FindAllStringSubmatch(c.Subject+" "+c.Body, -1) {
			if num, err := strconv.Atoi(match[1]); err == nil {
				issueNumbers[num] = true
			}
		}
	}

	// Filter issues
	var related []*issue.Issue
	for _, iss := range allIssues {
		if issueNumbers[iss.Number] {
			related = append(related, iss)
		}
	}

	return related
}

// buildReleaseContext builds the context string for AI.
func buildReleaseContext(fromRef, toRef string, commits []CommitInfo, stats *FileStats, issues []*issue.Issue) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Release: %s → %s\n\n", fromRef, toRef))

	// Commits section
	sb.WriteString("## Commits\n\n")
	for _, c := range commits {
		sb.WriteString(fmt.Sprintf("- %s: %s (%s, %s)\n", c.Hash, c.Subject, c.Author, c.Date))
		if c.Body != "" {
			// Indent body
			for _, line := range strings.Split(c.Body, "\n") {
				if line != "" {
					sb.WriteString(fmt.Sprintf("  %s\n", line))
				}
			}
		}
	}
	sb.WriteString("\n")

	// File stats section
	sb.WriteString("## File Changes\n\n")
	sb.WriteString(fmt.Sprintf("- Added: %d files\n", stats.Added))
	sb.WriteString(fmt.Sprintf("- Modified: %d files\n", stats.Modified))
	sb.WriteString(fmt.Sprintf("- Deleted: %d files\n", stats.Deleted))
	sb.WriteString(fmt.Sprintf("- Total: %d files changed\n\n", len(stats.Files)))

	// Major file changes (group by directory)
	if len(stats.Files) > 0 {
		sb.WriteString("### Changed Files\n\n")
		for _, f := range stats.Files {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	// Related issues section
	if len(issues) > 0 {
		sb.WriteString("## Related Issues\n\n")
		for _, iss := range issues {
			sb.WriteString(fmt.Sprintf("- #%d: %s (%s)\n", iss.Number, iss.Title, iss.State))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateReleaseNotesWithAI uses AI to generate formatted release notes.
func generateReleaseNotesWithAI(contextData string) (string, error) {
	// Load AI config
	cfg, err := ai.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}

	// Get AI client
	client, err := ai.GetClient(cfg)
	if err != nil {
		return "", fmt.Errorf("no AI CLI available (install claude, codex, or gemini): %w", err)
	}

	fmt.Fprintf(os.Stderr, "🤖 Using %s to generate release notes...\n", client.Name())

	// Build prompt
	systemPrompt := `You are a technical writer creating release notes for a software project.
Generate clean, well-organized release notes in Markdown format.

Guidelines:
- Start with a brief summary (1-2 sentences)
- Group changes by category (Features, Bug Fixes, Improvements, etc.)
- Use bullet points for individual changes
- Keep descriptions concise but informative
- Reference issue numbers where mentioned (e.g., #123)
- Do not include the raw commit hashes
- Write in a professional tone
- Output ONLY the release notes content, no additional commentary`

	userPrompt := fmt.Sprintf(`Based on the following git information, generate professional release notes:

%s

Generate the release notes now:`, contextData)

	ctx, cancel := context.WithTimeout(context.Background(), releaseNotesTimeout)
	defer cancel()

	resp, err := client.Complete(ctx, &ai.Request{
		System: systemPrompt,
		Prompt: userPrompt,
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
