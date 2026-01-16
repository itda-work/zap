package updater

import (
	"regexp"
	"strconv"
	"strings"
)

// CompareVersions compares two semantic versions.
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	v1 = NormalizeVersion(v1)
	v2 = NormalizeVersion(v2)

	parts1 := parseVersionParts(v1)
	parts2 := parseVersionParts(v2)

	for i := 0; i < 3; i++ {
		if parts1[i] < parts2[i] {
			return -1
		}
		if parts1[i] > parts2[i] {
			return 1
		}
	}

	return 0
}

// IsDevVersion checks if the version is a development build
// that cannot be reliably compared with release versions.
func IsDevVersion(version string) bool {
	if version == "" || version == "dev" || version == "unknown" {
		return true
	}

	// Contains "dirty" suffix (local modifications)
	if strings.Contains(version, "dirty") {
		return true
	}

	// Just a commit hash (no tag)
	if isCommitHash(version) {
		return true
	}

	return false
}

// NormalizeVersion ensures the version string has a consistent format.
// Adds 'v' prefix if missing.
func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "v0.0.0"
	}

	// Remove 'v' prefix for processing, will add back
	version = strings.TrimPrefix(version, "v")

	// Remove any suffix after dash (e.g., -dirty, -rc1)
	if idx := strings.Index(version, "-"); idx != -1 {
		version = version[:idx]
	}

	return "v" + version
}

// ParseVersion extracts major, minor, patch from a version string.
func ParseVersion(version string) (major, minor, patch int, err error) {
	version = strings.TrimPrefix(NormalizeVersion(version), "v")
	parts := strings.Split(version, ".")

	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return major, minor, patch, nil
}

// parseVersionParts returns [major, minor, patch] as integers
func parseVersionParts(version string) [3]int {
	var parts [3]int
	version = strings.TrimPrefix(version, "v")

	segments := strings.Split(version, ".")
	for i := 0; i < 3 && i < len(segments); i++ {
		parts[i], _ = strconv.Atoi(segments[i])
	}

	return parts
}

// isCommitHash checks if the string looks like a git commit hash
func isCommitHash(s string) bool {
	// Short hash (7 chars) or full hash (40 chars)
	if len(s) != 7 && len(s) != 40 {
		return false
	}

	matched, _ := regexp.MatchString("^[0-9a-f]+$", s)
	return matched
}
