package project

import (
	"fmt"
	"sort"

	"github.com/itda-work/zap/internal/issue"
)

// MultiStore manages multiple project stores
type MultiStore struct {
	projects map[string]*Project // alias -> project
	order    []string            // maintains order of projects
}

// NewMultiStore creates a MultiStore from project specifications
func NewMultiStore(specs []ProjectSpec, issuesDir string) (*MultiStore, error) {
	ms := &MultiStore{
		projects: make(map[string]*Project),
		order:    make([]string, 0, len(specs)),
	}

	for _, spec := range specs {
		proj, err := NewProject(spec, issuesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create project from %s: %w", spec.Path, err)
		}

		// Check for alias collision
		if _, exists := ms.projects[proj.Alias]; exists {
			return nil, fmt.Errorf("duplicate project alias: %s", proj.Alias)
		}

		ms.projects[proj.Alias] = proj
		ms.order = append(ms.order, proj.Alias)
	}

	return ms, nil
}

// Projects returns all projects in order
func (ms *MultiStore) Projects() []*Project {
	result := make([]*Project, len(ms.order))
	for i, alias := range ms.order {
		result[i] = ms.projects[alias]
	}
	return result
}

// GetProject returns a project by alias
func (ms *MultiStore) GetProject(alias string) (*Project, bool) {
	proj, ok := ms.projects[alias]
	return proj, ok
}

// ProjectCount returns the number of projects
func (ms *MultiStore) ProjectCount() int {
	return len(ms.projects)
}

// IsMultiProject returns true if there are multiple projects
func (ms *MultiStore) IsMultiProject() bool {
	return len(ms.projects) > 1
}

// ListAll returns all issues from all projects, optionally filtered by state
func (ms *MultiStore) ListAll(states ...issue.State) ([]*ProjectIssue, error) {
	var allIssues []*ProjectIssue

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		issues, err := proj.Store.List(states...)
		if err != nil {
			return nil, fmt.Errorf("failed to list issues from %s: %w", alias, err)
		}

		for _, iss := range issues {
			allIssues = append(allIssues, NewProjectIssue(iss, alias))
		}
	}

	// Sort by project alias, then by number
	sort.Slice(allIssues, func(i, j int) bool {
		if allIssues[i].Project != allIssues[j].Project {
			return allIssues[i].Project < allIssues[j].Project
		}
		return allIssues[i].Number < allIssues[j].Number
	})

	return allIssues, nil
}

// Get retrieves a specific issue by project alias and number
func (ms *MultiStore) Get(alias string, number int) (*ProjectIssue, error) {
	proj, ok := ms.projects[alias]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", alias)
	}

	iss, err := proj.Store.Get(number)
	if err != nil {
		return nil, err
	}

	return NewProjectIssue(iss, alias), nil
}

// GetByRef retrieves an issue by project reference
func (ms *MultiStore) GetByRef(ref *ProjectRef) (*ProjectIssue, error) {
	return ms.Get(ref.Project, ref.Number)
}

// FindByNumber searches for an issue by number across all projects
// Returns all matching issues (there may be collisions)
func (ms *MultiStore) FindByNumber(number int) []*ProjectIssue {
	var results []*ProjectIssue

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		if iss, err := proj.Store.Get(number); err == nil {
			results = append(results, NewProjectIssue(iss, alias))
		}
	}

	return results
}

// Move changes the state of an issue
func (ms *MultiStore) Move(alias string, number int, newState issue.State) error {
	proj, ok := ms.projects[alias]
	if !ok {
		return fmt.Errorf("project not found: %s", alias)
	}

	return proj.Store.Move(number, newState)
}

// MoveByRef changes the state of an issue by reference
func (ms *MultiStore) MoveByRef(ref *ProjectRef, newState issue.State) error {
	return ms.Move(ref.Project, ref.Number, newState)
}

// Stats returns aggregated statistics from all projects
type MultiStats struct {
	Total     int
	ByProject map[string]*issue.Stats
}

// Stats returns statistics for all projects
func (ms *MultiStore) Stats() (*MultiStats, error) {
	stats := &MultiStats{
		ByProject: make(map[string]*issue.Stats),
	}

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		projStats, err := proj.Store.Stats()
		if err != nil {
			return nil, fmt.Errorf("failed to get stats from %s: %w", alias, err)
		}
		stats.ByProject[alias] = projStats
		stats.Total += projStats.Total
	}

	return stats, nil
}

// Warnings returns all parse failures from all projects
type ProjectWarning struct {
	Project string
	issue.ParseFailure
}

// Warnings returns all warnings from all projects
func (ms *MultiStore) Warnings() []ProjectWarning {
	var warnings []ProjectWarning

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		for _, w := range proj.Store.Warnings() {
			warnings = append(warnings, ProjectWarning{
				Project:      alias,
				ParseFailure: w,
			})
		}
	}

	return warnings
}

// FilterByLabel returns issues with a specific label from all projects
func (ms *MultiStore) FilterByLabel(label string, states ...issue.State) ([]*ProjectIssue, error) {
	var results []*ProjectIssue

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		issues, err := proj.Store.FilterByLabel(label, states...)
		if err != nil {
			return nil, fmt.Errorf("failed to filter by label from %s: %w", alias, err)
		}
		for _, iss := range issues {
			results = append(results, NewProjectIssue(iss, alias))
		}
	}

	return results, nil
}

// FilterByAssignee returns issues assigned to a specific person from all projects
func (ms *MultiStore) FilterByAssignee(assignee string, states ...issue.State) ([]*ProjectIssue, error) {
	var results []*ProjectIssue

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		issues, err := proj.Store.FilterByAssignee(assignee, states...)
		if err != nil {
			return nil, fmt.Errorf("failed to filter by assignee from %s: %w", alias, err)
		}
		for _, iss := range issues {
			results = append(results, NewProjectIssue(iss, alias))
		}
	}

	return results, nil
}

// Search searches for issues matching a keyword across all projects
func (ms *MultiStore) Search(keyword string, titleOnly bool) ([]*ProjectIssue, error) {
	var results []*ProjectIssue

	for _, alias := range ms.order {
		proj := ms.projects[alias]
		issues, err := proj.Store.Search(keyword, titleOnly)
		if err != nil {
			return nil, fmt.Errorf("failed to search in %s: %w", alias, err)
		}
		for _, iss := range issues {
			results = append(results, NewProjectIssue(iss, alias))
		}
	}

	return results, nil
}
