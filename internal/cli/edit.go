package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:               "edit <number>",
	Aliases:           []string{"e"},
	Short:             "Edit an issue in your editor",
	Long:              `Open an issue file in your editor for editing.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueNumber,
	RunE:              runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	number, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	iss, err := store.Get(number)
	if err != nil {
		return err
	}

	editor := getEditor()
	return openInEditor(editor, iss.FilePath)
}

// getEditor returns the editor command following Git's priority:
// GIT_EDITOR -> VISUAL -> EDITOR -> default (vi on Unix, notepad on Windows)
func getEditor() string {
	if editor := os.Getenv("GIT_EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Default editor based on OS
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// openInEditor opens the file in the specified editor and waits for it to close
func openInEditor(editor, filePath string) error {
	// Parse editor command (may contain arguments like "code --wait")
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("empty editor command")
	}

	cmdName := parts[0]
	cmdArgs := append(parts[1:], filePath)

	editorCmd := exec.Command(cmdName, cmdArgs...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		// Check if it's an exit error (non-zero exit code)
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "Warning: editor exited with code %d\n", exitErr.ExitCode())
			return nil
		}
		return fmt.Errorf("failed to run editor: %w", err)
	}

	return nil
}
