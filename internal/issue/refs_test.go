package issue

import (
	"reflect"
	"testing"
)

func TestExtractRefs(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []int
	}{
		{
			name:     "basic reference",
			text:     "See #1 for details",
			expected: []int{1},
		},
		{
			name:     "multiple references",
			text:     "See #1 and #2 for details",
			expected: []int{1, 2},
		},
		{
			name:     "duplicate references",
			text:     "See #1 and #1 again, also #2",
			expected: []int{1, 2},
		},
		{
			name:     "no references",
			text:     "This has no references",
			expected: nil,
		},
		{
			name:     "reference in middle of word",
			text:     "Issue#123 test",
			expected: []int{123},
		},
		{
			name:     "multiple digits",
			text:     "See #123 and #456",
			expected: []int{123, 456},
		},
		{
			name:     "references sorted",
			text:     "See #5, #3, #1, #4, #2",
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "reference at start",
			text:     "#42 is the answer",
			expected: []int{42},
		},
		{
			name:     "reference at end",
			text:     "Related to #99",
			expected: []int{99},
		},
		{
			name:     "multiline text",
			text:     "Line 1 #1\nLine 2 #2\nLine 3 #3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "zero reference ignored",
			text:     "Issue #0 should be ignored",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRefs(tt.text)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ExtractRefs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefGraph_GetConnectedIssues(t *testing.T) {
	// Create a mock graph
	graph := NewRefGraph()

	// Add mock issues
	for i := 1; i <= 5; i++ {
		graph.Issues[i] = &Issue{Number: i, Title: "Issue " + string(rune('0'+i))}
	}

	// Setup relationships:
	// #1 mentions #2, #3
	// #2 mentions #4
	// #5 mentions #1
	graph.Mentions[1] = []int{2, 3}
	graph.Mentions[2] = []int{4}
	graph.Mentions[5] = []int{1}

	graph.MentionedBy[2] = []int{1}
	graph.MentionedBy[3] = []int{1}
	graph.MentionedBy[4] = []int{2}
	graph.MentionedBy[1] = []int{5}

	t.Run("basic connected issues", func(t *testing.T) {
		connected := graph.GetConnectedIssues(1)

		// Should find: #2 (mentions, d=1), #3 (mentions, d=1), #5 (mentioned_by, d=1), #4 (mentions, d=2)
		if len(connected) != 4 {
			t.Errorf("Expected 4 connected issues, got %d", len(connected))
		}

		// Check first level (distance 1)
		d1 := filterByDistance(connected, 1)
		if len(d1) != 3 {
			t.Errorf("Expected 3 issues at distance 1, got %d", len(d1))
		}

		// Check second level (distance 2)
		d2 := filterByDistance(connected, 2)
		if len(d2) != 1 {
			t.Errorf("Expected 1 issue at distance 2, got %d", len(d2))
		}
	})

	t.Run("non-existent issue", func(t *testing.T) {
		connected := graph.GetConnectedIssues(999)
		if connected != nil {
			t.Errorf("Expected nil for non-existent issue, got %v", connected)
		}
	})

	t.Run("issue with no connections", func(t *testing.T) {
		// Add isolated issue
		graph.Issues[6] = &Issue{Number: 6, Title: "Isolated"}

		connected := graph.GetConnectedIssues(6)
		if len(connected) != 0 {
			t.Errorf("Expected 0 connected issues for isolated node, got %d", len(connected))
		}
	})
}

func TestRefGraph_GetConnectedIssues_Cycle(t *testing.T) {
	// Create a graph with a cycle: #1 -> #2 -> #3 -> #1
	graph := NewRefGraph()

	for i := 1; i <= 3; i++ {
		graph.Issues[i] = &Issue{Number: i}
	}

	graph.Mentions[1] = []int{2}
	graph.Mentions[2] = []int{3}
	graph.Mentions[3] = []int{1}

	graph.MentionedBy[2] = []int{1}
	graph.MentionedBy[3] = []int{2}
	graph.MentionedBy[1] = []int{3}

	connected := graph.GetConnectedIssues(1)

	// Should handle cycle without infinite loop
	// #2 (mentions, d=1), #3 (mentioned_by, d=1 or mentions via #2)
	if len(connected) != 2 {
		t.Errorf("Expected 2 connected issues (cycle handled), got %d", len(connected))
	}
}

func TestRefGraph_GetRefCount(t *testing.T) {
	graph := NewRefGraph()

	graph.Issues[1] = &Issue{Number: 1}
	graph.Issues[2] = &Issue{Number: 2}
	graph.Issues[3] = &Issue{Number: 3}

	// #1 mentions #2, #3
	// #3 mentions #1
	graph.Mentions[1] = []int{2, 3}
	graph.Mentions[3] = []int{1}

	graph.MentionedBy[2] = []int{1}
	graph.MentionedBy[3] = []int{1}
	graph.MentionedBy[1] = []int{3}

	tests := []struct {
		issueNum int
		expected int
	}{
		{1, 3}, // mentions 2, 3 + mentioned by 3
		{2, 1}, // mentioned by 1
		{3, 2}, // mentioned by 1 + mentions 1
	}

	for _, tt := range tests {
		t.Run("issue_"+string(rune('0'+tt.issueNum)), func(t *testing.T) {
			got := graph.GetRefCount(tt.issueNum)
			if got != tt.expected {
				t.Errorf("GetRefCount(%d) = %d, want %d", tt.issueNum, got, tt.expected)
			}
		})
	}
}

func TestRefGraph_BuildTree(t *testing.T) {
	graph := NewRefGraph()

	for i := 1; i <= 4; i++ {
		graph.Issues[i] = &Issue{Number: i, Title: "Issue " + string(rune('0'+i))}
	}

	// #1 -> #2 -> #3
	// #4 -> #1
	graph.Mentions[1] = []int{2}
	graph.Mentions[2] = []int{3}
	graph.Mentions[4] = []int{1}

	graph.MentionedBy[2] = []int{1}
	graph.MentionedBy[3] = []int{2}
	graph.MentionedBy[1] = []int{4}

	tree := graph.BuildTree(1)

	if len(tree) != 2 {
		t.Errorf("Expected 2 root children, got %d", len(tree))
	}

	// Check that tree has proper structure
	var mentionsNode, mentionedByNode *TreeNode
	for _, n := range tree {
		if n.Direction == RefMentions {
			mentionsNode = n
		} else {
			mentionedByNode = n
		}
	}

	if mentionsNode == nil || mentionsNode.Issue.Number != 2 {
		t.Error("Expected mentions node with issue #2")
	}

	if mentionedByNode == nil || mentionedByNode.Issue.Number != 4 {
		t.Error("Expected mentioned_by node with issue #4")
	}

	// Check nested child
	if len(mentionsNode.Children) != 1 || mentionsNode.Children[0].Issue.Number != 3 {
		t.Error("Expected #2 to have child #3")
	}
}

// Helper function
func filterByDistance(connected []ConnectedIssue, distance int) []ConnectedIssue {
	var result []ConnectedIssue
	for _, c := range connected {
		if c.Distance == distance {
			result = append(result, c)
		}
	}
	return result
}
