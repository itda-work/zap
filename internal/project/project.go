package project

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/itda-work/zap/internal/issue"
)

// Project represents a single project with its issues
type Project struct {
	Alias string       // "zap", "alfred" (derived from path or explicit)
	Path  string       // Absolute path to project root
	Store *issue.Store // Issue store for this project
}

// NewProject creates a new Project from a spec
func NewProject(spec ProjectSpec, issuesDir string) (*Project, error) {
	absPath, err := filepath.Abs(spec.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	alias := spec.Alias
	if alias == "" {
		alias = filepath.Base(absPath)
	}

	issuesPath := filepath.Join(absPath, issuesDir)
	store := issue.NewStore(issuesPath)

	return &Project{
		Alias: alias,
		Path:  absPath,
		Store: store,
	}, nil
}

// IssuesDir returns the full path to the issues directory
func (p *Project) IssuesDir(issuesDir string) string {
	return filepath.Join(p.Path, issuesDir)
}

// ProjectRef represents a reference to an issue in a specific project
type ProjectRef struct {
	Project string // Project alias (e.g., "zap")
	Number  int    // Issue number
}

// String returns the string representation (e.g., "zap/#1")
func (r ProjectRef) String() string {
	return fmt.Sprintf("%s/#%d", r.Project, r.Number)
}

// ProjectSpec represents a project specification from CLI flags
// Format: "alias:path" or just "path"
type ProjectSpec struct {
	Alias string // Optional alias
	Path  string // Path to project (required)
}

// ParseProjectSpec parses a project specification string
// Formats:
//   - "path" -> ProjectSpec{Path: "path"}
//   - "alias:path" -> ProjectSpec{Alias: "alias", Path: "path"}
func ParseProjectSpec(spec string) ProjectSpec {
	// Check for "alias:path" format
	// But be careful with Windows absolute paths like "C:\path"
	idx := strings.Index(spec, ":")
	if idx > 0 {
		// Check if it looks like a Windows drive letter (single char before colon)
		if idx == 1 && len(spec) > 2 && (spec[2] == '/' || spec[2] == '\\') {
			// Windows absolute path
			return ProjectSpec{Path: spec}
		}
		// It's an alias:path format
		return ProjectSpec{
			Alias: spec[:idx],
			Path:  spec[idx+1:],
		}
	}
	return ProjectSpec{Path: spec}
}

// refPattern matches "project/#number" format
var refPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)/#(\d+)$`)

// ParseRef parses a project issue reference string
// Format: "project/#number" (e.g., "zap/#1", "alfred/#5")
func ParseRef(ref string) (*ProjectRef, error) {
	matches := refPattern.FindStringSubmatch(ref)
	if matches == nil {
		return nil, fmt.Errorf("invalid project reference format: %s (expected: project/#number)", ref)
	}

	number, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid issue number: %s", matches[2])
	}

	return &ProjectRef{
		Project: matches[1],
		Number:  number,
	}, nil
}

// IsProjectRef checks if a string looks like a project reference
func IsProjectRef(s string) bool {
	return refPattern.MatchString(s)
}
