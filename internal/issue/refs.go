package issue

import (
	"regexp"
	"sort"
)

var refPattern = regexp.MustCompile(`#(\d+)`)

// ExtractRefs extracts issue references (#N) from text.
// Returns unique issue numbers in ascending order.
func ExtractRefs(text string) []int {
	matches := refPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[int]bool)
	var refs []int

	for _, match := range matches {
		var num int
		// match[1] is the captured group (digits only)
		for _, c := range match[1] {
			num = num*10 + int(c-'0')
		}
		if num > 0 && !seen[num] {
			seen[num] = true
			refs = append(refs, num)
		}
	}

	sort.Ints(refs)
	return refs
}

// RefGraph represents the reference relationships between issues.
type RefGraph struct {
	// Mentions maps issue number -> issue numbers it mentions
	Mentions map[int][]int
	// MentionedBy maps issue number -> issue numbers that mention it
	MentionedBy map[int][]int
	// Issues maps issue number -> issue (for quick lookup)
	Issues map[int]*Issue
}

// NewRefGraph creates an empty RefGraph.
func NewRefGraph() *RefGraph {
	return &RefGraph{
		Mentions:    make(map[int][]int),
		MentionedBy: make(map[int][]int),
		Issues:      make(map[int]*Issue),
	}
}

// BuildRefGraph builds a reference graph from all issues in the store.
// Only includes references to issues that actually exist.
func (s *Store) BuildRefGraph() (*RefGraph, error) {
	issues, err := s.List()
	if err != nil {
		return nil, err
	}

	graph := NewRefGraph()

	// First pass: index all issues
	for _, iss := range issues {
		graph.Issues[iss.Number] = iss
	}

	// Second pass: extract references
	for _, iss := range issues {
		refs := ExtractRefs(iss.Body)

		for _, ref := range refs {
			// Skip self-references and non-existent issues
			if ref == iss.Number {
				continue
			}
			if _, exists := graph.Issues[ref]; !exists {
				continue
			}

			// Add to mentions
			graph.Mentions[iss.Number] = append(graph.Mentions[iss.Number], ref)

			// Add to mentioned by (reverse relationship)
			graph.MentionedBy[ref] = append(graph.MentionedBy[ref], iss.Number)
		}
	}

	return graph, nil
}

// RefDirection represents the direction of a reference.
type RefDirection string

const (
	RefMentions   RefDirection = "mentions"
	RefMentionedBy RefDirection = "mentioned_by"
)

// ConnectedIssue represents an issue connected through references.
type ConnectedIssue struct {
	Number    int
	Issue     *Issue
	Distance  int          // Distance from the root issue
	Direction RefDirection // How this issue is connected
	Parent    int          // Parent issue number in the tree
}

// GetConnectedIssues returns all issues connected to the given issue number.
// Uses BFS to traverse the graph, handling cycles.
// Results are sorted by distance, then by direction (mentions first), then by number.
func (g *RefGraph) GetConnectedIssues(issueNum int) []ConnectedIssue {
	if _, exists := g.Issues[issueNum]; !exists {
		return nil
	}

	var result []ConnectedIssue
	visited := make(map[int]bool)
	visited[issueNum] = true

	// BFS queue: (issue number, distance, direction, parent)
	type queueItem struct {
		num       int
		distance  int
		direction RefDirection
		parent    int
	}

	queue := []queueItem{}

	// Add direct mentions (issues this issue references)
	for _, ref := range g.Mentions[issueNum] {
		if !visited[ref] {
			queue = append(queue, queueItem{ref, 1, RefMentions, issueNum})
		}
	}

	// Add direct mentioned-by (issues that reference this issue)
	for _, ref := range g.MentionedBy[issueNum] {
		if !visited[ref] {
			queue = append(queue, queueItem{ref, 1, RefMentionedBy, issueNum})
		}
	}

	// BFS traversal
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.num] {
			continue
		}
		visited[item.num] = true

		result = append(result, ConnectedIssue{
			Number:    item.num,
			Issue:     g.Issues[item.num],
			Distance:  item.distance,
			Direction: item.direction,
			Parent:    item.parent,
		})

		// Continue traversing in the same direction
		nextDistance := item.distance + 1

		if item.direction == RefMentions {
			// Follow mentions chain
			for _, ref := range g.Mentions[item.num] {
				if !visited[ref] {
					queue = append(queue, queueItem{ref, nextDistance, RefMentions, item.num})
				}
			}
		} else {
			// Follow mentioned-by chain
			for _, ref := range g.MentionedBy[item.num] {
				if !visited[ref] {
					queue = append(queue, queueItem{ref, nextDistance, RefMentionedBy, item.num})
				}
			}
		}
	}

	// Sort by distance, then direction (mentions first), then number
	sort.Slice(result, func(i, j int) bool {
		if result[i].Distance != result[j].Distance {
			return result[i].Distance < result[j].Distance
		}
		if result[i].Direction != result[j].Direction {
			return result[i].Direction == RefMentions
		}
		return result[i].Number < result[j].Number
	})

	return result
}

// GetRefCount returns the total reference count for an issue.
// Count = issues it mentions + issues that mention it.
func (g *RefGraph) GetRefCount(issueNum int) int {
	return len(g.Mentions[issueNum]) + len(g.MentionedBy[issueNum])
}

// TreeNode represents a node in the reference tree for display.
type TreeNode struct {
	Issue     *Issue
	Direction RefDirection
	Children  []*TreeNode
}

// BuildTree builds a tree structure from connected issues for display.
// This groups issues by their parent relationship.
func (g *RefGraph) BuildTree(issueNum int) []*TreeNode {
	connected := g.GetConnectedIssues(issueNum)
	if len(connected) == 0 {
		return nil
	}

	// Group by parent
	childrenOf := make(map[int][]ConnectedIssue)
	for _, c := range connected {
		childrenOf[c.Parent] = append(childrenOf[c.Parent], c)
	}

	// Build tree recursively
	var buildChildren func(parent int) []*TreeNode
	buildChildren = func(parent int) []*TreeNode {
		children := childrenOf[parent]
		if len(children) == 0 {
			return nil
		}

		nodes := make([]*TreeNode, 0, len(children))
		for _, c := range children {
			node := &TreeNode{
				Issue:     c.Issue,
				Direction: c.Direction,
				Children:  buildChildren(c.Number),
			}
			nodes = append(nodes, node)
		}
		return nodes
	}

	return buildChildren(issueNum)
}
