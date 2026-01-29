package ai

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

// PromptTemplate represents a reusable prompt template.
type PromptTemplate struct {
	// Name is the template identifier
	Name string `yaml:"name"`

	// Description describes what this template does
	Description string `yaml:"description"`

	// System is the system prompt template
	System string `yaml:"system"`

	// User is the user prompt template
	User string `yaml:"user"`

	// Variables lists required template variables
	Variables []string `yaml:"variables"`
}

// Render renders the template with the given variables.
func (t *PromptTemplate) Render(vars map[string]string) (*Request, error) {
	// Check required variables
	for _, v := range t.Variables {
		if _, ok := vars[v]; !ok {
			return nil, fmt.Errorf("missing required variable: %s", v)
		}
	}

	system, err := renderTemplate(t.System, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to render system prompt: %w", err)
	}

	user, err := renderTemplate(t.User, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to render user prompt: %w", err)
	}

	return &Request{
		System: system,
		Prompt: user,
	}, nil
}

// renderTemplate renders a single template string with variables.
func renderTemplate(tmplStr string, vars map[string]string) (string, error) {
	if tmplStr == "" {
		return "", nil
	}

	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// builtinTemplates contains the default prompt templates.
var builtinTemplates = map[string]*PromptTemplate{
	"verify-renumber": {
		Name:        "verify-renumber",
		Description: "Verify that renumbering an issue is safe by checking references",
		System: `You are an issue management assistant helping to verify number conflict resolution.
Your task is to analyze whether renumbering an issue is safe by checking if other issues reference it.`,
		User: `We are about to renumber an issue. Please verify this is the correct action.

CONFLICT DETAILS:
- Conflict type: {{.conflict_type}}
- File to renumber: {{.filename}}
- Current number: {{.current_number}}
- New number: {{.new_number}}
- Reason: {{.reason}}

FILE CONTENT:
{{.file_content}}

ALL ISSUES IN PROJECT:
{{.all_issues}}

ANALYSIS REQUIRED:
1. Check if any other issues reference #{{.current_number}}
2. Verify this is the correct file to renumber (should be the later-created one)
3. Confirm the new number {{.new_number}} is not already in use

RESPOND WITH EXACTLY ONE OF:
- "SAFE: <brief explanation>" if renumbering is safe
- "WARNING: <explanation of concern>" if there are potential issues but can proceed
- "UNSAFE: <explanation>" if renumbering would cause problems

Keep your response concise (1-2 sentences).`,
		Variables: []string{"conflict_type", "filename", "current_number", "new_number", "reason", "file_content", "all_issues"},
	},
	"repair-frontmatter": {
		Name:        "repair-frontmatter",
		Description: "Repair malformed YAML frontmatter in issue files",
		System: `You are a YAML frontmatter repair assistant for issue tracking files.
Your task is to fix malformed frontmatter while preserving the original content as much as possible.`,
		User: `Fix the YAML frontmatter in this issue file.
Filename: {{.filename}}

Rules:
- Must start and end with ---
- Required fields: number, title, state, labels, assignees, created_at, updated_at
- Extract number from filename if missing (e.g., "158-feat..." → number: 158)
- state must be one of: open, wip, done, closed
- labels and assignees should be arrays (use [] if empty)
- Dates should be in YYYY-MM-DD format

Current content:
{{.content}}

Return ONLY the corrected file content with no explanation or markdown code blocks.`,
		Variables: []string{"filename", "content"},
	},
	"generate-issue": {
		Name:        "generate-issue",
		Description: "Generate issue content from a brief description",
		System: `You are a technical writer helping to create well-structured issue files.
Create clear, actionable issue content following best practices.`,
		User: `Create an issue file based on this description:
{{.description}}

Type: {{.type}}

Generate a complete issue with:
- Clear, concise title (in format: "type: description")
- Detailed description section
- Implementation plan if applicable
- Acceptance criteria / completion checklist

Return ONLY the complete markdown file content starting with --- frontmatter.`,
		Variables: []string{"description", "type"},
	},
	"summarize-issue": {
		Name:        "summarize-issue",
		Description: "Summarize a long issue into key points",
		System:      `You are a technical summarizer. Extract the key points concisely.`,
		User: `Summarize this issue in 2-3 bullet points:

{{.content}}

Focus on: what needs to be done, why, and key constraints.`,
		Variables: []string{"content"},
	},
}

// Templates is the global template registry.
var Templates = make(map[string]*PromptTemplate)

func init() {
	// Load builtin templates
	for name, tmpl := range builtinTemplates {
		Templates[name] = tmpl
	}
}

// LoadTemplates loads custom templates from a directory.
func LoadTemplates(dir string) error {
	if dir == "" {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, skip
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		tmpl, err := LoadTemplate(path)
		if err != nil {
			return fmt.Errorf("failed to load template %s: %w", entry.Name(), err)
		}

		Templates[tmpl.Name] = tmpl
	}

	return nil
}

// LoadTemplate loads a single template from a file.
func LoadTemplate(path string) (*PromptTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmpl PromptTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	if tmpl.Name == "" {
		// Use filename as name if not specified
		tmpl.Name = filepath.Base(path[:len(path)-5]) // Remove .yaml
	}

	return &tmpl, nil
}

// GetTemplate returns a template by name.
func GetTemplate(name string) (*PromptTemplate, bool) {
	tmpl, ok := Templates[name]
	return tmpl, ok
}
