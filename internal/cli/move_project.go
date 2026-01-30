package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var moveProjectCmd = &cobra.Command{
	Use:   "move <number> --to <project-path>",
	Short: "Move an issue to another project",
	Long: `Move an issue to another project's .issues/ directory.

The issue receives a new number in the destination project (like 'zap new').
Original issue is preserved by default; use --delete to remove it.

Examples:
  zap move 5 --to ~/other-project
  zap move 5 --to ~/other-project --delete
  zap move 5 --to ~/other-project --dst-dir .tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runMoveProject,
}

var (
	moveToPath string
	moveDelete bool
	moveDstDir string
)

func init() {
	rootCmd.AddCommand(moveProjectCmd)
	moveProjectCmd.Flags().StringVar(&moveToPath, "to", "", "Destination project path (required)")
	moveProjectCmd.Flags().BoolVar(&moveDelete, "delete", false, "Delete original issue after moving")
	moveProjectCmd.Flags().StringVar(&moveDstDir, "dst-dir", ".issues", "Issues directory name in destination project")
	_ = moveProjectCmd.MarkFlagRequired("to")
}

func runMoveProject(cmd *cobra.Command, args []string) error {
	number, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}

	srcDir, wasDiscovered, err := getIssuesDirWithDiscovery(cmd)
	if err != nil {
		return err
	}

	if wasDiscovered {
		fmt.Fprintf(os.Stderr, "info: Using .issues at %s\n", srcDir)
		if !IsTTY() {
			return fmt.Errorf("cannot modify issues in parent directory from non-interactive session (use --project or -d flag to specify directory explicitly)")
		}
		if !confirmYesDefault("Proceed with this .issues directory?") {
			return fmt.Errorf("operation cancelled")
		}
	}

	dstProjectPath := expandTilde(moveToPath)
	if !filepath.IsAbs(dstProjectPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		dstProjectPath = filepath.Join(cwd, dstProjectPath)
	}

	if stat, err := os.Stat(dstProjectPath); err != nil || !stat.IsDir() {
		return fmt.Errorf("destination project directory does not exist: %s", dstProjectPath)
	}

	dstDir := filepath.Join(dstProjectPath, moveDstDir)

	absSrcDir, _ := filepath.Abs(srcDir)
	absDstDir, _ := filepath.Abs(dstDir)
	if absSrcDir == absDstDir {
		return fmt.Errorf("source and destination are the same directory: %s", absSrcDir)
	}

	srcStore := issue.NewStore(srcDir)
	srcIssue, err := srcStore.Get(number)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination issues directory: %w", err)
	}

	dstStore := issue.NewStore(dstDir)
	nextNumber, err := findNextIssueNumber(dstStore)
	if err != nil {
		return fmt.Errorf("failed to determine next issue number in destination: %w", err)
	}

	now := time.Now().UTC()
	srcProjectName := filepath.Base(filepath.Dir(absSrcDir))
	provenanceNote := fmt.Sprintf("> Moved from %s #%d", srcProjectName, srcIssue.Number)

	body := srcIssue.Body
	if body != "" {
		body = provenanceNote + "\n\n" + body
	} else {
		body = provenanceNote
	}

	dstIssue := &issue.Issue{
		Number:    nextNumber,
		Title:     srcIssue.Title,
		State:     srcIssue.State,
		Labels:    srcIssue.Labels,
		Assignees: srcIssue.Assignees,
		CreatedAt: srcIssue.CreatedAt,
		UpdatedAt: now,
		ClosedAt:  srcIssue.ClosedAt,
		Body:      body,
	}

	slug := generateSlug(srcIssue.Title)
	filename := fmt.Sprintf("%03d-%s.md", nextNumber, slug)
	dstFilePath := filepath.Join(dstDir, filename)

	data, err := issue.Serialize(dstIssue)
	if err != nil {
		return fmt.Errorf("failed to serialize issue: %w", err)
	}

	if err := os.WriteFile(dstFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write issue file: %w", err)
	}

	dstName := filepath.Base(dstProjectPath)
	if moveDelete {
		if err := os.Remove(srcIssue.FilePath); err != nil {
			return fmt.Errorf("failed to delete original issue: %w", err)
		}
		fmt.Printf("Moved #%d → %s #%d (%s) [original deleted]\n",
			srcIssue.Number, dstName, nextNumber, filename)
	} else {
		fmt.Printf("Moved #%d → %s #%d (%s)\n",
			srcIssue.Number, dstName, nextNumber, filename)
	}

	return nil
}

func parseIssueNumber(s string) (int, error) {
	s = strings.TrimPrefix(s, "#")
	var number int
	_, err := fmt.Sscanf(s, "%d", &number)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("invalid issue number: %s", s)
	}
	return number, nil
}
