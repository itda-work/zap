package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version and BuildDate are set via ldflags at build time
// Example: go build -ldflags "-X github.com/itda-work/zap/internal/cli.Version=v1.0.0 -X github.com/itda-work/zap/internal/cli.BuildDate=2026-01-15"
var (
	Version   = "dev"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of zap.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("zap version %s (built %s)\n", Version, BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
