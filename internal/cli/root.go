package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
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
// This is used for single-project mode (backward compatibility)
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

// getProjectSpecs parses -C flags into ProjectSpec list
// Returns nil if no -C flags are provided
func getProjectSpecs(cmd *cobra.Command) []project.ProjectSpec {
	projectDirs, _ := cmd.Flags().GetStringArray("project")
	if len(projectDirs) == 0 {
		return nil
	}

	specs := make([]project.ProjectSpec, 0, len(projectDirs))
	for _, dir := range projectDirs {
		spec := project.ParseProjectSpec(dir)
		// Expand tilde in the path part
		spec.Path = expandTilde(spec.Path)
		specs = append(specs, spec)
	}
	return specs
}

// isMultiProjectMode returns true if multiple -C flags are provided
func isMultiProjectMode(cmd *cobra.Command) bool {
	projectDirs, _ := cmd.Flags().GetStringArray("project")
	return len(projectDirs) > 1
}

// getMultiStore creates a MultiStore from -C flags
// Returns nil if in single-project mode
func getMultiStore(cmd *cobra.Command) (*project.MultiStore, error) {
	specs := getProjectSpecs(cmd)
	if len(specs) <= 1 {
		return nil, nil // Single project mode
	}

	issuesDir, _ := cmd.Flags().GetString("dir")
	return project.NewMultiStore(specs, issuesDir)
}

// getStore returns an issue.Store for single-project mode
// This is the existing behavior for backward compatibility
func getStore(cmd *cobra.Command) (*issue.Store, error) {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return nil, err
	}
	return issue.NewStore(dir), nil
}
