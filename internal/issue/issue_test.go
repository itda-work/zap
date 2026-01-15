package issue

import "testing"

func TestParseState(t *testing.T) {
	tests := []struct {
		input string
		want  State
		ok    bool
	}{
		{"open", StateOpen, true},
		{"in-progress", StateInProgress, true},
		{"done", StateDone, true},
		{"closed", StateClosed, true},
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseState(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseState(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAllStates(t *testing.T) {
	states := AllStates()
	if len(states) != 4 {
		t.Errorf("AllStates() returned %d states, want 4", len(states))
	}

	expected := []State{StateOpen, StateInProgress, StateDone, StateClosed}
	for i, s := range expected {
		if states[i] != s {
			t.Errorf("AllStates()[%d] = %q, want %q", i, states[i], s)
		}
	}
}

func TestActiveStates(t *testing.T) {
	states := ActiveStates()
	if len(states) != 2 {
		t.Errorf("ActiveStates() returned %d states, want 2", len(states))
	}

	if states[0] != StateOpen || states[1] != StateInProgress {
		t.Errorf("ActiveStates() = %v, want [open, in-progress]", states)
	}
}

func TestStateDir(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateOpen, "open"},
		{StateInProgress, "in-progress"},
		{StateDone, "done"},
		{StateClosed, "closed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := StateDir(tt.state)
			if got != tt.want {
				t.Errorf("StateDir(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestIssueIsActive(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateOpen, true},
		{StateInProgress, true},
		{StateDone, false},
		{StateClosed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			issue := &Issue{State: tt.state}
			got := issue.IsActive()
			if got != tt.want {
				t.Errorf("Issue{State: %q}.IsActive() = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
