package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		// Basic comparisons
		{"equal versions", "v0.2.0", "v0.2.0", 0},
		{"v1 less than v2", "v0.1.0", "v0.2.0", -1},
		{"v1 greater than v2", "v0.3.0", "v0.2.0", 1},

		// Major version differences
		{"major difference less", "v0.9.9", "v1.0.0", -1},
		{"major difference greater", "v2.0.0", "v1.9.9", 1},

		// Minor version differences
		{"minor difference less", "v0.1.9", "v0.2.0", -1},
		{"minor difference greater", "v0.10.0", "v0.9.0", 1},

		// Patch version differences
		{"patch difference less", "v0.2.0", "v0.2.1", -1},
		{"patch difference greater", "v0.2.2", "v0.2.1", 1},

		// Without v prefix
		{"without v prefix", "0.2.0", "v0.2.0", 0},
		{"both without v prefix", "0.1.0", "0.2.0", -1},

		// With suffixes (should be stripped)
		{"with dirty suffix", "v0.2.0-dirty", "v0.2.0", 0},
		{"with rc suffix", "v0.2.0-rc1", "v0.2.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestIsDevVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"dev string", "dev", true},
		{"empty string", "", true},
		{"unknown string", "unknown", true},
		{"dirty suffix", "v0.2.0-dirty", true},
		{"short commit hash", "abc1234", true},
		{"long commit hash", "abc1234567890abc1234567890abc1234567890a", true},
		{"normal version", "v0.2.0", false},
		{"version without v", "0.2.0", false},
		{"version with rc", "v0.2.0-rc1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDevVersion(tt.version)
			if got != tt.want {
				t.Errorf("IsDevVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"with v prefix", "v0.2.0", "v0.2.0"},
		{"without v prefix", "0.2.0", "v0.2.0"},
		{"with dirty suffix", "v0.2.0-dirty", "v0.2.0"},
		{"empty string", "", "v0.0.0"},
		{"with whitespace", "  v0.2.0  ", "v0.2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeVersion(tt.version)
			if got != tt.want {
				t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		major int
		minor int
		patch int
	}{
		{"full version", "v0.2.3", 0, 2, 3},
		{"without v", "1.2.3", 1, 2, 3},
		{"major only", "v1", 1, 0, 0},
		{"major.minor only", "v1.2", 1, 2, 0},
		{"double digits", "v10.20.30", 10, 20, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := ParseVersion(tt.input)
			if err != nil {
				t.Errorf("ParseVersion(%q) returned error: %v", tt.input, err)
				return
			}
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("ParseVersion(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		})
	}
}
