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

func TestFlatStructureList(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "zap-test-flat-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create issues with different states in flat directory
	issues := []struct {
		filename string
		number   int
		state    string
	}{
		{"001-open-issue.md", 1, "open"},
		{"002-wip-issue.md", 2, "wip"},
		{"003-done-issue.md", 3, "done"},
	}

	for _, iss := range issues {
		content := `---
number: ` + string(rune('0'+iss.number)) + `
title: "Test Issue ` + string(rune('0'+iss.number)) + `"
state: ` + iss.state + `
labels: []
assignees: []
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

Body content.
`
		if err := os.WriteFile(filepath.Join(tempDir, iss.filename), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStore(tempDir)

	// Test List all
	all, err := store.List()
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 issues, got %d", len(all))
	}

	// Test List by state - open
	open, err := store.List(StateOpen)
	if err != nil {
		t.Fatalf("List open failed: %v", err)
	}
	if len(open) != 1 {
		t.Errorf("Expected 1 open issue, got %d", len(open))
	}

	// Test List by state - wip
	wip, err := store.List(StateWip)
	if err != nil {
		t.Fatalf("List wip failed: %v", err)
	}
	if len(wip) != 1 {
		t.Errorf("Expected 1 wip issue, got %d", len(wip))
	}

	// Verify state comes from frontmatter, not directory
	if len(open) > 0 && open[0].State != StateOpen {
		t.Errorf("Expected state open, got %s", open[0].State)
	}
}

func TestUpdateState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zap-test-update-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	content := `---
number: 1
title: "Test Issue"
state: open
labels: []
assignees: []
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

Body content.
`
	filePath := filepath.Join(tempDir, "001-test.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tempDir)

	// Get the issue
	issue, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get issue failed: %v", err)
	}

	if issue.State != StateOpen {
		t.Errorf("Initial state = %v, want open", issue.State)
	}

	// Save original updated_at before UpdateState modifies the issue
	originalUpdatedAt := issue.UpdatedAt

	// Update state
	err = store.UpdateState(issue, StateWip)
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Verify state changed by re-reading
	updatedIssue, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get updated issue failed: %v", err)
	}

	if updatedIssue.State != StateWip {
		t.Errorf("Updated state = %v, want wip", updatedIssue.State)
	}

	// Verify file still exists in same location
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File should still exist at original location")
	}

	// Verify updated_at was changed (comparing with the saved original value)
	if !updatedIssue.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("updated_at should be updated: original=%v, updated=%v", originalUpdatedAt, updatedIssue.UpdatedAt)
	}
}

func TestDetectLegacyStructure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zap-test-legacy-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)

	// Test with no structure
	info, err := store.DetectLegacyStructure()
	if err != nil {
		t.Fatalf("DetectLegacyStructure failed: %v", err)
	}
	if info.HasLegacyStructure {
		t.Error("Should not detect legacy structure when no directories exist")
	}

	// Create legacy structure
	for _, state := range []string{"open", "done"} {
		if err := os.MkdirAll(filepath.Join(tempDir, state), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create issue in open directory
	content := `---
number: 1
title: "Legacy Issue"
state: open
labels: []
assignees: []
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

Body.
`
	if err := os.WriteFile(filepath.Join(tempDir, "open", "001-legacy.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Test detection
	info, err = store.DetectLegacyStructure()
	if err != nil {
		t.Fatalf("DetectLegacyStructure failed: %v", err)
	}

	if !info.HasLegacyStructure {
		t.Error("Should detect legacy structure")
	}
	if info.TotalIssues != 1 {
		t.Errorf("Expected 1 issue, got %d", info.TotalIssues)
	}
	if len(info.IssuesByState[StateOpen]) != 1 {
		t.Errorf("Expected 1 open issue, got %d", len(info.IssuesByState[StateOpen]))
	}
}

func TestMigration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zap-test-migrate-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create legacy structure
	openDir := filepath.Join(tempDir, "open")
	doneDir := filepath.Join(tempDir, "done")
	if err := os.MkdirAll(openDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(doneDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create issues in legacy directories
	openContent := `---
number: 1
title: "Open Issue"
state: open
labels: []
assignees: []
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

Open issue body.
`
	doneContent := `---
number: 2
title: "Done Issue"
state: done
labels: []
assignees: []
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

Done issue body.
`
	if err := os.WriteFile(filepath.Join(openDir, "001-open.md"), []byte(openContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(doneDir, "002-done.md"), []byte(doneContent), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tempDir)

	// Detect legacy structure
	info, err := store.DetectLegacyStructure()
	if err != nil {
		t.Fatalf("DetectLegacyStructure failed: %v", err)
	}
	if !info.HasLegacyStructure {
		t.Fatal("Should detect legacy structure")
	}
	if info.TotalIssues != 2 {
		t.Errorf("Expected 2 issues, got %d", info.TotalIssues)
	}

	// Migrate
	result, err := store.Migrate()
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if result.Migrated != 2 {
		t.Errorf("Expected 2 migrated, got %d", result.Migrated)
	}
	if result.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", result.Failed)
	}

	// Verify files moved to flat structure
	if _, err := os.Stat(filepath.Join(tempDir, "001-open.md")); os.IsNotExist(err) {
		t.Error("001-open.md should exist in flat structure")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "002-done.md")); os.IsNotExist(err) {
		t.Error("002-done.md should exist in flat structure")
	}

	// Verify old files are gone
	if _, err := os.Stat(filepath.Join(openDir, "001-open.md")); !os.IsNotExist(err) {
		t.Error("001-open.md should not exist in legacy location")
	}
	if _, err := os.Stat(filepath.Join(doneDir, "002-done.md")); !os.IsNotExist(err) {
		t.Error("002-done.md should not exist in legacy location")
	}

	// Verify issues can be read from flat structure
	issues, err := store.List()
	if err != nil {
		t.Fatalf("List after migration failed: %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues after migration, got %d", len(issues))
	}

	// Verify states are preserved
	openIssue, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get open issue failed: %v", err)
	}
	if openIssue.State != StateOpen {
		t.Errorf("Open issue state = %v, want open", openIssue.State)
	}

	doneIssue, err := store.Get(2)
	if err != nil {
		t.Fatalf("Get done issue failed: %v", err)
	}
	if doneIssue.State != StateDone {
		t.Errorf("Done issue state = %v, want done", doneIssue.State)
	}
}

func TestMoveInFlatStructure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zap-test-move-flat-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	content := `---
number: 1
title: "Test Issue"
state: open
labels: []
assignees: []
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

Body content.
`
	filePath := filepath.Join(tempDir, "001-test.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tempDir)

	// Move using Move() function (should use UpdateState internally for flat structure)
	err = store.Move(1, StateWip)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	// Verify state changed
	issue, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get issue failed: %v", err)
	}
	if issue.State != StateWip {
		t.Errorf("State = %v, want wip", issue.State)
	}

	// Verify file still in same location (flat structure behavior)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File should still exist at original location")
	}
}
