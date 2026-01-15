package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "lim",
	Short:   "Local Issue Manager - CLI tool for managing local issues",
	Version: Version,
	Long: `lim (Local Issue Manager) is a CLI tool for managing issues
stored in the .issues/ directory of your project.

Use 'lim list' to see issues or 'lim --help' for all commands.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 글로벌 플래그 설정
	rootCmd.PersistentFlags().StringP("dir", "d", ".issues", "Issues directory path")
}
