package ai

import (
	"strings"
	"testing"
)

func TestPromptTemplateRender(t *testing.T) {
	tmpl := &PromptTemplate{
		Name:      "test",
		System:    "You are a {{.role}}.",
		User:      "Help me with {{.task}}.",
		Variables: []string{"role", "task"},
	}

	req, err := tmpl.Render(map[string]string{
		"role": "helpful assistant",
		"task": "coding",
	})

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(req.System, "helpful assistant") {
		t.Errorf("System prompt not rendered correctly: %s", req.System)
	}

	if !strings.Contains(req.Prompt, "coding") {
		t.Errorf("User prompt not rendered correctly: %s", req.Prompt)
	}
}

func TestPromptTemplateRenderMissingVariable(t *testing.T) {
	tmpl := &PromptTemplate{
		Name:      "test",
		User:      "Hello {{.name}}",
		Variables: []string{"name"},
	}

	_, err := tmpl.Render(map[string]string{})

	if err == nil {
		t.Error("Render should fail with missing variable")
	}

	if !strings.Contains(err.Error(), "missing required variable") {
		t.Errorf("Error should mention missing variable: %v", err)
	}
}

func TestBuiltinTemplates(t *testing.T) {
	expectedTemplates := []string{
		"repair-frontmatter",
		"generate-issue",
		"summarize-issue",
	}

	for _, name := range expectedTemplates {
		tmpl, ok := GetTemplate(name)
		if !ok {
			t.Errorf("Template %q not found", name)
			continue
		}
		if tmpl.Name != name {
			t.Errorf("Template name = %q, want %q", tmpl.Name, name)
		}
		if tmpl.User == "" {
			t.Errorf("Template %q has empty user prompt", name)
		}
	}
}

func TestRepairFrontmatterTemplate(t *testing.T) {
	tmpl, ok := GetTemplate("repair-frontmatter")
	if !ok {
		t.Fatal("repair-frontmatter template not found")
	}

	req, err := tmpl.Render(map[string]string{
		"filename": "123-test-issue.md",
		"content":  "some broken content",
	})

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(req.Prompt, "123-test-issue.md") {
		t.Error("Prompt should contain filename")
	}

	if !strings.Contains(req.Prompt, "some broken content") {
		t.Error("Prompt should contain content")
	}
}
