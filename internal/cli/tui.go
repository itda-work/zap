package cli

import (
	"github.com/allieus/lim/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start TUI mode",
	Long:  `Start the Terminal User Interface for browsing issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		return tui.Run(dir)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
