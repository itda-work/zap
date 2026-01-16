package issue

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreWarnings(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "zap-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create open directory
	openDir := filepath.Join(tempDir, "open")
	if err := os.MkdirAll(openDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a valid issue file
	validContent := `---
number: 1
title: "Valid Issue"
state: open
labels: []
assignees: []
created_at: 2024-01-01
updated_at: 2024-01-01
---

Valid issue body.
`
	if err := os.WriteFile(filepath.Join(openDir, "001-valid-issue.md"), []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an invalid issue file (no frontmatter)
	invalidContent := `# No Frontmatter
This file has no YAML frontmatter.
`
	if err := os.WriteFile(filepath.Join(openDir, "002-broken-issue.md"), []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create another invalid file (incomplete frontmatter)
	incompleteContent := `---
title: "Missing number field"
---
Body content.
`
	if err := os.WriteFile(filepath.Join(openDir, "003-incomplete-issue.md"), []byte(incompleteContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create store and list issues
	store := NewStore(tempDir)
	issues, err := store.List(StateOpen)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have 1 valid issue (003 has number=0 which is technically valid)
	// Actually, let's check the results
	t.Logf("Found %d issues", len(issues))
	for _, iss := range issues {
		t.Logf("  Issue #%d: %s", iss.Number, iss.Title)
	}

	// Check warnings
	warnings := store.Warnings()
	t.Logf("Found %d warnings", len(warnings))
	for _, w := range warnings {
		t.Logf("  Warning: %s - %s", w.FileName, w.Error)
	}

	// We expect at least 1 warning (the file without frontmatter)
	if len(warnings) < 1 {
		t.Errorf("Expected at least 1 warning, got %d", len(warnings))
	}

	// Check that the warning contains the expected filename
	found := false
	for _, w := range warnings {
		if w.FileName == "002-broken-issue.md" {
			found = true
			if w.State != StateOpen {
				t.Errorf("Warning state = %v, want %v", w.State, StateOpen)
			}
			break
		}
	}
	if !found {
		t.Error("Expected warning for 002-broken-issue.md not found")
	}
}

func TestStoreWarningsReset(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "zap-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create open directory with one broken file
	openDir := filepath.Join(tempDir, "open")
	if err := os.MkdirAll(openDir, 0755); err != nil {
		t.Fatal(err)
	}

	invalidContent := `# No Frontmatter`
	if err := os.WriteFile(filepath.Join(openDir, "001-broken.md"), []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tempDir)

	// First call should populate warnings
	store.List(StateOpen)
	if len(store.Warnings()) == 0 {
		t.Error("Expected warnings after first List call")
	}

	// Remove the broken file
	os.Remove(filepath.Join(openDir, "001-broken.md"))

	// Second call should reset warnings
	store.List(StateOpen)
	if len(store.Warnings()) != 0 {
		t.Errorf("Expected 0 warnings after removing broken file, got %d", len(store.Warnings()))
	}
}
