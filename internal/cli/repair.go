package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
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

Examples:
  zap repair            # Show files that need repair
  zap repair --auto     # Auto-repair all failed files
  zap repair 155        # Repair issue #155
  zap repair 155 159    # Repair issues #155 and #159
  zap repair --all      # Repair all failed files (with confirmation)`,
	RunE: runRepair,
}

var (
	repairAll    bool
	repairAuto   bool
	repairDryRun bool
	repairAI     string
	repairYes    bool
)

func init() {
	rootCmd.AddCommand(repairCmd)

	repairCmd.Flags().BoolVarP(&repairAll, "all", "a", false, "Repair all files with parse failures")
	repairCmd.Flags().BoolVar(&repairAuto, "auto", false, "Automatically repair all files without confirmation (same as --all --yes)")
	repairCmd.Flags().BoolVar(&repairDryRun, "dry-run", false, "Show what would be changed without modifying files")
	repairCmd.Flags().StringVar(&repairAI, "ai", "", "AI CLI to use (claude, codex, gemini)")
	repairCmd.Flags().BoolVarP(&repairYes, "yes", "y", false, "Skip confirmation prompts")
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
