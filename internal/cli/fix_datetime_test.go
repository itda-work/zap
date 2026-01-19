package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itda-work/zap/internal/issue"
)

func TestTimeEqualRFC3339(t *testing.T) {
	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected bool
	}{
		{
			name:     "same time same timezone",
			t1:       time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			t2:       time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "same instant different timezone",
			t1:       time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			t2:       time.Date(2026, 1, 18, 0, 30, 0, 0, time.FixedZone("KST", 9*60*60)),
			expected: true,
		},
		{
			name:     "different times",
			t1:       time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			t2:       time.Date(2026, 1, 17, 16, 30, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "subsecond difference ignored in RFC3339",
			t1:       time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			t2:       time.Date(2026, 1, 17, 15, 30, 0, 500000000, time.UTC),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeEqualRFC3339(tt.t1, tt.t2)
			if result != tt.expected {
				t.Errorf("timeEqualRFC3339(%v, %v) = %v, want %v", tt.t1, tt.t2, result, tt.expected)
			}
		})
	}
}

func TestFixDatetimeFormatDryRun(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "zap-fix-datetime-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an issue with local timezone
	localTime := time.Date(2026, 1, 17, 15, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	issueContent := `---
number: 1
title: "Test Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-17T15:30:00+09:00
updated_at: 2026-01-17T15:30:00+09:00
---

Test body.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "001-test-issue.md"), []byte(issueContent), 0644); err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	// Read the issue
	store := issue.NewStore(tmpDir)
	iss, err := store.Get(1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	// Verify the time is parsed with offset
	if iss.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	// The UTC equivalent should be 06:30:00Z
	expectedUTC := localTime.UTC()
	if !iss.CreatedAt.UTC().Equal(expectedUTC) {
		t.Errorf("CreatedAt UTC = %v, want %v", iss.CreatedAt.UTC(), expectedUTC)
	}
}

func TestFixDatetimeFormatApply(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "zap-fix-datetime-apply-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an issue with non-UTC timezone
	issueContent := `---
number: 1
title: "Test Issue"
state: open
labels: []
assignees: []
created_at: 2026-01-17T15:30:00+09:00
updated_at: 2026-01-17T16:00:00+09:00
---

Test body.
`
	filePath := filepath.Join(tmpDir, "001-test-issue.md")
	if err := os.WriteFile(filePath, []byte(issueContent), 0644); err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	// Load, convert to UTC, and save
	store := issue.NewStore(tmpDir)
	iss, err := store.Get(1)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	// Convert to UTC
	iss.CreatedAt = iss.CreatedAt.UTC()
	iss.UpdatedAt = iss.UpdatedAt.UTC()

	// Serialize and write
	data, err := issue.Serialize(iss)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Read file and verify it contains UTC format
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Should contain UTC timestamps (ending with Z)
	if !strings.Contains(contentStr, "2026-01-17T06:30:00Z") {
		t.Errorf("File should contain UTC created_at, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "2026-01-17T07:00:00Z") {
		t.Errorf("File should contain UTC updated_at, got:\n%s", contentStr)
	}

	// Should NOT contain the original +09:00 offset
	if strings.Contains(contentStr, "+09:00") {
		t.Errorf("File should not contain +09:00 offset, got:\n%s", contentStr)
	}
}

func TestFixDatetimeFormatSpecificIssue(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "zap-fix-datetime-specific-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create two issues
	issue1 := `---
number: 1
title: "Issue One"
state: open
labels: []
assignees: []
created_at: 2026-01-17T15:30:00+09:00
updated_at: 2026-01-17T15:30:00+09:00
---
`
	issue2 := `---
number: 2
title: "Issue Two"
state: open
labels: []
assignees: []
created_at: 2026-01-18T10:00:00+09:00
updated_at: 2026-01-18T10:00:00+09:00
---
`

	if err := os.WriteFile(filepath.Join(tmpDir, "001-issue-one.md"), []byte(issue1), 0644); err != nil {
		t.Fatalf("Failed to create issue 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "002-issue-two.md"), []byte(issue2), 0644); err != nil {
		t.Fatalf("Failed to create issue 2: %v", err)
	}

	// Verify both issues are loaded
	store := issue.NewStore(tmpDir)
	issues, err := store.List(issue.AllStates()...)
	if err != nil {
		t.Fatalf("Failed to list issues: %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}
}

func TestSerializeRFC3339UTC(t *testing.T) {
	// Test that Serialize outputs RFC3339 UTC format
	iss := &issue.Issue{
		Number:    1,
		Title:     "Test",
		State:     issue.StateOpen,
		Labels:    []string{},
		Assignees: []string{},
		CreatedAt: time.Date(2026, 1, 17, 15, 30, 0, 0, time.FixedZone("KST", 9*60*60)),
		UpdatedAt: time.Date(2026, 1, 17, 16, 0, 0, 0, time.FixedZone("KST", 9*60*60)),
		Body:      "Test body",
	}

	data, err := issue.Serialize(iss)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	content := string(data)

	// Should contain UTC timestamps
	if !strings.Contains(content, "2026-01-17T06:30:00Z") {
		t.Errorf("Should contain UTC created_at (06:30:00Z), got:\n%s", content)
	}
	if !strings.Contains(content, "2026-01-17T07:00:00Z") {
		t.Errorf("Should contain UTC updated_at (07:00:00Z), got:\n%s", content)
	}
}

func TestSerializeWithClosedAt(t *testing.T) {
	closedTime := time.Date(2026, 1, 17, 18, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	iss := &issue.Issue{
		Number:    1,
		Title:     "Closed Issue",
		State:     issue.StateDone,
		Labels:    []string{},
		Assignees: []string{},
		CreatedAt: time.Date(2026, 1, 17, 15, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 17, 18, 0, 0, 0, time.UTC),
		ClosedAt:  &closedTime,
		Body:      "",
	}

	data, err := issue.Serialize(iss)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	content := string(data)

	// ClosedAt should be in UTC (18:00 KST = 09:00 UTC)
	// YAML may quote strings, so check for the timestamp value
	if !strings.Contains(content, "2026-01-17T09:00:00Z") {
		t.Errorf("Should contain UTC closed_at timestamp, got:\n%s", content)
	}
}
