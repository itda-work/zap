package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/itda-work/zap/internal/updater"
	"github.com/spf13/cobra"
)

const (
	installScriptUnix    = "https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.sh"
	installScriptWindows = "https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.ps1"
)

var (
	updateCheck   bool
	updateForce   bool
	updateYes     bool
	updateVersion string
	updateScript  bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update zap to the latest version",
	Long: `Check for and install updates to zap from GitHub releases.

Examples:
  zap update              # Check and update interactively
  zap update --check      # Check for updates only
  zap update -y           # Update without confirmation
  zap update --force      # Update without confirmation (same as -y)
  zap update -v v0.3.0    # Update to a specific version
  zap update --script     # Update using OS install script (curl/PowerShell)`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVarP(&updateCheck, "check", "c", false, "Check for updates only, do not install")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Update without confirmation")
	updateCmd.Flags().BoolVarP(&updateYes, "yes", "y", false, "Update without confirmation (alias for --force)")
	updateCmd.Flags().StringVarP(&updateVersion, "version", "v", "", "Update to a specific version")
	updateCmd.Flags().BoolVar(&updateScript, "script", false, "Update using OS-specific install script (curl/PowerShell)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Use install script if --script flag is set
	if updateScript {
		return runScriptUpdate()
	}

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
	if !updateForce && !updateYes {
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
	fmt.Println("Or use the install script:")
	fmt.Println("  zap update --script")
	return nil
}

func handlePermissionError(reason, execPath string) error {
	fmt.Printf("Error: %s\n\n", reason)
	fmt.Println("Use the install script to reinstall:")
	fmt.Println("  zap update --script")
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
		printScriptHint()
		return err
	case errors.As(err, &notFoundErr):
		fmt.Println("Release not found. Check the version number.")
		return err
	case errors.As(err, &rateLimitErr):
		fmt.Println("GitHub API rate limit exceeded. Try again later or set GITHUB_TOKEN.")
		printScriptHint()
		return err
	case errors.As(err, &checksumErr):
		fmt.Println("Checksum verification failed. Download may be corrupted.")
		printScriptHint()
		return err
	case errors.As(err, &noAssetErr):
		fmt.Printf("No binary available for your platform (%s).\n", err.Error())
		fmt.Println("You can build from source:")
		fmt.Println("  go install github.com/itda-work/zap/cmd/zap@latest")
		return err
	default:
		printScriptHint()
		return err
	}
}

func printScriptHint() {
	fmt.Println()
	fmt.Println("Or try using the install script:")
	fmt.Println("  zap update --script")
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

func runScriptUpdate() error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		script := fmt.Sprintf("iex ((New-Object System.Net.WebClient).DownloadString('%s'))", installScriptWindows)
		if updateVersion != "" {
			script = fmt.Sprintf("$env:ZAP_VERSION='%s'; ", updateVersion) + script
		}
		cmd = exec.Command("powershell", "-Command", script)
	default:
		script := fmt.Sprintf("curl -fsSL %s | bash", installScriptUnix)
		if updateVersion != "" {
			script = fmt.Sprintf("ZAP_VERSION=%s %s", updateVersion, script)
		}
		cmd = exec.Command("bash", "-c", script)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Running install script for %s...\n\n", runtime.GOOS)
	return cmd.Run()
}
