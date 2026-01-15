package issue

import (
	"time"
)

// State represents the state of an issue
type State string

const (
	StateOpen       State = "open"
	StateInProgress State = "in-progress"
	StateDone       State = "done"
	StateClosed     State = "closed"
)

// AllStates returns all valid states
func AllStates() []State {
	return []State{StateOpen, StateInProgress, StateDone, StateClosed}
}

// ActiveStates returns states considered "active" (not done)
func ActiveStates() []State {
	return []State{StateOpen, StateInProgress}
}

// Issue represents a single issue
type Issue struct {
	Number    int       `yaml:"number"`
	Title     string    `yaml:"title"`
	State     State     `yaml:"state"`
	Labels    []string  `yaml:"labels"`
	Assignees []string  `yaml:"assignees"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
	ClosedAt  *time.Time `yaml:"closed_at,omitempty"`

	// Body contains the markdown content after frontmatter
	Body string `yaml:"-"`

	// FilePath is the path to the issue file
	FilePath string `yaml:"-"`
}

// IsActive returns true if the issue is in an active state
func (i *Issue) IsActive() bool {
	return i.State == StateOpen || i.State == StateInProgress
}

// StateDir returns the directory name for a given state
func StateDir(s State) string {
	return string(s)
}

// ParseState converts a string to State
func ParseState(s string) (State, bool) {
	switch s {
	case "open":
		return StateOpen, true
	case "in-progress":
		return StateInProgress, true
	case "done":
		return StateDone, true
	case "closed":
		return StateClosed, true
	default:
		return "", false
	}
}
