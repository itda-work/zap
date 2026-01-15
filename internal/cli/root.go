package cli

import (
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
}
