package issue

import (
	"testing"
	"time"
)

func TestParseBytes(t *testing.T) {
	content := `---
number: 1
title: "Test Issue"
state: open
labels:
  - bug
  - urgent
assignees:
  - alice
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

## Description

This is a test issue.
`

	issue, err := ParseBytes([]byte(content), "test.md")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if issue.Number != 1 {
		t.Errorf("Number = %d, want 1", issue.Number)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Issue")
	}

	if issue.State != StateOpen {
		t.Errorf("State = %q, want %q", issue.State, StateOpen)
	}

	if len(issue.Labels) != 2 {
		t.Errorf("Labels count = %d, want 2", len(issue.Labels))
	}

	if issue.Labels[0] != "bug" || issue.Labels[1] != "urgent" {
		t.Errorf("Labels = %v, want [bug, urgent]", issue.Labels)
	}

	if len(issue.Assignees) != 1 || issue.Assignees[0] != "alice" {
		t.Errorf("Assignees = %v, want [alice]", issue.Assignees)
	}

	if issue.FilePath != "test.md" {
		t.Errorf("FilePath = %q, want %q", issue.FilePath, "test.md")
	}

	if !containsString(issue.Body, "This is a test issue") {
		t.Errorf("Body should contain 'This is a test issue', got %q", issue.Body)
	}
}

func TestParseBytesMinimal(t *testing.T) {
	content := `---
number: 42
title: "Minimal Issue"
state: in-progress
---
`

	issue, err := ParseBytes([]byte(content), "minimal.md")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if issue.Number != 42 {
		t.Errorf("Number = %d, want 42", issue.Number)
	}

	if issue.State != StateInProgress {
		t.Errorf("State = %q, want %q", issue.State, StateInProgress)
	}

	if len(issue.Labels) != 0 {
		t.Errorf("Labels should be empty, got %v", issue.Labels)
	}
}

func TestParseBytesInvalidFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"no frontmatter", "Just some content"},
		{"unclosed frontmatter", "---\ntitle: test\n"},
		{"empty file", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.content), "test.md")
			if err == nil {
				t.Error("Expected error for invalid frontmatter")
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	issue := &Issue{
		Number:    1,
		Title:     "Test Issue",
		State:     StateOpen,
		Labels:    []string{"bug"},
		Assignees: []string{"alice"},
		CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:      "## Description\n\nTest body.",
	}

	data, err := Serialize(issue)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Parse it back
	parsed, err := ParseBytes(data, "test.md")
	if err != nil {
		t.Fatalf("ParseBytes failed on serialized data: %v", err)
	}

	if parsed.Number != issue.Number {
		t.Errorf("Number = %d, want %d", parsed.Number, issue.Number)
	}

	if parsed.Title != issue.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, issue.Title)
	}

	if parsed.State != issue.State {
		t.Errorf("State = %q, want %q", parsed.State, issue.State)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseBytesAlternativeFieldNames(t *testing.T) {
	// Test with "created" and "updated" instead of "created_at" and "updated_at"
	content := `---
number: 21
title: "iOS 빌드 환경 설정"
state: done
created: 2026-01-17 15:47
updated: 2026-01-17 15:48
---

# Description

This issue uses alternative field names.
`

	issue, err := ParseBytes([]byte(content), "test.md")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if issue.Number != 21 {
		t.Errorf("Number = %d, want 21", issue.Number)
	}

	if issue.Title != "iOS 빌드 환경 설정" {
		t.Errorf("Title = %q, want %q", issue.Title, "iOS 빌드 환경 설정")
	}

	if issue.State != StateDone {
		t.Errorf("State = %q, want %q", issue.State, StateDone)
	}

	// Check that dates were parsed correctly
	expectedCreated := time.Date(2026, 1, 17, 15, 47, 0, 0, time.UTC)
	if !issue.CreatedAt.Equal(expectedCreated) {
		t.Errorf("CreatedAt = %v, want %v", issue.CreatedAt, expectedCreated)
	}

	expectedUpdated := time.Date(2026, 1, 17, 15, 48, 0, 0, time.UTC)
	if !issue.UpdatedAt.Equal(expectedUpdated) {
		t.Errorf("UpdatedAt = %v, want %v", issue.UpdatedAt, expectedUpdated)
	}
}

func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "RFC3339",
			input:    "2026-01-17T15:47:00Z",
			expected: time.Date(2026, 1, 17, 15, 47, 0, 0, time.UTC),
		},
		{
			name:     "datetime without timezone",
			input:    "2026-01-17T15:47:00",
			expected: time.Date(2026, 1, 17, 15, 47, 0, 0, time.UTC),
		},
		{
			name:     "datetime with space",
			input:    "2026-01-17 15:47:00",
			expected: time.Date(2026, 1, 17, 15, 47, 0, 0, time.UTC),
		},
		{
			name:     "datetime without seconds",
			input:    "2026-01-17 15:47",
			expected: time.Date(2026, 1, 17, 15, 47, 0, 0, time.UTC),
		},
		{
			name:     "date only",
			input:    "2026-01-17",
			expected: time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "empty string",
			input:    "",
			expected: time.Time{},
		},
		{
			name:    "invalid format",
			input:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFlexibleTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !result.Equal(tt.expected) {
				t.Errorf("parseFlexibleTime(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
