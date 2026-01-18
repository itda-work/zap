package cli

import (
	"bufio"
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

var repairCmd = &cobra.Command{
	Use:     "repair [number...]",
	Aliases: []string{"r"},
	Short:   "Repair issue files using AI",
	Long:    `Repair malformed issue files using AI CLI tools (claude, codex, gemini).

Without arguments, shows files that need repair.
With --auto flag, automatically repairs all failed files without confirmation.
With --all flag, repairs all failed files (with confirmation).
With number arguments, repairs specific files sequentially.
With --conflicts flag, detects and fixes issue number conflicts.

Examples:
  zap repair              # Show files that need repair
  zap repair --auto       # Auto-repair all failed files
  zap repair 155          # Repair issue #155
  zap repair 155 159      # Repair issues #155 and #159
  zap repair --all        # Repair all failed files (with confirmation)
  zap repair --conflicts  # Detect and fix number conflicts
  zap repair --conflicts --dry-run  # Preview conflict fixes`,
	RunE: runRepair,
}

var (
	repairAll       bool
	repairAuto      bool
	repairDryRun    bool
	repairAI        string
	repairYes       bool
	repairConflicts bool
)

func init() {
	rootCmd.AddCommand(repairCmd)

	repairCmd.Flags().BoolVarP(&repairAll, "all", "a", false, "Repair all files with parse failures")
	repairCmd.Flags().BoolVar(&repairAuto, "auto", false, "Automatically repair all files without confirmation (same as --all --yes)")
	repairCmd.Flags().BoolVar(&repairDryRun, "dry-run", false, "Show what would be changed without modifying files")
	repairCmd.Flags().StringVar(&repairAI, "ai", "", "AI CLI to use (claude, codex, gemini)")
	repairCmd.Flags().BoolVarP(&repairYes, "yes", "y", false, "Skip confirmation prompts")
	repairCmd.Flags().BoolVar(&repairConflicts, "conflicts", false, "Detect and fix issue number conflicts")
}

func runRepair(cmd *cobra.Command, args []string) error {
	// --auto implies --all --yes
	if repairAuto {
		repairAll = true
		repairYes = true
	}

	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	// Handle --conflicts mode
	if repairConflicts {
		return runRepairConflicts(cmd, dir)
	}

	store := issue.NewStore(dir)

	// Load issues to populate warnings
	store.List(issue.AllStates()...)
	warnings := store.WarningsWithContent()

	if len(warnings) == 0 {
		fmt.Println("No files need repair.")
		return nil
	}

	// Determine what to repair
	var toRepair []issue.ParseFailure

	if len(args) > 0 {
		// Repair specific issues by number
		for _, arg := range args {
			number, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", arg)
			}

			failure := store.GetFailureByNumber(number)
			if failure == nil {
				fmt.Printf("⚠️  No parse failure found for issue #%d, skipping\n", number)
				continue
			}
			toRepair = append(toRepair, *failure)
		}
		if len(toRepair) == 0 {
			return fmt.Errorf("no valid parse failures found for the specified issues")
		}
	} else if repairAll {
		toRepair = warnings
	} else {
		// Show what needs repair
		fmt.Printf("Files needing repair (%d):\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("  - %s: %s\n", w.FileName, w.Error)
		}
		fmt.Println("\nUse 'zap repair --all' to repair all, or 'zap repair <number>' for a specific file.")
		return nil
	}

	// Get AI client
	client, err := getAIClient()
	if err != nil {
		return err
	}

	fmt.Printf("🤖 Using %s to repair %d file(s)...\n\n", client.Name(), len(toRepair))

	// Get the repair template
	tmpl, ok := ai.GetTemplate("repair-frontmatter")
	if !ok {
		return fmt.Errorf("repair-frontmatter template not found")
	}

	cfg, _ := ai.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout*time.Duration(len(toRepair)))
	defer cancel()

	successCount := 0
	for _, failure := range toRepair {
		fmt.Printf("Processing %s...\n", failure.FileName)

		// Render prompt
		req, err := tmpl.Render(map[string]string{
			"filename": failure.FileName,
			"content":  failure.Content,
		})
		if err != nil {
			fmt.Printf("  ❌ Failed to render prompt: %v\n", err)
			continue
		}

		// Call AI
		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Printf("  ❌ AI error: %v\n", err)
			continue
		}

		newContent := cleanAIResponse(resp.Content)

		// Validate the response looks like a valid issue file
		if !strings.HasPrefix(strings.TrimSpace(newContent), "---") {
			fmt.Printf("  ❌ AI response doesn't look like valid frontmatter\n")
			continue
		}

		if repairDryRun {
			// Show diff
			fmt.Printf("  📝 Proposed changes:\n")
			printDiff(failure.Content, newContent)
			fmt.Println()
		} else {
			// Confirm unless --yes
			if !repairYes {
				fmt.Printf("  📝 Changes:\n")
				printDiff(failure.Content, newContent)
				if !confirm("  Apply these changes?") {
					fmt.Printf("  ⏭️  Skipped\n")
					continue
				}
			}

			// Backup original
			backupPath := failure.FilePath + ".backup"
			if err := os.WriteFile(backupPath, []byte(failure.Content), 0644); err != nil {
				fmt.Printf("  ❌ Failed to create backup: %v\n", err)
				continue
			}

			// Write new content
			if err := os.WriteFile(failure.FilePath, []byte(newContent), 0644); err != nil {
				fmt.Printf("  ❌ Failed to write file: %v\n", err)
				// Restore from backup
				os.WriteFile(failure.FilePath, []byte(failure.Content), 0644)
				continue
			}

			fmt.Printf("  ✅ Repaired (backup: %s)\n", backupPath)
			successCount++
		}
	}

	if repairDryRun {
		fmt.Printf("\nDry run complete. No files were modified.\n")
	} else {
		fmt.Printf("\nRepaired %d/%d files.\n", successCount, len(toRepair))
	}

	return nil
}

// getAIClient returns an AI client based on flags or auto-detection.
func getAIClient() (ai.Client, error) {
	cfg, err := ai.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load AI config: %w", err)
	}

	if repairAI != "" {
		provider, ok := ai.ParseProvider(repairAI)
		if !ok {
			return nil, fmt.Errorf("unknown AI provider: %s (supported: claude, codex, gemini)", repairAI)
		}
		client := ai.NewClient(provider, cfg)
		if client == nil || !client.IsAvailable() {
			return nil, fmt.Errorf("%s CLI is not installed or not available", repairAI)
		}
		return client, nil
	}

	// Auto-detect
	client, err := ai.AutoDetect(cfg)
	if err != nil {
		return nil, fmt.Errorf("no AI CLI available. Install one of: claude, codex, gemini")
	}
	return client, nil
}

// cleanAIResponse removes markdown code blocks if present.
func cleanAIResponse(content string) string {
	content = strings.TrimSpace(content)

	// Remove ```markdown or ```yaml wrapper if present
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			// Remove first and last lines if they're code block markers
			if strings.HasPrefix(lines[0], "```") && strings.HasPrefix(lines[len(lines)-1], "```") {
				content = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}
	}

	return strings.TrimSpace(content)
}

// printDiff shows a simple diff between old and new content.
func printDiff(old, new string) {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	// Simple line-by-line comparison (not a real diff algorithm)
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	// Show first 20 lines max
	if maxLines > 20 {
		maxLines = 20
	}

	for i := 0; i < maxLines; i++ {
		oldLine := ""
		newLine := ""
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if oldLine != "" {
				fmt.Printf("     %s\n", colorize("- "+oldLine, colorRed))
			}
			if newLine != "" {
				fmt.Printf("     %s\n", colorize("+ "+newLine, colorGreen))
			}
		}
	}

	if len(oldLines) > 20 || len(newLines) > 20 {
		fmt.Printf("     ... (%d more lines)\n", maxLines-20)
	}
}

// confirm prompts the user for yes/no confirmation.
func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// runRepairConflicts handles the --conflicts mode.
func runRepairConflicts(cmd *cobra.Command, dir string) error {
	fmt.Println("🔍 Checking for number conflicts...")
	fmt.Println()

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

	if repairDryRun {
		fmt.Println("\n📋 Dry run complete. No files were modified.")
		fmt.Println("Run without --dry-run to apply changes.")
		return nil
	}

	// Confirm before proceeding
	if !repairYes {
		fmt.Println()
		if !confirm("Proceed with conflict resolution?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Get AI client for verification
	client, err := getAIClient()
	if err != nil {
		return err
	}

	// Get all issue contents for AI context
	allIssues, err := detector.GetAllIssueContents()
	if err != nil {
		return fmt.Errorf("failed to load issues for context: %w", err)
	}

	cfg, _ := ai.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout*time.Duration(len(conflicts)))
	defer cancel()

	fmt.Printf("\n🤖 Using %s for verification...\n\n", client.Name())

	successCount := 0
	for i, conflict := range conflicts {
		fmt.Printf("Processing conflict %d/%d...\n", i+1, len(conflicts))

		// AI verification
		safe, err := verifyConflictResolution(ctx, client, conflict, allIssues)
		if err != nil {
			fmt.Printf("  ⚠️  AI verification failed: %v\n", err)
			if !repairYes {
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
