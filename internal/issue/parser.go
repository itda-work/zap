package issue

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse reads an issue file and returns an Issue
func Parse(filePath string) (*Issue, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseBytes(data, filePath)
}

// ParseBytes parses issue content from bytes
func ParseBytes(data []byte, filePath string) (*Issue, error) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var issue Issue
	if err := yaml.Unmarshal(frontmatter, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}

	issue.Body = body
	issue.FilePath = filePath

	return &issue, nil
}

// splitFrontmatter splits the frontmatter and body from markdown content
func splitFrontmatter(data []byte) ([]byte, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// First line must be ---
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("empty file")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, "", fmt.Errorf("file must start with ---")
	}

	var frontmatterLines []string
	foundEnd := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundEnd = true
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	if !foundEnd {
		return nil, "", fmt.Errorf("frontmatter not properly closed with ---")
	}

	// Rest is body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	frontmatter := []byte(strings.Join(frontmatterLines, "\n"))
	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return frontmatter, body, nil
}

// Serialize converts an Issue back to markdown format
func Serialize(issue *Issue) ([]byte, error) {
	frontmatter, err := yaml.Marshal(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(frontmatter)
	buf.WriteString("---\n")
	if issue.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(issue.Body)
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}
