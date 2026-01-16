package issue

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ParseFailure represents a file that failed to parse.
type ParseFailure struct {
	FilePath string // Full path to the file
	FileName string // Just the filename
	Error    string // Error message
	State    State  // Which state directory it was in
	Content  string // File content (loaded on demand)
}

// Store manages issues in a directory
type Store struct {
	baseDir  string
	warnings []ParseFailure // Collected during List operations
}

// NewStore creates a new Store
func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// Warnings returns parse failures from the last List operation.
func (s *Store) Warnings() []ParseFailure {
	return s.warnings
}

// WarningsWithContent returns parse failures with file content loaded.
// This is useful for repair operations that need the original content.
func (s *Store) WarningsWithContent() []ParseFailure {
	result := make([]ParseFailure, len(s.warnings))
	for i, w := range s.warnings {
		result[i] = w
		if content, err := os.ReadFile(w.FilePath); err == nil {
			result[i].Content = string(content)
		}
	}
	return result
}

// GetFailureByNumber finds a parse failure by extracting number from filename.
// Returns nil if not found. The filename format is expected to be "NNN-title.md".
func (s *Store) GetFailureByNumber(number int) *ParseFailure {
	prefix := fmt.Sprintf("%d-", number)
	prefixPadded := fmt.Sprintf("%03d-", number)

	for _, w := range s.warnings {
		if strings.HasPrefix(w.FileName, prefix) || strings.HasPrefix(w.FileName, prefixPadded) {
			content, _ := os.ReadFile(w.FilePath)
			return &ParseFailure{
				FilePath: w.FilePath,
				FileName: w.FileName,
				Error:    w.Error,
				State:    w.State,
				Content:  string(content),
			}
		}
	}
	return nil
}

// List returns all issues, optionally filtered by state.
// Call Warnings() after List() to get any parse failures.
// Supports both flat structure (.issues/*.md) and legacy structure (.issues/{state}/*.md).
func (s *Store) List(states ...State) ([]*Issue, error) {
	if len(states) == 0 {
		states = AllStates()
	}

	// Reset warnings for this operation
	s.warnings = nil

	// Create state filter map
	stateFilter := make(map[State]bool)
	for _, state := range states {
		stateFilter[state] = true
	}

	// Try flat structure first
	flatIssues, flatFailures, flatErr := s.loadFromFlatDir()

	// Check if we have any flat structure issues
	if flatErr == nil && len(flatIssues) > 0 {
		// Filter by state
		var filtered []*Issue
		for _, issue := range flatIssues {
			if stateFilter[issue.State] {
				filtered = append(filtered, issue)
			}
		}
		s.warnings = flatFailures

		// Sort by number
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Number < filtered[j].Number
		})

		return filtered, nil
	}

	// Fallback to legacy structure
	var issues []*Issue

	for _, state := range states {
		dir := filepath.Join(s.baseDir, StateDir(state))
		stateIssues, failures, err := s.loadFromDir(dir, state)
		if err != nil {
			// 디렉토리가 없으면 무시
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		issues = append(issues, stateIssues...)
		s.warnings = append(s.warnings, failures...)
	}

	// 번호로 정렬
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Number < issues[j].Number
	})

	return issues, nil
}

// loadFromDir loads all issues from a legacy directory, returning both successful parses and failures.
// This is used for backward compatibility with directory-based state management.
func (s *Store) loadFromDir(dir string, state State) ([]*Issue, []ParseFailure, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	var issues []*Issue
	var failures []ParseFailure

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		issue, err := Parse(filePath)
		if err != nil {
			// 파싱 실패 기록
			failures = append(failures, ParseFailure{
				FilePath: filePath,
				FileName: entry.Name(),
				Error:    err.Error(),
				State:    state,
			})
			continue
		}

		// 디렉토리 기반 상태로 덮어씀 (legacy behavior)
		issue.State = state
		issues = append(issues, issue)
	}

	return issues, failures, nil
}

// loadFromFlatDir loads all issues from the flat directory structure.
// State is determined from frontmatter, not directory location.
func (s *Store) loadFromFlatDir() ([]*Issue, []ParseFailure, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, nil, err
	}

	var issues []*Issue
	var failures []ParseFailure

	for _, entry := range entries {
		// Skip directories and non-markdown files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name())
		issue, err := Parse(filePath)
		if err != nil {
			failures = append(failures, ParseFailure{
				FilePath: filePath,
				FileName: entry.Name(),
				Error:    err.Error(),
				State:    "", // Unknown state for flat files
			})
			continue
		}

		// State comes from frontmatter (already parsed, no override)
		issues = append(issues, issue)
	}

	return issues, failures, nil
}

// Get returns a single issue by number
func (s *Store) Get(number int) (*Issue, error) {
	issues, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		if issue.Number == number {
			return issue, nil
		}
	}

	return nil, fmt.Errorf("issue #%d not found", number)
}

// Move changes the state of an issue.
// For flat structure: updates frontmatter state.
// For legacy structure: moves file to new directory.
func (s *Store) Move(number int, newState State) error {
	issue, err := s.Get(number)
	if err != nil {
		return err
	}

	if issue.State == newState {
		return nil // 이미 같은 상태
	}

	// Check if using flat structure (file is directly in baseDir)
	if filepath.Dir(issue.FilePath) == s.baseDir {
		// Flat structure: update frontmatter
		return s.UpdateState(issue, newState)
	}

	// Legacy structure: move file to new directory
	fileName := filepath.Base(issue.FilePath)
	newDir := filepath.Join(s.baseDir, StateDir(newState))
	newPath := filepath.Join(newDir, fileName)

	// 디렉토리 생성 (없으면)
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 파일 이동
	if err := os.Rename(issue.FilePath, newPath); err != nil {
		return fmt.Errorf("failed to move issue: %w", err)
	}

	return nil
}

// UpdateState changes the state of an issue by updating its frontmatter.
// This is used for flat structure where files don't move between directories.
func (s *Store) UpdateState(issue *Issue, newState State) error {
	if issue.State == newState {
		return nil
	}

	// Update state and timestamps
	issue.State = newState
	issue.UpdatedAt = time.Now()

	// Handle closed_at timestamp
	if newState == StateDone || newState == StateClosed {
		now := time.Now()
		issue.ClosedAt = &now
	} else {
		issue.ClosedAt = nil
	}

	// Serialize and write back
	data, err := Serialize(issue)
	if err != nil {
		return fmt.Errorf("failed to serialize issue: %w", err)
	}

	if err := os.WriteFile(issue.FilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write issue file: %w", err)
	}

	return nil
}

// Search searches issues by keyword in title and body
func (s *Store) Search(keyword string, titleOnly bool) ([]*Issue, error) {
	issues, err := s.List()
	if err != nil {
		return nil, err
	}

	keyword = strings.ToLower(keyword)
	var results []*Issue

	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue.Title), keyword) {
			results = append(results, issue)
			continue
		}

		if !titleOnly && strings.Contains(strings.ToLower(issue.Body), keyword) {
			results = append(results, issue)
		}
	}

	return results, nil
}

// Stats returns statistics about issues
type Stats struct {
	Total      int
	ByState    map[State]int
	ByLabel    map[string]int
	ByAssignee map[string]int
}

// Stats returns statistics about issues
func (s *Store) Stats() (*Stats, error) {
	issues, err := s.List()
	if err != nil {
		return nil, err
	}

	stats := &Stats{
		Total:      len(issues),
		ByState:    make(map[State]int),
		ByLabel:    make(map[string]int),
		ByAssignee: make(map[string]int),
	}

	for _, issue := range issues {
		stats.ByState[issue.State]++

		for _, label := range issue.Labels {
			stats.ByLabel[label]++
		}

		for _, assignee := range issue.Assignees {
			stats.ByAssignee[assignee]++
		}
	}

	return stats, nil
}

// FilterByLabel returns issues with a specific label
func (s *Store) FilterByLabel(label string, states ...State) ([]*Issue, error) {
	issues, err := s.List(states...)
	if err != nil {
		return nil, err
	}

	var results []*Issue
	for _, issue := range issues {
		for _, l := range issue.Labels {
			if strings.EqualFold(l, label) {
				results = append(results, issue)
				break
			}
		}
	}

	return results, nil
}

// FilterByAssignee returns issues assigned to a specific person
func (s *Store) FilterByAssignee(assignee string, states ...State) ([]*Issue, error) {
	issues, err := s.List(states...)
	if err != nil {
		return nil, err
	}

	var results []*Issue
	for _, issue := range issues {
		for _, a := range issue.Assignees {
			if strings.EqualFold(a, assignee) {
				results = append(results, issue)
				break
			}
		}
	}

	return results, nil
}
