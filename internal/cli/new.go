package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/spf13/cobra"
	"golang.org/x/text/unicode/norm"
)

var newCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new issue with proper format",
	Long: `Create a new issue file with the correct frontmatter format.

This command ensures issues are created with valid YAML frontmatter
that can be properly parsed by 'zap list' and other commands.

Examples:
  zap new "Fix login bug"
  zap new "Add user authentication" -l enhancement -l priority-high
  zap new "Refactor database layer" -a alice -a bob
  zap new "Update docs" --body "Need to update API documentation"
  echo "Issue description" | zap new "New feature"
  zap new "Complex issue" --editor`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

var (
	newLabels    []string
	newAssignees []string
	newBody      string
	newEditor    bool
	newState     string
	newProject   string
)

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringArrayVarP(&newLabels, "label", "l", nil, "Add label (can be used multiple times)")
	newCmd.Flags().StringArrayVarP(&newAssignees, "assignee", "a", nil, "Add assignee (can be used multiple times)")
	newCmd.Flags().StringVarP(&newBody, "body", "b", "", "Issue body content")
	newCmd.Flags().BoolVarP(&newEditor, "editor", "e", false, "Open editor to write issue body")
	newCmd.Flags().StringVarP(&newState, "state", "s", "open", "Initial state (open, in-progress, done, closed)")
	newCmd.Flags().StringVarP(&newProject, "project", "p", "", "Project alias (required for multi-project mode)")
}

func runNew(cmd *cobra.Command, args []string) error {
	title := strings.TrimSpace(args[0])
	if title == "" {
		return fmt.Errorf("title cannot be empty")
	}

	// Validate state
	state, ok := issue.ParseState(newState)
	if !ok {
		return fmt.Errorf("invalid state: %s (valid: open, in-progress, done, closed)", newState)
	}

	// Check for multi-project mode
	if isMultiProjectMode(cmd) {
		// Multi-project mode requires --project flag
		if newProject == "" {
			return fmt.Errorf("--project flag is required when using multiple -C flags")
		}

		multiStore, err := getMultiStore(cmd)
		if err != nil {
			return err
		}

		proj, ok := multiStore.GetProject(newProject)
		if !ok {
			return fmt.Errorf("project not found: %s", newProject)
		}

		issuesDir, _ := cmd.Flags().GetString("dir")
		return createIssueInProject(proj, issuesDir, title, state)
	}

	// Single project mode (existing behavior)
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	// Ensure issues directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create issues directory: %w", err)
	}

	store := issue.NewStore(dir)

	// Find next issue number
	nextNumber, err := findNextIssueNumber(store)
	if err != nil {
		return fmt.Errorf("failed to determine next issue number: %w", err)
	}

	// Determine body content
	body := newBody

	// Check for stdin input (piped content)
	// Only read from stdin if data is actually being piped
	if body == "" && !newEditor {
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeNamedPipe) != 0 {
			// Data is being piped
			scanner := bufio.NewScanner(os.Stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			body = strings.Join(lines, "\n")
		}
	}

	// Open editor if requested
	if newEditor {
		editedBody, err := openEditor(body)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
		body = editedBody
	}

	// Create issue struct
	now := time.Now()
	iss := &issue.Issue{
		Number:    nextNumber,
		Title:     title,
		State:     state,
		Labels:    newLabels,
		Assignees: newAssignees,
		CreatedAt: now,
		UpdatedAt: now,
		Body:      strings.TrimSpace(body),
	}

	// Generate filename
	slug := generateSlug(title)
	filename := fmt.Sprintf("%03d-%s.md", nextNumber, slug)
	filePath := filepath.Join(dir, filename)

	// Serialize issue
	data, err := issue.Serialize(iss)
	if err != nil {
		return fmt.Errorf("failed to serialize issue: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write issue file: %w", err)
	}

	fmt.Printf("✅ Created issue #%d: %s\n", nextNumber, filename)
	return nil
}

// findNextIssueNumber finds the next available issue number.
// It considers both successfully parsed issues and parse failures.
func findNextIssueNumber(store *issue.Store) (int, error) {
	// Load all issues (this also populates warnings)
	issues, err := store.List(issue.AllStates()...)
	if err != nil {
		return 0, err
	}

	maxNumber := 0

	// Check parsed issues
	for _, iss := range issues {
		if iss.Number > maxNumber {
			maxNumber = iss.Number
		}
	}

	// Check parse failures (extract number from filename)
	for _, w := range store.Warnings() {
		if num := extractNumberFromFilename(w.FileName); num > maxNumber {
			maxNumber = num
		}
	}

	return maxNumber + 1, nil
}

// extractNumberFromFilename extracts the issue number from a filename.
// Supports formats: "NNN-title.md", "N-title.md", etc.
func extractNumberFromFilename(filename string) int {
	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Find the first hyphen
	idx := strings.Index(name, "-")
	if idx == -1 {
		return 0
	}

	// Try to parse the number part
	numStr := name[:idx]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}

	return num
}

// generateSlug creates a URL-friendly slug from the title.
// Supports Korean and other Unicode characters.
func generateSlug(title string) string {
	// Normalize Unicode (NFC)
	title = norm.NFC.String(title)

	// Convert to lowercase
	title = strings.ToLower(title)

	// Remove common prefixes like "feat:", "fix:", "docs:", etc.
	prefixPattern := regexp.MustCompile(`^(feat|fix|docs|chore|refactor|test|style|perf|ci|build):\s*`)
	title = prefixPattern.ReplaceAllString(title, "")

	// Replace spaces and underscores with hyphens
	title = strings.ReplaceAll(title, " ", "-")
	title = strings.ReplaceAll(title, "_", "-")

	// Keep only alphanumeric, Korean, and hyphens
	var result strings.Builder
	prevHyphen := false
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			prevHyphen = false
		} else if r == '-' && !prevHyphen && result.Len() > 0 {
			result.WriteRune('-')
			prevHyphen = true
		}
	}

	slug := result.String()

	// Remove trailing hyphen
	slug = strings.TrimSuffix(slug, "-")

	// Limit length to 50 characters (cut at word boundary if possible)
	if len(slug) > 50 {
		// Try to cut at a hyphen boundary
		truncated := slug[:50]
		lastHyphen := strings.LastIndex(truncated, "-")
		if lastHyphen > 30 {
			slug = truncated[:lastHyphen]
		} else {
			slug = truncated
		}
	}

	// Ensure slug is not empty
	if slug == "" {
		slug = "issue"
	}

	return slug
}

// openEditor opens the user's preferred editor for writing the issue body.
func openEditor(initialContent string) (string, error) {
	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Default editors based on platform
		editor = "vi"
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "zap-issue-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write initial content
	if initialContent != "" {
		if _, err := tmpFile.WriteString(initialContent); err != nil {
			tmpFile.Close()
			return "", fmt.Errorf("failed to write initial content: %w", err)
		}
	}
	tmpFile.Close()

	// Open editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read edited content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read edited content: %w", err)
	}

	return string(content), nil
}

// createIssueInProject creates an issue in a specific project
func createIssueInProject(proj *project.Project, issuesDir string, title string, state issue.State) error {
	dir := proj.IssuesDir(issuesDir)

	// Ensure issues directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create issues directory: %w", err)
	}

	store := issue.NewStore(dir)

	// Find next issue number
	nextNumber, err := findNextIssueNumber(store)
	if err != nil {
		return fmt.Errorf("failed to determine next issue number: %w", err)
	}

	// Determine body content
	body := newBody

	// Check for stdin input (piped content)
	if body == "" && !newEditor {
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeNamedPipe) != 0 {
			scanner := bufio.NewScanner(os.Stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			body = strings.Join(lines, "\n")
		}
	}

	// Open editor if requested
	if newEditor {
		editedBody, err := openEditor(body)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
		body = editedBody
	}

	// Create issue struct
	now := time.Now()
	iss := &issue.Issue{
		Number:    nextNumber,
		Title:     title,
		State:     state,
		Labels:    newLabels,
		Assignees: newAssignees,
		CreatedAt: now,
		UpdatedAt: now,
		Body:      strings.TrimSpace(body),
	}

	// Generate filename
	slug := generateSlug(title)
	filename := fmt.Sprintf("%03d-%s.md", nextNumber, slug)
	filePath := filepath.Join(dir, filename)

	// Serialize issue
	data, err := issue.Serialize(iss)
	if err != nil {
		return fmt.Errorf("failed to serialize issue: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write issue file: %w", err)
	}

	fmt.Printf("✅ Created %s/#%d: %s\n", proj.Alias, nextNumber, filename)
	return nil
}
