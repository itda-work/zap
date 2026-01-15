package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lim",
	Short: "Local Issue Manager - CLI/TUI tool for managing local issues",
	Long: `lim (Local Issue Manager) is a CLI/TUI tool for managing issues
stored in the .issues/ directory of your project.

When run without arguments, it starts the TUI mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 인자 없이 실행하면 TUI 모드
		return runTUI()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 글로벌 플래그 설정
	rootCmd.PersistentFlags().StringP("dir", "d", ".issues", "Issues directory path")
}

// runTUI starts the TUI mode
func runTUI() error {
	// TODO: TUI 구현 후 연결
	return tuiCmd.RunE(tuiCmd, nil)
}
