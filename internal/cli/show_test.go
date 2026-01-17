package cli

import (
	"os"
	"strings"
	"testing"
)

func TestRenderMarkdownNoConsecutiveNewlines(t *testing.T) {
	// Read test markdown file with all elements
	content, err := os.ReadFile("testdata/all_elements.md")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	rendered, err := renderMarkdown(string(content))
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	// Check for consecutive newlines (blank lines)
	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Rendered output contains consecutive newlines (blank lines)")
		// Find and report the location
		lines := strings.Split(rendered, "\n")
		for i, line := range lines {
			if i > 0 && strings.TrimSpace(lines[i-1]) == "" && strings.TrimSpace(line) == "" {
				t.Errorf("Blank line found at line %d", i)
			}
		}
	}
}

func TestRenderMarkdownHeadings(t *testing.T) {
	content := `# H1
## H2
### H3`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Headings contain consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownTaskList(t *testing.T) {
	content := `- [x] Task 1
- [ ] Task 2
- [x] Task 3`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Task list contains consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownList(t *testing.T) {
	content := `- Item 1
- Item 2
- Item 3`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("List contains consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	content := "```go\nfunc main() {}\n```"

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Code block contains consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownTable(t *testing.T) {
	content := `| A | B |
|---|---|
| 1 | 2 |`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Table contains consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownBlockQuote(t *testing.T) {
	content := `> Quote line 1
> Quote line 2`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Block quote contains consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownHorizontalRule(t *testing.T) {
	content := `Before

---

After`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Horizontal rule section contains consecutive newlines:\n%s", rendered)
	}
}

func TestRenderMarkdownMixed(t *testing.T) {
	content := `## Section 1
Paragraph text here.
- List item 1
- List item 2
## Section 2
More text.`

	rendered, err := renderMarkdown(content)
	if err != nil {
		t.Fatalf("renderMarkdown failed: %v", err)
	}

	if strings.Contains(rendered, "\n\n") {
		t.Errorf("Mixed content contains consecutive newlines:\n%s", rendered)
	}
}
