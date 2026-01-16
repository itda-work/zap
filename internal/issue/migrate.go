package issue

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MigrationInfo contains information about detected legacy structure
type MigrationInfo struct {
	HasLegacyStructure bool
	IssuesByState      map[State][]string // filename list per state
	TotalIssues        int
}

// MigrateResult contains the result of migration
type MigrateResult struct {
	Migrated    int
	Failed      int
	FailedFiles []string
	Errors      []string
}

// DetectLegacyStructure checks if old directory-based structure exists
func (s *Store) DetectLegacyStructure() (*MigrationInfo, error) {
	info := &MigrationInfo{
		IssuesByState: make(map[State][]string),
	}

	for _, state := range AllStates() {
		dir := filepath.Join(s.baseDir, StateDir(state))
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			info.IssuesByState[state] = append(info.IssuesByState[state], entry.Name())
			info.TotalIssues++
		}
	}

	// Has legacy structure if any state directory contains .md files
	info.HasLegacyStructure = info.TotalIssues > 0

	return info, nil
}

// Migrate converts from directory-based to flat structure
func (s *Store) Migrate() (*MigrateResult, error) {
	result := &MigrateResult{}

	info, err := s.DetectLegacyStructure()
	if err != nil {
		return nil, fmt.Errorf("failed to detect legacy structure: %w", err)
	}

	if !info.HasLegacyStructure {
		return nil, fmt.Errorf("no legacy structure detected")
	}

	for state, files := range info.IssuesByState {
		for _, filename := range files {
			srcPath := filepath.Join(s.baseDir, StateDir(state), filename)
			dstPath := filepath.Join(s.baseDir, filename)

			// Check if destination already exists
			if _, err := os.Stat(dstPath); err == nil {
				result.Failed++
				result.FailedFiles = append(result.FailedFiles, filename)
				result.Errors = append(result.Errors, fmt.Sprintf("%s: destination file already exists", filename))
				continue
			}

			// Step 1: Update frontmatter state before moving
			if err := s.updateFrontmatterState(srcPath, state); err != nil {
				result.Failed++
				result.FailedFiles = append(result.FailedFiles, filename)
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filename, err))
				continue
			}

			// Step 2: Try git mv first
			if err := s.gitMove(srcPath, dstPath); err != nil {
				// Step 3: Fall back to regular mv if git mv fails
				if err := os.Rename(srcPath, dstPath); err != nil {
					result.Failed++
					result.FailedFiles = append(result.FailedFiles, filename)
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filename, err))
					continue
				}
			}

			result.Migrated++
		}
	}

	// Clean up empty state directories
	for _, state := range AllStates() {
		dir := filepath.Join(s.baseDir, StateDir(state))
		s.removeIfEmpty(dir)
	}

	return result, nil
}

// updateFrontmatterState ensures the frontmatter state matches the source directory
func (s *Store) updateFrontmatterState(filePath string, state State) error {
	issue, err := Parse(filePath)
	if err != nil {
		return err
	}

	// Update state to match directory (ensure consistency before move)
	if issue.State != state {
		issue.State = state
		data, err := Serialize(issue)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

// gitMove attempts to use git mv for the file
func (s *Store) gitMove(src, dst string) error {
	cmd := exec.Command("git", "mv", src, dst)
	// Set working directory to the parent of .issues to ensure git works
	cmd.Dir = filepath.Dir(s.baseDir)
	return cmd.Run()
}

// removeIfEmpty removes directory if it's empty (except .gitkeep)
func (s *Store) removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Check if directory only has .gitkeep or is empty
	hasOnlyGitkeep := true
	for _, entry := range entries {
		if entry.Name() != ".gitkeep" {
			hasOnlyGitkeep = false
			break
		}
	}

	if hasOnlyGitkeep {
		// Remove .gitkeep first if it exists
		gitkeepPath := filepath.Join(dir, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); err == nil {
			// Try git rm first, then regular rm
			cmd := exec.Command("git", "rm", "-f", gitkeepPath)
			cmd.Dir = filepath.Dir(s.baseDir)
			if cmd.Run() != nil {
				os.Remove(gitkeepPath)
			}
		}
		os.Remove(dir)
	}
}
