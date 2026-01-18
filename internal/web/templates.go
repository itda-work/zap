package web

import (
	"embed"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/itda-work/zap/internal/issue"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"formatTime":   formatTime,
		"formatDate":   formatDate,
		"stateClass":   stateClass,
		"stateIcon":    stateIcon,
		"stateCount":   stateCount,
		"displayState": displayState,
		"renderMarkdown": func(md string) template.HTML {
			html, err := RenderHTML(md)
			if err != nil {
				return template.HTML("<p>Error rendering markdown</p>")
			}
			return template.HTML(html)
		},
		"join": strings.Join,
		"chromaCSS": func() template.CSS {
			return template.CSS(GetChromaCSS())
		},
	}

	templates = template.Must(
		template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html"),
	)
}

// RenderDashboard renders the dashboard page
func RenderDashboard(w io.Writer, data *DashboardData) error {
	return templates.ExecuteTemplate(w, "dashboard.html", data)
}

// RenderIssuePage renders a single issue page
func RenderIssuePage(w io.Writer, data *IssuePageData) error {
	return templates.ExecuteTemplate(w, "issue.html", data)
}

// DashboardData holds data for the dashboard template
type DashboardData struct {
	Issues      []*issue.Issue
	Stats       *issue.Stats
	StateFilter string
}

// IssuePageData holds data for the issue page template
type IssuePageData struct {
	Issue    *issue.Issue
	HTMLBody template.HTML
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func stateClass(s issue.State) string {
	switch s {
	case issue.StateOpen:
		return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
	case issue.StateInProgress:
		return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
	case issue.StateDone:
		return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
	case issue.StateClosed:
		return "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	default:
		return "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	}
}

func stateIcon(s issue.State) string {
	switch s {
	case issue.StateOpen:
		return `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke-width="2"/></svg>`
	case issue.StateInProgress:
		return `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`
	case issue.StateDone:
		return `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`
	case issue.StateClosed:
		return `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`
	default:
		return ""
	}
}

func stateCount(stats *issue.Stats, state string) int {
	if stats == nil || stats.ByState == nil {
		return 0
	}
	s, ok := issue.ParseState(state)
	if !ok {
		return 0
	}
	return stats.ByState[s]
}

func displayState(s issue.State) string {
	if s == issue.StateInProgress {
		return "wip"
	}
	return string(s)
}
