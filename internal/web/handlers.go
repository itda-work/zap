package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/itda-work/zap/internal/issue"
)

// Handler handles HTTP requests
type Handler struct {
	store *issue.Store
}

// NewHandler creates a new Handler
func NewHandler(store *issue.Store) *Handler {
	return &Handler{store: store}
}

// Dashboard serves the main dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Parse state filter from query
	stateFilter := r.URL.Query().Get("state")

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
	// Extract number from /issues/:number/view
	path := strings.TrimPrefix(r.URL.Path, "/issues/")
	path = strings.TrimSuffix(path, "/view")
	number, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid issue number", http.StatusBadRequest)
		return
	}

	iss, err := h.store.Get(number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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
