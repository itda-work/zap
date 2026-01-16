package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/itda-work/zap/internal/updater"
	"github.com/spf13/cobra"
)

var (
	updateCheck   bool
	updateForce   bool
	updateVersion string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update zap to the latest version",
	Long: `Check for and install updates to zap from GitHub releases.

Examples:
  zap update              # Check and update interactively
  zap update --check      # Check for updates only
  zap update --force      # Update without confirmation
  zap update -v v0.3.0    # Update to a specific version`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVarP(&updateCheck, "check", "c", false, "Check for updates only, do not install")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Update without confirmation")
	updateCmd.Flags().StringVarP(&updateVersion, "version", "v", "", "Update to a specific version")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Check if running a dev build
	if updater.IsDevVersion(Version) {
		return handleDevBuild()
	}

	// Create updater
	u, err := updater.NewUpdater(Version)
	if err != nil {
		return fmt.Errorf("initialize updater: %w", err)
	}

	// Check for updates
	var info *updater.UpdateInfo
	if updateVersion != "" {
		info, err = u.CheckForUpdateToVersion(updateVersion)
	} else {
		info, err = u.CheckForUpdate()
	}

	if err != nil {
		return handleUpdateError(err)
	}

	// Display version info
	fmt.Printf("Current version: %s\n", info.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", info.LatestVersion)
	fmt.Println()

	// No update available
	if !info.UpdateAvailable {
		fmt.Println("You're already running the latest version.")
		return nil
	}

	// Check only mode
	if updateCheck {
		fmt.Println("Update available! Run 'zap update' to install.")
		return nil
	}

	// Check if we can self-update
	canUpdate, reason := u.CanSelfUpdate()
	if !canUpdate {
		return handlePermissionError(reason, u.ExecPath())
	}

	// Confirm update
	if !updateForce {
		if !confirmUpdate(info.CurrentVersion, info.LatestVersion) {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	// Perform update
	fmt.Println()
	var lastStage string
	err = u.Update(info.ReleaseInfo, func(stage string, pct int) {
		if stage != lastStage {
			// New stage: print stage name
			if lastStage != "" {
				fmt.Println("OK")
			}
			fmt.Printf("%s... ", stage)
			lastStage = stage
		}
		// Only show percentage for download progress (optional: could add spinner)
	})
	if err == nil && lastStage != "" {
		fmt.Println("OK")
	}

	if err != nil {
		if lastStage != "" {
			fmt.Println("FAILED")
		}
		return handleUpdateError(err)
	}

	fmt.Println()
	fmt.Printf("Successfully updated to %s!\n", info.LatestVersion)
	return nil
}

func handleDevBuild() error {
	fmt.Printf("Current version: %s\n\n", Version)
	fmt.Println("You're running a development build. Cannot determine update status.")
	fmt.Println()
	fmt.Println("If you installed via 'go install', use:")
	fmt.Println("  go install github.com/itda-work/zap/cmd/zap@latest")
	fmt.Println()
	fmt.Println("Otherwise, download the latest release:")
	fmt.Println("  https://github.com/itda-work/zap/releases/latest")
	return nil
}

func handlePermissionError(reason, execPath string) error {
	fmt.Printf("Error: %s\n\n", reason)
	fmt.Println("The binary is installed in a system directory. Try:")
	fmt.Println("  sudo zap update")
	fmt.Println()
	fmt.Println("Or reinstall to a user directory:")
	fmt.Println("  ZAP_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.sh | bash")
	return fmt.Errorf("permission denied")
}

func handleUpdateError(err error) error {
	var netErr *updater.NetworkError
	var notFoundErr *updater.NotFoundError
	var rateLimitErr *updater.RateLimitError
	var checksumErr *updater.ChecksumError
	var noAssetErr *updater.NoAssetError

	switch {
	case errors.As(err, &netErr):
		fmt.Println("Failed to connect to GitHub. Check your internet connection.")
		return err
	case errors.As(err, &notFoundErr):
		fmt.Println("Release not found. Check the version number.")
		return err
	case errors.As(err, &rateLimitErr):
		fmt.Printf("GitHub API rate limit exceeded. Try again later or set GITHUB_TOKEN.\n")
		return err
	case errors.As(err, &checksumErr):
		fmt.Println("Checksum verification failed. Download may be corrupted.")
		return err
	case errors.As(err, &noAssetErr):
		fmt.Printf("No binary available for your platform (%s).\n", err.Error())
		fmt.Println("You can build from source:")
		fmt.Println("  go install github.com/itda-work/zap/cmd/zap@latest")
		return err
	default:
		return err
	}
}

func confirmUpdate(current, latest string) bool {
	fmt.Printf("Update zap %s → %s? [Y/n]: ", current, latest)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}
