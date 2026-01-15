package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/allieus/lim/internal/issue"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// Run starts the TUI application
func Run(issueDir string) error {
	store := issue.NewStore(issueDir)

	issues, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to load issues: %w", err)
	}

	m := newModel(store, issues, issueDir)

	p := tea.NewProgram(m, tea.WithAltScreen())

	// 파일 감시 시작
	go watchIssues(issueDir, p)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// issuesReloadMsg is sent when issues need to be reloaded
type issuesReloadMsg struct{}

// watchIssues watches the issue directory for changes
func watchIssues(issueDir string, p *tea.Program) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	// 모든 상태 디렉토리 감시
	for _, state := range issue.AllStates() {
		dir := filepath.Join(issueDir, issue.StateDir(state))
		_ = watcher.Add(dir)
	}
	_ = watcher.Add(issueDir)

	// 디바운스를 위한 타이머
	var debounceTimer *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// .md 파일 변경만 처리
			if filepath.Ext(event.Name) != ".md" && event.Op != fsnotify.Rename {
				continue
			}

			// 디바운스: 100ms 내 여러 이벤트를 하나로 합침
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
				p.Send(issuesReloadMsg{})
			})

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// View mode
type viewMode int

const (
	listView viewMode = iota
	detailView
)

// Model represents the TUI state
type model struct {
	store       *issue.Store
	issueDir    string
	list        list.Model
	issues      []*issue.Issue
	selected    *issue.Issue
	mode        viewMode
	width       int
	height      int
	filterState string // "" for all, or specific state
}

// issueItem wraps an issue for the list
type issueItem struct {
	issue *issue.Issue
}

func (i issueItem) Title() string {
	stateSymbol := map[issue.State]string{
		issue.StateOpen:       "○",
		issue.StateInProgress: "◐",
		issue.StateDone:       "●",
	}
	return fmt.Sprintf("%s #%d %s", stateSymbol[i.issue.State], i.issue.Number, i.issue.Title)
}

func (i issueItem) Description() string {
	desc := string(i.issue.State)
	if len(i.issue.Labels) > 0 {
		desc += " • " + fmt.Sprintf("%v", i.issue.Labels)
	}
	return desc
}

func (i issueItem) FilterValue() string {
	return i.issue.Title
}

func newModel(store *issue.Store, issues []*issue.Issue, issueDir string) model {
	items := make([]list.Item, len(issues))
	for i, iss := range issues {
		items[i] = issueItem{issue: iss}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("170")).
		BorderLeftForeground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")).
		BorderLeftForeground(lipgloss.Color("170"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Issues"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// 커스텀 키바인딩 추가
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "view"),
			),
			key.NewBinding(
				key.WithKeys("1", "2", "3"),
				key.WithHelp("1-3", "filter state"),
			),
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "refresh"),
			),
		}
	}

	return model{
		store:    store,
		issueDir: issueDir,
		list:     l,
		issues:   issues,
		mode:     listView,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case issuesReloadMsg:
		// 파일 변경 감지 시 이슈 목록 새로고침
		return m.reload(), nil

	case tea.KeyMsg:
		// 상세 뷰에서의 키 처리
		if m.mode == detailView {
			switch msg.String() {
			case "q", "esc", "backspace":
				m.mode = listView
				m.selected = nil
				return m, nil
			case "r":
				return m.reload(), nil
			}
			return m, nil
		}

		// 리스트 뷰에서의 키 처리
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			if item, ok := m.list.SelectedItem().(issueItem); ok {
				m.selected = item.issue
				m.mode = detailView
			}
			return m, nil

		case "r": // 수동 새로고침
			return m.reload(), nil

		case "1": // open만
			return m.filterByState(issue.StateOpen), nil
		case "2": // in-progress만
			return m.filterByState(issue.StateInProgress), nil
		case "3": // done만
			return m.filterByState(issue.StateDone), nil
		case "0", "`": // 전체
			return m.showAll(), nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// reload refreshes the issue list from disk
func (m model) reload() model {
	issues, err := m.store.List()
	if err != nil {
		return m
	}

	m.issues = issues

	// 현재 필터 상태 유지하며 새로고침
	if m.filterState != "" {
		state, ok := issue.ParseState(m.filterState)
		if ok {
			return m.filterByState(state)
		}
	}

	return m.showAll()
}

func (m model) filterByState(state issue.State) model {
	var filtered []*issue.Issue
	for _, iss := range m.issues {
		if iss.State == state {
			filtered = append(filtered, iss)
		}
	}

	items := make([]list.Item, len(filtered))
	for i, iss := range filtered {
		items[i] = issueItem{issue: iss}
	}

	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("Issues [%s]", state)
	m.filterState = string(state)
	return m
}

func (m model) showAll() model {
	items := make([]list.Item, len(m.issues))
	for i, iss := range m.issues {
		items[i] = issueItem{issue: iss}
	}

	m.list.SetItems(items)
	m.list.Title = "Issues"
	m.filterState = ""
	return m
}

func (m model) View() string {
	if m.mode == detailView && m.selected != nil {
		return m.renderDetailView()
	}
	return m.list.View()
}

func (m model) renderDetailView() string {
	iss := m.selected

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	contentStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		Padding(1, 2)

	header := titleStyle.Render(fmt.Sprintf("#%d %s", iss.Number, iss.Title))

	meta := fmt.Sprintf("State: %s\n", iss.State)
	if len(iss.Labels) > 0 {
		meta += fmt.Sprintf("Labels: %v\n", iss.Labels)
	}
	if len(iss.Assignees) > 0 {
		meta += fmt.Sprintf("Assignees: %v\n", iss.Assignees)
	}
	meta += fmt.Sprintf("Created: %s\n", iss.CreatedAt.Format("2006-01-02"))
	meta += fmt.Sprintf("Updated: %s\n", iss.UpdatedAt.Format("2006-01-02"))

	metaRendered := labelStyle.Render(meta)

	body := ""
	if iss.Body != "" {
		// glamour로 마크다운 렌더링
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.width-8),
		)
		if err == nil {
			rendered, err := renderer.Render(iss.Body)
			if err == nil {
				body = removeBlankLines(rendered)
			} else {
				body = "\n" + iss.Body
			}
		} else {
			body = "\n" + iss.Body
		}
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Render("Press q/esc/backspace to go back")

	content := header + "\n" + metaRendered + body + "\n\n" + help

	return contentStyle.Render(content)
}

// removeBlankLines removes all blank lines (lines with only whitespace)
func removeBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var result []string

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
