package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "zap",
	Short:   "Local Issue Manager - CLI tool for managing local issues",
	Version: Version,
	Long: `zap (Local Issue Manager) is a CLI tool for managing issues
stored in the .issues/ directory of your project.

Use 'zap list' to see issues or 'zap --help' for all commands.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 글로벌 플래그 설정
	rootCmd.PersistentFlags().StringP("dir", "d", ".issues", "Issues directory path")
	rootCmd.PersistentFlags().StringArrayP("project", "C", nil, "Run as if zap was started in <path> (can be used multiple times)")
}

// expandTilde expands ~ to home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// getProjectDir returns the combined project directory from -C flags
func getProjectDir(cmd *cobra.Command) (string, error) {
	projectDirs, _ := cmd.Flags().GetStringArray("project")
	if len(projectDirs) == 0 {
		return "", nil
	}

	// Combine all -C paths (like git does)
	basePath := ""
	for _, dir := range projectDirs {
		expanded := expandTilde(dir)
		if filepath.IsAbs(expanded) {
			basePath = expanded
		} else {
			basePath = filepath.Join(basePath, expanded)
		}
	}

	// Validate path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return "", fmt.Errorf("project directory does not exist: %s", basePath)
	}

	return basePath, nil
}

// getIssuesDir returns the issues directory path, combining -C and -d flags
func getIssuesDir(cmd *cobra.Command) (string, error) {
	projectDir, err := getProjectDir(cmd)
	if err != nil {
		return "", err
	}

	issuesDir, _ := cmd.Flags().GetString("dir")

	if projectDir != "" {
		return filepath.Join(projectDir, issuesDir), nil
	}
	return issuesDir, nil
}
