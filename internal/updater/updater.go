package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// UpdateInfo contains the result of an update check.
type UpdateInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseInfo     *ReleaseInfo
}

// Updater handles the self-update process.
type Updater struct {
	currentVersion string
	github         *GitHubClient
	execPath       string
}

// NewUpdater creates a new Updater instance.
func NewUpdater(currentVersion string) (*Updater, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot determine executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve symlinks: %w", err)
	}

	return &Updater{
		currentVersion: currentVersion,
		github:         NewGitHubClient(),
		execPath:       execPath,
	}, nil
}

// CheckForUpdate checks if a newer version is available.
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
	release, err := u.github.GetLatestRelease()
	if err != nil {
		return nil, err
	}

	latestVersion := release.TagName
	updateAvailable := CompareVersions(u.currentVersion, latestVersion) < 0

	return &UpdateInfo{
		CurrentVersion:  u.currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
		ReleaseInfo:     release,
	}, nil
}

// CheckForUpdateToVersion checks if a specific version is available.
func (u *Updater) CheckForUpdateToVersion(targetVersion string) (*UpdateInfo, error) {
	targetVersion = NormalizeVersion(targetVersion)

	release, err := u.github.GetRelease(targetVersion)
	if err != nil {
		return nil, err
	}

	return &UpdateInfo{
		CurrentVersion:  u.currentVersion,
		LatestVersion:   release.TagName,
		UpdateAvailable: CompareVersions(u.currentVersion, release.TagName) != 0,
		ReleaseInfo:     release,
	}, nil
}

// Update downloads and installs a new version.
func (u *Updater) Update(release *ReleaseInfo, progress func(stage string, pct int)) error {
	// Find the correct asset for this platform
	asset, err := u.GetAssetForPlatform(release)
	if err != nil {
		return err
	}

	// Get checksums
	if progress != nil {
		progress("Downloading checksums", 0)
	}
	checksums, err := u.github.GetChecksums(release)
	if err != nil {
		return fmt.Errorf("get checksums: %w", err)
	}

	// Create temp file for download
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, asset.Name)
	defer os.Remove(tempFile)

	// Download the binary
	if progress != nil {
		progress("Downloading "+asset.Name, 0)
	}
	downloadProgress := func(downloaded, total int64) {
		if progress != nil && total > 0 {
			pct := int(float64(downloaded) / float64(total) * 100)
			progress("Downloading "+asset.Name, pct)
		}
	}

	if err := u.github.DownloadAsset(asset, tempFile, downloadProgress); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Verify checksum
	if progress != nil {
		progress("Verifying checksum", 0)
	}
	if err := u.VerifyChecksum(tempFile, asset.Name, checksums); err != nil {
		return err
	}

	// Make executable (Unix only)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempFile, 0755); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
	}

	// Perform atomic replacement
	if progress != nil {
		progress("Installing", 0)
	}
	if err := u.AtomicReplace(tempFile); err != nil {
		return err
	}

	if progress != nil {
		progress("Done", 100)
	}

	return nil
}

// GetAssetForPlatform returns the correct asset for the current OS/arch.
func (u *Updater) GetAssetForPlatform(release *ReleaseInfo) (*Asset, error) {
	osName, archName := getPlatformInfo()
	expectedName := getAssetName(osName, archName)

	for i := range release.Assets {
		if release.Assets[i].Name == expectedName {
			return &release.Assets[i], nil
		}
	}

	return nil, &NoAssetError{OS: osName, Arch: archName}
}

// VerifyChecksum validates the downloaded file against checksums.
func (u *Updater) VerifyChecksum(filePath, assetName string, checksums map[string]string) error {
	expected, ok := checksums[assetName]
	if !ok {
		return fmt.Errorf("no checksum found for %s", assetName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("compute hash: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return &ChecksumError{Expected: expected, Actual: actual}
	}

	return nil
}

// AtomicReplace safely replaces the current binary with a new one.
func (u *Updater) AtomicReplace(newBinaryPath string) error {
	// Create backup path
	backupPath := u.execPath + ".old"

	// Remove any existing backup
	_ = os.Remove(backupPath)

	// Rename current binary to backup
	if err := os.Rename(u.execPath, backupPath); err != nil {
		return &PermissionError{
			Path:    u.execPath,
			Message: fmt.Sprintf("cannot rename current binary: %v", err),
		}
	}

	// Move new binary to current location
	if err := copyFile(newBinaryPath, u.execPath); err != nil {
		// Try to restore from backup
		_ = os.Rename(backupPath, u.execPath)
		return fmt.Errorf("install new binary: %w", err)
	}

	// Make the new binary executable (Unix only)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(u.execPath, 0755); err != nil {
			// Try to restore from backup
			_ = os.Remove(u.execPath)
			_ = os.Rename(backupPath, u.execPath)
			return fmt.Errorf("chmod new binary: %w", err)
		}
	}

	// Remove backup on success
	_ = os.Remove(backupPath)

	return nil
}

// CanSelfUpdate checks if the binary can be updated in place.
func (u *Updater) CanSelfUpdate() (bool, string) {
	dir := filepath.Dir(u.execPath)

	// Try to create a test file
	testFile := filepath.Join(dir, ".zap-update-test")
	f, err := os.Create(testFile)
	if err != nil {
		if os.IsPermission(err) {
			return false, fmt.Sprintf("no write permission to %s", dir)
		}
		return false, err.Error()
	}
	f.Close()
	os.Remove(testFile)

	return true, ""
}

// ExecPath returns the path to the current executable.
func (u *Updater) ExecPath() string {
	return u.execPath
}

// Helper functions

func getPlatformInfo() (osName, archName string) {
	switch runtime.GOOS {
	case "darwin":
		osName = "macos"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "windows"
	default:
		osName = runtime.GOOS
	}

	switch runtime.GOARCH {
	case "amd64":
		archName = "amd64"
	case "arm64":
		archName = "arm64"
	default:
		archName = runtime.GOARCH
	}

	return osName, archName
}

func getAssetName(osName, archName string) string {
	name := fmt.Sprintf("zap-%s-%s", osName, archName)
	if osName == "windows" {
		name += ".exe"
	}
	return name
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// Error types

// NoAssetError indicates no binary is available for the platform.
type NoAssetError struct {
	OS   string
	Arch string
}

func (e *NoAssetError) Error() string {
	return fmt.Sprintf("no binary available for %s/%s", e.OS, e.Arch)
}

// ChecksumError indicates checksum verification failed.
type ChecksumError struct {
	Expected string
	Actual   string
}

func (e *ChecksumError) Error() string {
	return "checksum verification failed"
}

// PermissionError indicates a permission problem.
type PermissionError struct {
	Path    string
	Message string
}

func (e *PermissionError) Error() string {
	return e.Message
}
