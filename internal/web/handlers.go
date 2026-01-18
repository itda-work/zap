package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
)

// Handler handles HTTP requests
type Handler struct {
	store      *issue.Store          // For single-project mode
	multiStore *project.MultiStore   // For multi-project mode
}

// NewHandler creates a new Handler for single-project mode
func NewHandler(store *issue.Store) *Handler {
	return &Handler{store: store}
}

// NewMultiHandler creates a new Handler for multi-project mode
func NewMultiHandler(multiStore *project.MultiStore) *Handler {
	return &Handler{multiStore: multiStore}
}

// IsMultiProject returns true if this is a multi-project handler
func (h *Handler) IsMultiProject() bool {
	return h.multiStore != nil
}

// Dashboard serves the main dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Parse state filter from query
	stateFilter := r.URL.Query().Get("state")
	projectFilter := r.URL.Query().Get("project")

	var states []issue.State
	if stateFilter != "" {
		state, ok := issue.ParseState(stateFilter)
		if !ok {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
		states = []issue.State{state}
	} else {
		states = issue.ActiveStates()
	}

	// Handle multi-project mode
	if h.IsMultiProject() {
		h.multiProjectDashboard(w, r, states, stateFilter, projectFilter)
		return
	}

	// Single-project mode
	issues, err := h.store.List(states...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := h.store.Stats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := &DashboardData{
		Issues:      issues,
		Stats:       stats,
		StateFilter: stateFilter,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := RenderDashboard(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// multiProjectDashboard handles dashboard for multi-project mode
func (h *Handler) multiProjectDashboard(w http.ResponseWriter, r *http.Request, states []issue.State, stateFilter, projectFilter string) {
	projectIssues, err := h.multiStore.ListAll(states...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply project filter if specified
	if projectFilter != "" {
		var filtered []*project.ProjectIssue
		for _, pIss := range projectIssues {
			if pIss.Project == projectFilter {
				filtered = append(filtered, pIss)
			}
		}
		projectIssues = filtered
	}

	stats, err := h.multiStore.Stats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get list of project names for filter dropdown
	var projectNames []string
	for _, proj := range h.multiStore.Projects() {
		projectNames = append(projectNames, proj.Alias)
	}

	data := &MultiDashboardData{
		Issues:        projectIssues,
		Stats:         stats,
		StateFilter:   stateFilter,
		ProjectFilter: projectFilter,
		Projects:      projectNames,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := RenderMultiDashboard(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ListIssues returns issues as JSON
func (h *Handler) ListIssues(w http.ResponseWriter, r *http.Request) {
	stateFilter := r.URL.Query().Get("state")

	var states []issue.State
	if stateFilter != "" {
		state, ok := issue.ParseState(stateFilter)
		if !ok {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
		states = []issue.State{state}
	}

	issues, err := h.store.List(states...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issues)
}

// IssueResponse is the JSON response for a single issue
type IssueResponse struct {
	*issue.Issue
	HTMLBody string `json:"html_body,omitempty"`
}

// GetIssue returns a single issue as JSON
func (h *Handler) GetIssue(w http.ResponseWriter, r *http.Request) {
	number, err := h.parseIssueNumber(r.URL.Path, "/issues/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	iss, err := h.store.Get(number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	htmlBody, _ := RenderHTML(iss.Body)

	resp := &IssueResponse{
		Issue:    iss,
		HTMLBody: htmlBody,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ViewIssue renders a single issue as HTML
func (h *Handler) ViewIssue(w http.ResponseWriter, r *http.Request) {
	// Extract from path: /issues/:number/view or /issues/:project/:number/view
	path := strings.TrimPrefix(r.URL.Path, "/issues/")
	path = strings.TrimSuffix(path, "/view")

	var iss *issue.Issue

	if h.IsMultiProject() {
		// Multi-project mode: /issues/:project/:number/view
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			http.Error(w, "Invalid path format", http.StatusBadRequest)
			return
		}
		projectAlias := parts[0]
		number, err := strconv.Atoi(parts[1])
		if err != nil {
			http.Error(w, "Invalid issue number", http.StatusBadRequest)
			return
		}
		projIssue, err := h.multiStore.Get(projectAlias, number)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		iss = projIssue.Issue
	} else {
		// Single-project mode: /issues/:number/view
		number, err := strconv.Atoi(path)
		if err != nil {
			http.Error(w, "Invalid issue number", http.StatusBadRequest)
			return
		}
		var err2 error
		iss, err2 = h.store.Get(number)
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusNotFound)
			return
		}
	}

	htmlBody, _ := RenderHTML(iss.Body)

	data := &IssuePageData{
		Issue:    iss,
		HTMLBody: template.HTML(htmlBody),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := RenderIssuePage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) parseIssueNumber(path, prefix string) (int, error) {
	numStr := strings.TrimPrefix(path, prefix)
	// Remove trailing slash if present
	numStr = strings.TrimSuffix(numStr, "/")
	return strconv.Atoi(numStr)
}
