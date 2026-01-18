package issue

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConflictDetector_DetectDuplicateFilenames(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "zap-conflict-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create two files with same number prefix
	file1Content := `---
number: 1
title: "First Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-10T00:00:00Z
updated_at: 2026-01-10T00:00:00Z
---

First issue content.
`
	file2Content := `---
number: 1
title: "Second Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

Second issue content.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "001-first.md"), []byte(file1Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "001-second.md"), []byte(file2Content), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewConflictDetector(tmpDir)
	conflicts, err := detector.DetectConflicts()
	if err != nil {
		t.Fatal(err)
	}

	if len(conflicts) == 0 {
		t.Fatal("Expected at least one conflict, got none")
	}

	// Should detect duplicate filename
	found := false
	for _, c := range conflicts {
		if c.Type == ConflictDuplicateFilename && c.Number == 1 {
			found = true
			if len(c.Files) != 2 {
				t.Errorf("Expected 2 files in conflict, got %d", len(c.Files))
			}
			// The later-created file should be marked for renumbering
			if c.ToRenumber == nil {
				t.Error("Expected ToRenumber to be set")
			} else if c.ToRenumber.FileName != "001-second.md" {
				t.Errorf("Expected 001-second.md to be renumbered, got %s", c.ToRenumber.FileName)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find duplicate filename conflict for number 1")
	}
}

func TestConflictDetector_DetectMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zap-conflict-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file where filename number differs from frontmatter number
	content := `---
number: 5
title: "Mismatched Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-10T00:00:00Z
updated_at: 2026-01-10T00:00:00Z
---

Content.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "003-issue.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewConflictDetector(tmpDir)
	conflicts, err := detector.DetectConflicts()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, c := range conflicts {
		if c.Type == ConflictMismatch {
			found = true
			if c.Number != 3 {
				t.Errorf("Expected conflict number 3 (filename), got %d", c.Number)
			}
			if c.NewNumber != 3 {
				t.Errorf("Expected new number 3, got %d", c.NewNumber)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find mismatch conflict")
	}
}

func TestConflictDetector_NoConflicts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zap-conflict-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create two properly numbered files
	file1Content := `---
number: 1
title: "First Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-10T00:00:00Z
updated_at: 2026-01-10T00:00:00Z
---

First issue content.
`
	file2Content := `---
number: 2
title: "Second Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

Second issue content.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "001-first.md"), []byte(file1Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "002-second.md"), []byte(file2Content), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewConflictDetector(tmpDir)
	conflicts, err := detector.DetectConflicts()
	if err != nil {
		t.Fatal(err)
	}

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(conflicts))
	}
}

func TestFileInfo_GetEffectiveCreatedAt(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	tests := []struct {
		name     string
		fi       *FileInfo
		expected time.Time
	}{
		{
			name: "git time takes precedence",
			fi: &FileInfo{
				CreatedAt:    now,
				GitCreatedAt: &earlier,
			},
			expected: earlier,
		},
		{
			name: "falls back to created_at",
			fi: &FileInfo{
				CreatedAt:    earlier,
				GitCreatedAt: nil,
			},
			expected: earlier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fi.GetEffectiveCreatedAt()
			if !result.Equal(tt.expected) {
				t.Errorf("GetEffectiveCreatedAt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractSlugFromFilename(t *testing.T) {
	// This tests the internal logic used by renumberIssue
	tests := []struct {
		filename string
		expected string
	}{
		{"001-feature-name.md", "feature-name"},
		{"123-bug-fix.md", "bug-fix"},
		{"001-한글-slug.md", "한글-slug"},
		{"001.md", ""},
		{"no-number.md", "number"}, // First dash found, returns rest (after .md removed)
	}

	for _, tt := range tests {
		// Extract logic inline since extractSlugFromFilename is in cli package
		name := tt.filename[:len(tt.filename)-3] // Remove .md
		idx := -1
		for i, c := range name {
			if c == '-' {
				idx = i
				break
			}
		}
		var result string
		if idx != -1 {
			result = name[idx+1:]
		}

		if result != tt.expected {
			t.Errorf("extractSlug(%q) = %q, want %q", tt.filename, result, tt.expected)
		}
	}
}
