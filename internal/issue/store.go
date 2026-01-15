package issue

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store manages issues in a directory
type Store struct {
	baseDir string
}

// NewStore creates a new Store
func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// List returns all issues, optionally filtered by state
func (s *Store) List(states ...State) ([]*Issue, error) {
	if len(states) == 0 {
		states = AllStates()
	}

	var issues []*Issue

	for _, state := range states {
		dir := filepath.Join(s.baseDir, StateDir(state))
		stateIssues, err := s.loadFromDir(dir, state)
		if err != nil {
			// 디렉토리가 없으면 무시
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		issues = append(issues, stateIssues...)
	}

	// 번호로 정렬
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Number < issues[j].Number
	})

	return issues, nil
}

// loadFromDir loads all issues from a directory
func (s *Store) loadFromDir(dir string, state State) ([]*Issue, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		issue, err := Parse(filePath)
		if err != nil {
			// 파싱 실패한 파일은 건너뜀
			continue
		}

		// 디렉토리 기반 상태로 덮어씀
		issue.State = state
		issues = append(issues, issue)
	}

	return issues, nil
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

// Move changes the state of an issue by moving it to a different directory
func (s *Store) Move(number int, newState State) error {
	issue, err := s.Get(number)
	if err != nil {
		return err
	}

	if issue.State == newState {
		return nil // 이미 같은 상태
	}

	// 새 경로 계산
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
