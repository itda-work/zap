package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/ai"
	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var fixNumbersCmd = &cobra.Command{
	Use:   "fix-numbers",
	Short: "Detect and fix issue number conflicts",
	Long: `Detect and resolve issue number conflicts in the .issues/ directory.

This command finds and fixes:
- Duplicate filename numbers (e.g., two files starting with 001-)
- Duplicate frontmatter numbers (e.g., two files with number: 1)
- Filename-frontmatter mismatches (e.g., 001-*.md with number: 2)

The older issue (by created_at) keeps its number, newer issues are renumbered.

Examples:
  zap fix-numbers              # Detect and fix conflicts
  zap fix-numbers --dry-run    # Preview changes without modifying files
  zap fix-numbers --yes        # Skip confirmation prompts
  zap fix-numbers --no-ai      # Skip AI verification`,
	RunE: runFixNumbers,
}

var (
	fixNumbersDryRun bool
	fixNumbersYes    bool
	fixNumbersAI     string
	fixNumbersNoAI   bool
)

func init() {
	rootCmd.AddCommand(fixNumbersCmd)

	fixNumbersCmd.Flags().BoolVar(&fixNumbersDryRun, "dry-run", false, "Show what would be changed without modifying files")
	fixNumbersCmd.Flags().BoolVarP(&fixNumbersYes, "yes", "y", false, "Skip confirmation prompts")
	fixNumbersCmd.Flags().StringVar(&fixNumbersAI, "ai", "", "AI CLI to use (claude, codex, gemini)")
	fixNumbersCmd.Flags().BoolVar(&fixNumbersNoAI, "no-ai", false, "Skip AI verification")
}

func runFixNumbers(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Checking for number conflicts...")
	fmt.Println()

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

	detector := issue.NewConflictDetector(dir)
	conflicts, err := detector.DetectConflicts()
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}

	if len(conflicts) == 0 {
		fmt.Println("✅ No number conflicts found.")
		return nil
	}

	// Display conflicts
	fmt.Printf("Found %d conflict(s):\n\n", len(conflicts))
	for i, conflict := range conflicts {
		printConflict(i+1, conflict)
	}

	if fixNumbersDryRun {
		fmt.Println("\n📋 Dry run complete. No files were modified.")
		fmt.Println("Run without --dry-run to apply changes.")
		return nil
	}

	// Confirm before proceeding
	if !fixNumbersYes {
		fmt.Println()
		if !confirm("Proceed with conflict resolution?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Get AI client for verification (unless --no-ai)
	var client ai.Client
	if !fixNumbersNoAI {
		client, err = getAIClient(fixNumbersAI)
		if err != nil {
			return err
		}
		fmt.Printf("\n🤖 Using %s for verification...\n\n", client.Name())
	} else {
		fmt.Println("\n⚠️  Skipping AI verification (--no-ai)")
	}

	// Get all issue contents for AI context
	var allIssues map[string]string
	if client != nil {
		allIssues, err = detector.GetAllIssueContents()
		if err != nil {
			return fmt.Errorf("failed to load issues for context: %w", err)
		}
	}

	cfg, _ := ai.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout*time.Duration(len(conflicts)))
	defer cancel()

	successCount := 0
	for i, conflict := range conflicts {
		fmt.Printf("Processing conflict %d/%d...\n", i+1, len(conflicts))

		// AI verification (if enabled)
		if client != nil {
			safe, err := verifyConflictResolution(ctx, client, conflict, allIssues)
			if err != nil {
				fmt.Printf("  ⚠️  AI verification failed: %v\n", err)
				if !fixNumbersYes {
					if !confirm("  Continue anyway?") {
						fmt.Println("  Skipped.")
						continue
					}
				}
			} else {
				fmt.Printf("  🤖 AI: %s\n", safe)
				if strings.HasPrefix(safe, "UNSAFE:") {
					fmt.Println("  ❌ Skipping due to AI warning.")
					continue
				}
			}
		}

		// Apply the fix
		if err := applyConflictFix(conflict); err != nil {
			fmt.Printf("  ❌ Failed to fix: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ Fixed: %s\n", conflict.ToRenumber.FileName)
		successCount++
	}

	fmt.Printf("\n✅ Resolved %d/%d conflicts.\n", successCount, len(conflicts))
	return nil
}

// printConflict displays information about a single conflict.
func printConflict(num int, conflict *issue.Conflict) {
	var typeStr string
	switch conflict.Type {
	case issue.ConflictDuplicateFilename:
		typeStr = "Duplicate filename number"
	case issue.ConflictDuplicateFrontmatter:
		typeStr = "Duplicate frontmatter number"
	case issue.ConflictMismatch:
		typeStr = "Filename-frontmatter mismatch"
	}

	fmt.Printf("%d. %s: %03d\n", num, typeStr, conflict.Number)

	for _, fi := range conflict.Files {
		marker := "  "
		if fi == conflict.ToRenumber {
			marker = "→ "
		}
		createdAt := fi.GetEffectiveCreatedAt().Format("2006-01-02")
		fmt.Printf("   %s%s (created: %s)", marker, fi.FileName, createdAt)
		if fi == conflict.ToRenumber {
			if conflict.Type == issue.ConflictMismatch {
				fmt.Printf(" ← will update frontmatter to %d", conflict.NewNumber)
			} else {
				fmt.Printf(" ← will renumber to %03d", conflict.NewNumber)
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

// verifyConflictResolution uses AI to verify the resolution is safe.
func verifyConflictResolution(ctx context.Context, client ai.Client, conflict *issue.Conflict, allIssues map[string]string) (string, error) {
	tmpl, ok := ai.GetTemplate("verify-renumber")
	if !ok {
		return "", fmt.Errorf("verify-renumber template not found")
	}

	// Build all issues summary
	var issuesSummary strings.Builder
	for filename, content := range allIssues {
		// Only include first 500 chars of each issue to stay within limits
		preview := content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		issuesSummary.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", filename, preview))
	}

	// Read the file content
	fileContent := ""
	if conflict.ToRenumber != nil {
		if data, err := os.ReadFile(conflict.ToRenumber.FilePath); err == nil {
			fileContent = string(data)
		}
	}

	var conflictType, reason string
	switch conflict.Type {
	case issue.ConflictDuplicateFilename:
		conflictType = "duplicate_filename"
		reason = "Multiple files share the same filename number prefix"
	case issue.ConflictDuplicateFrontmatter:
		conflictType = "duplicate_frontmatter"
		reason = "Multiple files have the same number in frontmatter"
	case issue.ConflictMismatch:
		conflictType = "mismatch"
		reason = "Filename number differs from frontmatter number"
	}

	req, err := tmpl.Render(map[string]string{
		"conflict_type":  conflictType,
		"filename":       conflict.ToRenumber.FileName,
		"current_number": strconv.Itoa(conflict.Number),
		"new_number":     strconv.Itoa(conflict.NewNumber),
		"reason":         reason,
		"file_content":   fileContent,
		"all_issues":     issuesSummary.String(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to render prompt: %w", err)
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}

// applyConflictFix applies the fix for a single conflict.
func applyConflictFix(conflict *issue.Conflict) error {
	fi := conflict.ToRenumber
	if fi == nil {
		return fmt.Errorf("no file to renumber")
	}

	// Create backup
	backupPath := fi.FilePath + ".backup"
	originalContent, err := os.ReadFile(fi.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	if err := os.WriteFile(backupPath, originalContent, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	switch conflict.Type {
	case issue.ConflictMismatch:
		// Update frontmatter number to match filename
		return updateFrontmatterNumber(fi, conflict.NewNumber)

	case issue.ConflictDuplicateFilename, issue.ConflictDuplicateFrontmatter:
		// Rename file and update frontmatter
		return renumberIssue(fi, conflict.NewNumber)
	}

	return nil
}

// updateFrontmatterNumber updates the number in frontmatter.
func updateFrontmatterNumber(fi *issue.FileInfo, newNumber int) error {
	if fi.Issue == nil {
		return fmt.Errorf("cannot update unparseable file")
	}

	fi.Issue.Number = newNumber
	fi.Issue.UpdatedAt = time.Now()

	data, err := issue.Serialize(fi.Issue)
	if err != nil {
		return fmt.Errorf("failed to serialize: %w", err)
	}

	return os.WriteFile(fi.FilePath, data, 0644)
}

// renumberIssue renames the file and updates frontmatter.
func renumberIssue(fi *issue.FileInfo, newNumber int) error {
	// Extract slug from current filename (e.g., "001-feature-name.md" -> "feature-name")
	slug := extractSlugFromFilename(fi.FileName)
	if slug == "" {
		slug = "issue"
	}

	// Build new filename
	newFilename := fmt.Sprintf("%03d-%s.md", newNumber, slug)
	newPath := filepath.Join(filepath.Dir(fi.FilePath), newFilename)

	// Check if new path already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target file already exists: %s", newFilename)
	}

	// Update frontmatter if issue is parseable
	if fi.Issue != nil {
		fi.Issue.Number = newNumber
		fi.Issue.UpdatedAt = time.Now()
		fi.Issue.FilePath = newPath

		data, err := issue.Serialize(fi.Issue)
		if err != nil {
			return fmt.Errorf("failed to serialize: %w", err)
		}

		if err := os.WriteFile(fi.FilePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write updated content: %w", err)
		}
	}

	// Rename the file
	if err := os.Rename(fi.FilePath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// extractSlugFromFilename extracts the slug part from a filename.
// e.g., "001-feature-name.md" -> "feature-name"
func extractSlugFromFilename(filename string) string {
	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Find first dash after number
	idx := strings.Index(name, "-")
	if idx == -1 {
		return ""
	}

	return name[idx+1:]
}
