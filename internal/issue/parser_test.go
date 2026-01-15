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
