package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time
// Example: go build -ldflags "-X github.com/itda-work/zap/internal/cli.Version=v1.0.0"
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of zap.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("zap version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
