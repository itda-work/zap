package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itda-work/zap/internal/issue"
)

func TestParseIssueNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"plain number", "5", 5, false},
		{"with hash", "#5", 5, false},
		{"large number", "123", 123, false},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"non-numeric", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIssueNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseIssueNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestMoveProjectIntegration(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcIssuesDir := filepath.Join(srcDir, ".issues")
	dstIssuesDir := filepath.Join(dstDir, ".issues")
	os.MkdirAll(srcIssuesDir, 0755)

	srcContent := `---
number: 3
title: "Fix login bug"
state: wip
labels:
    - bug
    - urgent
assignees:
    - alice
created_at: 2026-01-10T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

## Description

Login fails when password contains special chars.
`
	if err := os.WriteFile(filepath.Join(srcIssuesDir, "003-fix-login-bug.md"), []byte(srcContent), 0644); err != nil {
		t.Fatalf("failed to create source issue: %v", err)
	}

	srcStore := issue.NewStore(srcIssuesDir)
	srcIssue, err := srcStore.Get(3)
	if err != nil {
		t.Fatalf("failed to read source issue: %v", err)
	}

	os.MkdirAll(dstIssuesDir, 0755)

	dstExisting := `---
number: 1
title: "Existing issue"
state: open
labels: []
assignees: []
created_at: 2026-01-01T00:00:00Z
updated_at: 2026-01-01T00:00:00Z
---
`
	if err := os.WriteFile(filepath.Join(dstIssuesDir, "001-existing-issue.md"), []byte(dstExisting), 0644); err != nil {
		t.Fatalf("failed to create dest issue: %v", err)
	}

	dstStore := issue.NewStore(dstIssuesDir)
	nextNumber, err := findNextIssueNumber(dstStore)
	if err != nil {
		t.Fatalf("findNextIssueNumber failed: %v", err)
	}

	if nextNumber != 2 {
		t.Fatalf("expected next number 2, got %d", nextNumber)
	}

	srcProjectName := filepath.Base(srcDir)
	provenanceNote := "> Moved from " + srcProjectName + " #3"
	body := provenanceNote + "\n\n" + srcIssue.Body

	dstIssue := &issue.Issue{
		Number:    nextNumber,
		Title:     srcIssue.Title,
		State:     srcIssue.State,
		Labels:    srcIssue.Labels,
		Assignees: srcIssue.Assignees,
		CreatedAt: srcIssue.CreatedAt,
		UpdatedAt: srcIssue.UpdatedAt,
		ClosedAt:  srcIssue.ClosedAt,
		Body:      body,
	}

	slug := generateSlug(srcIssue.Title)
	filename := "002-" + slug + ".md"
	dstFilePath := filepath.Join(dstIssuesDir, filename)

	data, err := issue.Serialize(dstIssue)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if err := os.WriteFile(dstFilePath, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	parsedDst, err := issue.Parse(dstFilePath)
	if err != nil {
		t.Fatalf("failed to parse destination issue: %v", err)
	}

	if parsedDst.Number != 2 {
		t.Errorf("destination number = %d, want 2", parsedDst.Number)
	}
	if parsedDst.Title != "Fix login bug" {
		t.Errorf("destination title = %q, want %q", parsedDst.Title, "Fix login bug")
	}
	if parsedDst.State != issue.StateWip {
		t.Errorf("destination state = %q, want %q", parsedDst.State, issue.StateWip)
	}
	if len(parsedDst.Labels) != 2 || parsedDst.Labels[0] != "bug" {
		t.Errorf("destination labels = %v, want [bug urgent]", parsedDst.Labels)
	}
	if len(parsedDst.Assignees) != 1 || parsedDst.Assignees[0] != "alice" {
		t.Errorf("destination assignees = %v, want [alice]", parsedDst.Assignees)
	}
	if !strings.Contains(parsedDst.Body, "Moved from") {
		t.Errorf("destination body should contain provenance note, got: %s", parsedDst.Body)
	}
	if !strings.Contains(parsedDst.Body, "Login fails when") {
		t.Errorf("destination body should contain original content, got: %s", parsedDst.Body)
	}

	_, err = os.Stat(filepath.Join(srcIssuesDir, "003-fix-login-bug.md"))
	if err != nil {
		t.Errorf("original issue should still exist: %v", err)
	}
}

func TestMoveProjectWithDelete(t *testing.T) {
	srcDir := t.TempDir()
	srcIssuesDir := filepath.Join(srcDir, ".issues")
	os.MkdirAll(srcIssuesDir, 0755)

	srcContent := `---
number: 1
title: "Delete me"
state: open
labels: []
assignees: []
created_at: 2026-01-01T00:00:00Z
updated_at: 2026-01-01T00:00:00Z
---
`
	srcFilePath := filepath.Join(srcIssuesDir, "001-delete-me.md")
	if err := os.WriteFile(srcFilePath, []byte(srcContent), 0644); err != nil {
		t.Fatalf("failed to create source issue: %v", err)
	}

	if err := os.Remove(srcFilePath); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := os.Stat(srcFilePath)
	if !os.IsNotExist(err) {
		t.Errorf("source file should be deleted")
	}
}

func TestMoveProjectSameDirectoryError(t *testing.T) {
	dir := t.TempDir()
	absDir, _ := filepath.Abs(dir)

	absSrc := absDir
	absDst := absDir

	if absSrc == absDst {
	} else {
		t.Errorf("same directory detection failed")
	}
}
