package project

import (
	"testing"
)

func TestParseProjectSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAlias string
		wantPath  string
	}{
		{
			name:      "path only",
			input:     "/path/to/project",
			wantAlias: "",
			wantPath:  "/path/to/project",
		},
		{
			name:      "alias:path",
			input:     "myproject:/path/to/project",
			wantAlias: "myproject",
			wantPath:  "/path/to/project",
		},
		{
			name:      "relative path",
			input:     "./relative/path",
			wantAlias: "",
			wantPath:  "./relative/path",
		},
		{
			name:      "current directory",
			input:     ".",
			wantAlias: "",
			wantPath:  ".",
		},
		{
			name:      "alias with relative path",
			input:     "main:./src",
			wantAlias: "main",
			wantPath:  "./src",
		},
		{
			name:      "Windows path (should not split)",
			input:     "C:/Users/test/project",
			wantAlias: "",
			wantPath:  "C:/Users/test/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := ParseProjectSpec(tt.input)
			if spec.Alias != tt.wantAlias {
				t.Errorf("ParseProjectSpec(%q).Alias = %q, want %q", tt.input, spec.Alias, tt.wantAlias)
			}
			if spec.Path != tt.wantPath {
				t.Errorf("ParseProjectSpec(%q).Path = %q, want %q", tt.input, spec.Path, tt.wantPath)
			}
		})
	}
}

func TestParseRef(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantProject string
		wantNumber  int
		wantErr     bool
	}{
		{
			name:        "valid ref",
			input:       "zap/#1",
			wantProject: "zap",
			wantNumber:  1,
			wantErr:     false,
		},
		{
			name:        "ref with larger number",
			input:       "alfred/#123",
			wantProject: "alfred",
			wantNumber:  123,
			wantErr:     false,
		},
		{
			name:        "ref with underscores",
			input:       "my_project/#5",
			wantProject: "my_project",
			wantNumber:  5,
			wantErr:     false,
		},
		{
			name:        "ref with hyphens",
			input:       "my-project/#10",
			wantProject: "my-project",
			wantNumber:  10,
			wantErr:     false,
		},
		{
			name:    "invalid - no hash",
			input:   "zap/1",
			wantErr: true,
		},
		{
			name:    "invalid - missing number",
			input:   "zap/#",
			wantErr: true,
		},
		{
			name:    "invalid - just number",
			input:   "#1",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseRef(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRef(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil {
				if ref.Project != tt.wantProject {
					t.Errorf("ParseRef(%q).Project = %q, want %q", tt.input, ref.Project, tt.wantProject)
				}
				if ref.Number != tt.wantNumber {
					t.Errorf("ParseRef(%q).Number = %d, want %d", tt.input, ref.Number, tt.wantNumber)
				}
			}
		})
	}
}

func TestIsProjectRef(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"zap/#1", true},
		{"alfred/#123", true},
		{"my-project/#5", true},
		{"my_project/#5", true},
		{"1", false},
		{"#1", false},
		{"zap/1", false},
		{"zap#1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsProjectRef(tt.input)
			if got != tt.want {
				t.Errorf("IsProjectRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProjectRef_String(t *testing.T) {
	ref := ProjectRef{Project: "zap", Number: 1}
	want := "zap/#1"
	if got := ref.String(); got != want {
		t.Errorf("ProjectRef.String() = %q, want %q", got, want)
	}
}
