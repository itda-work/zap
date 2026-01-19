package issue

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DatetimeFormat represents the detected format of a datetime string
type DatetimeFormat string

const (
	FormatRFC3339       DatetimeFormat = "RFC3339"           // 2026-01-17T15:47:00Z
	FormatISO8601       DatetimeFormat = "ISO8601"           // 2026-01-17T15:47:00
	FormatDatetimeSpace DatetimeFormat = "YYYY-MM-DD HH:MM:SS" // 2026-01-17 15:47:00
	FormatDatetimeShort DatetimeFormat = "YYYY-MM-DD HH:MM"    // 2026-01-17 15:47
	FormatDateOnly      DatetimeFormat = "YYYY-MM-DD"          // 2026-01-17
	FormatEmpty         DatetimeFormat = "(empty)"
	FormatUnknown       DatetimeFormat = "(unknown)"
)

// datetimeFormats maps Go time formats to our DatetimeFormat constants
var datetimeFormats = []struct {
	layout string
	format DatetimeFormat
}{
	{time.RFC3339, FormatRFC3339},
	{"2006-01-02T15:04:05", FormatISO8601},
	{"2006-01-02 15:04:05", FormatDatetimeSpace},
	{"2006-01-02 15:04", FormatDatetimeShort},
	{"2006-01-02", FormatDateOnly},
}

// DetectDatetimeFormat detects the format of a datetime string
func DetectDatetimeFormat(s string) DatetimeFormat {
	if s == "" {
		return FormatEmpty
	}

	for _, f := range datetimeFormats {
		if _, err := time.Parse(f.layout, s); err == nil {
			return f.format
		}
	}

	return FormatUnknown
}

// RawDatetimeInfo contains raw datetime strings from an issue file
type RawDatetimeInfo struct {
	Number    int
	Title     string
	FilePath  string
	CreatedAt string
	UpdatedAt string
	ClosedAt  string
}

// GetRawDatetimeInfo extracts raw datetime strings from an issue file
func GetRawDatetimeInfo(filePath string) (*RawDatetimeInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	frontmatter, _, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var raw rawFrontmatter
	if err := yaml.Unmarshal(frontmatter, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}

	return &RawDatetimeInfo{
		Number:    raw.Number,
		Title:     raw.Title,
		FilePath:  filePath,
		CreatedAt: coalesce(raw.CreatedAt, raw.Created),
		UpdatedAt: coalesce(raw.UpdatedAt, raw.Updated),
		ClosedAt:  raw.ClosedAt,
	}, nil
}

// rawFrontmatter is an intermediate struct that supports both field naming conventions
type rawFrontmatter struct {
	Number    int      `yaml:"number"`
	Title     string   `yaml:"title"`
	State     State    `yaml:"state"`
	Labels    []string `yaml:"labels"`
	Assignees []string `yaml:"assignees"`

	// Support both naming conventions
	CreatedAt string `yaml:"created_at"`
	Created   string `yaml:"created"`
	UpdatedAt string `yaml:"updated_at"`
	Updated   string `yaml:"updated"`
	ClosedAt  string `yaml:"closed_at"`
}

// parseFlexibleTime parses time from various formats
func parseFlexibleTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	formats := []string{
		time.RFC3339,           // 2026-01-17T15:47:00Z
		"2006-01-02T15:04:05",  // 2026-01-17T15:47:00
		"2006-01-02 15:04:05",  // 2026-01-17 15:47:00
		"2006-01-02 15:04",     // 2026-01-17 15:47
		"2006-01-02",           // 2026-01-17
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

// coalesce returns the first non-empty string
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

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

	// Parse into intermediate struct that supports both field naming conventions
	var raw rawFrontmatter
	if err := yaml.Unmarshal(frontmatter, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}

	// Convert to Issue struct
	issue := Issue{
		Number:    raw.Number,
		Title:     raw.Title,
		State:     raw.State,
		Labels:    raw.Labels,
		Assignees: raw.Assignees,
		Body:      body,
		FilePath:  filePath,
	}

	// Parse created time (prefer created_at, fallback to created)
	createdStr := coalesce(raw.CreatedAt, raw.Created)
	if createdStr != "" {
		if t, err := parseFlexibleTime(createdStr); err == nil {
			issue.CreatedAt = t
		}
	}

	// Parse updated time (prefer updated_at, fallback to updated)
	updatedStr := coalesce(raw.UpdatedAt, raw.Updated)
	if updatedStr != "" {
		if t, err := parseFlexibleTime(updatedStr); err == nil {
			issue.UpdatedAt = t
		}
	}

	// Parse closed time
	if raw.ClosedAt != "" {
		if t, err := parseFlexibleTime(raw.ClosedAt); err == nil {
			issue.ClosedAt = &t
		}
	}

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

// serializableFrontmatter is used for consistent RFC3339 UTC datetime serialization
type serializableFrontmatter struct {
	Number    int      `yaml:"number"`
	Title     string   `yaml:"title"`
	State     State    `yaml:"state"`
	Labels    []string `yaml:"labels"`
	Assignees []string `yaml:"assignees"`
	CreatedAt string   `yaml:"created_at"`
	UpdatedAt string   `yaml:"updated_at"`
	ClosedAt  string   `yaml:"closed_at,omitempty"`
}

// Serialize converts an Issue back to markdown format
func Serialize(issue *Issue) ([]byte, error) {
	// Convert to serializable format with RFC3339 UTC timestamps
	sf := serializableFrontmatter{
		Number:    issue.Number,
		Title:     issue.Title,
		State:     issue.State,
		Labels:    issue.Labels,
		Assignees: issue.Assignees,
		CreatedAt: issue.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: issue.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if issue.ClosedAt != nil {
		sf.ClosedAt = issue.ClosedAt.UTC().Format(time.RFC3339)
	}

	frontmatter, err := yaml.Marshal(sf)
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
