package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itda-work/zap/internal/issue"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple english",
			title:    "Fix login bug",
			expected: "fix-login-bug",
		},
		{
			name:     "with prefix feat:",
			title:    "feat: Add user authentication",
			expected: "add-user-authentication",
		},
		{
			name:     "with prefix fix:",
			title:    "fix: Handle null pointer",
			expected: "handle-null-pointer",
		},
		{
			name:     "korean only",
			title:    "사용자 인증 추가",
			expected: "사용자-인증-추가",
		},
		{
			name:     "mixed korean and english",
			title:    "User 인증 기능 추가",
			expected: "user-인증-기능-추가",
		},
		{
			name:     "special characters",
			title:    "Fix bug #123: Handle @mention",
			expected: "fix-bug-123-handle-mention",
		},
		{
			name:     "multiple spaces",
			title:    "Fix   multiple   spaces",
			expected: "fix-multiple-spaces",
		},
		{
			name:     "underscores",
			title:    "fix_underscore_title",
			expected: "fix-underscore-title",
		},
		{
			name:     "long title truncation",
			title:    "This is a very long title that should be truncated at fifty characters boundary",
			expected: "this-is-a-very-long-title-that-should-be",
		},
		{
			name:     "empty after processing",
			title:    "!@#$%",
			expected: "issue",
		},
		{
			name:     "numbers",
			title:    "Update API v2 endpoints",
			expected: "update-api-v2-endpoints",
		},
		{
			name:     "leading special chars",
			title:    "## Fix header bug",
			expected: "fix-header-bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.title)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestExtractNumberFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected int
	}{
		{
			name:     "three digit padded",
			filename: "001-feat-login.md",
			expected: 1,
		},
		{
			name:     "two digit",
			filename: "24-fix-bug.md",
			expected: 24,
		},
		{
			name:     "large number",
			filename: "999-final-issue.md",
			expected: 999,
		},
		{
			name:     "no hyphen",
			filename: "readme.md",
			expected: 0,
		},
		{
			name:     "non-numeric prefix",
			filename: "abc-title.md",
			expected: 0,
		},
		{
			name:     "empty filename",
			filename: "",
			expected: 0,
		},
		{
			name:     "just number",
			filename: "123-.md",
			expected: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNumberFromFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("extractNumberFromFilename(%q) = %d, want %d", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestFindNextIssueNumber(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // filename -> content
		expected int
	}{
		{
			name:     "empty directory",
			files:    map[string]string{},
			expected: 1,
		},
		{
			name: "sequential issues",
			files: map[string]string{
				"001-first.md": `---
number: 1
title: "First"
state: open
---`,
				"002-second.md": `---
number: 2
title: "Second"
state: open
---`,
			},
			expected: 3,
		},
		{
			name: "with gap",
			files: map[string]string{
				"001-first.md": `---
number: 1
title: "First"
state: open
---`,
				"005-fifth.md": `---
number: 5
title: "Fifth"
state: open
---`,
			},
			expected: 6,
		},
		{
			name: "with parse failure",
			files: map[string]string{
				"001-first.md": `---
number: 1
title: "First"
state: open
---`,
				"003-broken.md": `invalid content without frontmatter`,
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "zap-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Test
			store := issue.NewStore(tmpDir)
			result, err := findNextIssueNumber(store)
			if err != nil {
				t.Fatalf("findNextIssueNumber failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("findNextIssueNumber() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestNewCommandIntegration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "zap-new-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	issuesDir := filepath.Join(tmpDir, ".issues")
	if err := os.MkdirAll(issuesDir, 0755); err != nil {
		t.Fatalf("Failed to create issues dir: %v", err)
	}

	// Create an existing issue
	existingIssue := `---
number: 1
title: "Existing issue"
state: open
labels: []
assignees: []
created_at: 2026-01-17T00:00:00Z
updated_at: 2026-01-17T00:00:00Z
---
`
	if err := os.WriteFile(filepath.Join(issuesDir, "001-existing-issue.md"), []byte(existingIssue), 0644); err != nil {
		t.Fatalf("Failed to create existing issue: %v", err)
	}

	// Test finding next number
	store := issue.NewStore(issuesDir)
	nextNum, err := findNextIssueNumber(store)
	if err != nil {
		t.Fatalf("findNextIssueNumber failed: %v", err)
	}

	if nextNum != 2 {
		t.Errorf("Expected next number to be 2, got %d", nextNum)
	}

	// Test slug generation for various titles
	slugTests := []struct {
		title    string
		expected string
	}{
		{"New Feature", "new-feature"},
		{"버그 수정", "버그-수정"},
		{"feat: Add login", "add-login"},
	}

	for _, st := range slugTests {
		slug := generateSlug(st.title)
		if slug != st.expected {
			t.Errorf("generateSlug(%q) = %q, want %q", st.title, slug, st.expected)
		}
	}
}
