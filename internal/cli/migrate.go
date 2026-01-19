package cli

import (
	"fmt"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate from directory-based to flat structure",
	Long: `Migrate issue files from the old directory-based structure
(.issues/{state}/*.md) to the new flat structure (.issues/*.md).

This command will:
1. Update frontmatter state to match source directory
2. Move files using git mv (falls back to mv if not git-tracked)
3. Remove empty state directories

After migration, state is determined solely from frontmatter.`,
	RunE: runMigrate,
}

var (
	migrateDryRun bool
	migrateYes    bool
)

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would be migrated without making changes")
	migrateCmd.Flags().BoolVarP(&migrateYes, "yes", "y", false, "Skip confirmation prompt")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	// Detect legacy structure
	info, err := store.DetectLegacyStructure()
	if err != nil {
		return err
	}

	if !info.HasLegacyStructure {
		fmt.Println("No legacy structure detected. Already using flat structure.")
		return nil
	}

	// Show what will be migrated
	fmt.Printf("Found %d issues in legacy directory structure:\n\n", info.TotalIssues)
	for _, state := range issue.AllStates() {
		files := info.IssuesByState[state]
		if len(files) > 0 {
			fmt.Printf("  %s/ (%d files)\n", state, len(files))
			for _, f := range files {
				fmt.Printf("    - %s\n", f)
			}
		}
	}

	if migrateDryRun {
		fmt.Println("\nDry run complete. No changes made.")
		return nil
	}

	// Confirm unless --yes
	if !migrateYes {
		fmt.Println()
		if !confirm("Migrate to flat structure?") {
			fmt.Println("Migration cancelled.")
			return nil
		}
	}

	// Execute migration
	result, err := store.Migrate()
	if err != nil {
		return err
	}

	fmt.Printf("\nMigration complete:\n")
	fmt.Printf("  Migrated: %d\n", result.Migrated)
	if result.Failed > 0 {
		fmt.Printf("  Failed:   %d\n", result.Failed)
		for i, f := range result.FailedFiles {
			fmt.Printf("    - %s: %s\n", f, result.Errors[i])
		}
	}

	return nil
}
