package project

import (
	"fmt"

	"github.com/itda-work/zap/internal/issue"
)

// ProjectIssue wraps an issue with its project context
type ProjectIssue struct {
	*issue.Issue
	Project string // Project alias
}

// Ref returns the project-qualified reference (e.g., "zap/#1")
func (pi *ProjectIssue) Ref() string {
	return fmt.Sprintf("%s/#%d", pi.Project, pi.Number)
}

// ProjectRef returns a ProjectRef for this issue
func (pi *ProjectIssue) ProjectRef() ProjectRef {
	return ProjectRef{
		Project: pi.Project,
		Number:  pi.Number,
	}
}

// NewProjectIssue creates a ProjectIssue from an issue and project alias
func NewProjectIssue(iss *issue.Issue, project string) *ProjectIssue {
	return &ProjectIssue{
		Issue:   iss,
		Project: project,
	}
}
