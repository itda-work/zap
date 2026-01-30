package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/itda-work/zap/internal/ai"
	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch issues in real-time",
	Long:  `Watch issues from the .issues directory in real-time. Updates automatically when files change.`,
	RunE:  runWatch,
}

const (
	// DefaultWatchChangeMinutes is the default duration to show change summaries
	DefaultWatchChangeMinutes = 10
	// EnvWatchChangeMinutes is the environment variable for configuring change summary duration
	EnvWatchChangeMinutes = "ZAP_WATCH_CHANGE_MINUTES"
)

var (
	watchAll      bool
	watchState    string
	watchLabel    string
	watchAssignee string
	watchNoDate   bool
	watchDuration int
	watchAI       bool
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().BoolVarP(&watchAll, "all", "a", false, "Show all issues including done and closed")
	watchCmd.Flags().StringVarP(&watchState, "state", "s", "", "Filter by state (open, wip, done, closed)")
	watchCmd.Flags().StringVarP(&watchLabel, "label", "l", "", "Filter by label")
	watchCmd.Flags().StringVar(&watchAssignee, "assignee", "", "Filter by assignee")
	watchCmd.Flags().BoolVar(&watchNoDate, "no-date", false, "Hide updated time from output")
	watchCmd.Flags().IntVar(&watchDuration, "duration", 0, "Duration in minutes to show change summaries (default: 10, 0=disabled)")
	watchCmd.Flags().BoolVar(&watchAI, "ai", false, "Enable AI-powered change summaries (gemini → claude fallback)")
}

func runWatch(cmd *cobra.Command, args []string) error {
	if isMultiProjectMode(cmd) {
		return runMultiProjectWatch(cmd, args)
	}

	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	var tracker *changeTracker
	if changeDur := getWatchChangeDuration(); changeDur > 0 {
		tracker = newChangeTracker(changeDur)
		store := issue.NewStore(dir)
		if initIssues, err := store.List(issue.AllStates()...); err == nil {
			tracker.takeSnapshot(initIssues)
		}
		if watchAI {
			tracker.renderNotify = make(chan struct{}, 1)
			tracker.initAI()
			if tracker.aiClient != nil {
				fmt.Fprintf(os.Stderr, "AI summary: %s\n", tracker.aiClient.Name())
			}
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	renderWatch(dir, tracker)

	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	var aiNotify <-chan struct{}
	if tracker != nil && tracker.renderNotify != nil {
		aiNotify = tracker.renderNotify
	}

	for {
		select {
		case <-sigChan:
			fmt.Print("\033[H\033[2J")
			fmt.Println("Watch mode exited.")
			return nil

		case <-ticker.C:
			renderWatch(dir, tracker)

		case <-aiNotify:
			renderWatch(dir, tracker)

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			if tracker != nil {
				if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					tracker.processRemoval(event.Name)
				} else {
					tracker.processChange(event.Name)
				}
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				renderWatch(dir, tracker)
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)
		}
	}
}

func runMultiProjectWatch(cmd *cobra.Command, args []string) error {
	multiStore, err := getMultiStore(cmd)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	for _, proj := range multiStore.Projects() {
		dir := proj.Store.BaseDir()
		if err := watcher.Add(dir); err != nil {
			return fmt.Errorf("failed to watch directory %s: %w", dir, err)
		}
	}

	var tracker *changeTracker
	if changeDur := getWatchChangeDuration(); changeDur > 0 {
		tracker = newChangeTracker(changeDur)
		if allPIssues, err := multiStore.ListAll(issue.AllStates()...); err == nil {
			initIssues := make([]*issue.Issue, len(allPIssues))
			for i, pi := range allPIssues {
				initIssues[i] = pi.Issue
			}
			tracker.takeSnapshot(initIssues)
		}
		if watchAI {
			tracker.renderNotify = make(chan struct{}, 1)
			tracker.initAI()
			if tracker.aiClient != nil {
				fmt.Fprintf(os.Stderr, "AI summary: %s\n", tracker.aiClient.Name())
			}
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	renderMultiProjectWatch(multiStore, tracker)

	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	var aiNotify <-chan struct{}
	if tracker != nil && tracker.renderNotify != nil {
		aiNotify = tracker.renderNotify
	}

	for {
		select {
		case <-sigChan:
			fmt.Print("\033[H\033[2J")
			fmt.Println("Watch mode exited.")
			return nil

		case <-ticker.C:
			renderMultiProjectWatch(multiStore, tracker)

		case <-aiNotify:
			renderMultiProjectWatch(multiStore, tracker)

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			if tracker != nil {
				if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					tracker.processRemoval(event.Name)
				} else {
					tracker.processChange(event.Name)
				}
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				renderMultiProjectWatch(multiStore, tracker)
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)
		}
	}
}

func renderMultiProjectWatch(multiStore *project.MultiStore, tracker *changeTracker) {
	fmt.Print("\033[H\033[2J")

	fmt.Println(colorize("Issue Monitor", colorCyan) + " " +
		colorize(fmt.Sprintf("(%d projects)", multiStore.ProjectCount()), colorGray) + " " +
		colorize("(Press Ctrl+C to exit)", colorGray))
	fmt.Println(strings.Repeat("─", 60))

	allProjectIssues, err := multiStore.ListAll(issue.AllStates()...)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	allIssues := make([]*issue.Issue, len(allProjectIssues))
	for i, pIss := range allProjectIssues {
		allIssues[i] = pIss.Issue
	}
	stats := calculateStats(allIssues)
	printWatchStats(stats)

	fmt.Println(strings.Repeat("─", 60))

	var states []issue.State
	if watchState != "" {
		state, ok := issue.ParseState(watchState)
		if !ok {
			fmt.Printf("Invalid state: %s\n", watchState)
			return
		}
		states = []issue.State{state}
	} else if watchAll {
		states = issue.AllStates()
	} else {
		states = issue.ActiveStates()
	}

	var projectIssues []*project.ProjectIssue
	if watchLabel != "" {
		projectIssues, err = multiStore.FilterByLabel(watchLabel, states...)
	} else if watchAssignee != "" {
		projectIssues, err = multiStore.FilterByAssignee(watchAssignee, states...)
	} else {
		projectIssues, err = multiStore.ListAll(states...)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(projectIssues) == 0 {
		fmt.Println(colorize("No active issues.", colorGray))
	} else {
		sortProjectIssuesByStateAndTime(projectIssues)
		printMultiProjectWatchIssueList(projectIssues, tracker)
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Last updated: %s\n", colorize(time.Now().Format("15:04:05"), colorGray))
}

func printMultiProjectWatchIssueList(issues []*project.ProjectIssue, tracker *changeTracker) {
	stateStyle := map[issue.State]struct {
		tag        string
		color      string
		titleColor string
	}{
		issue.StateOpen:   {"[open]", "", ""},
		issue.StateWip:    {"[wip]", colorBrightYellow, colorBrightYellow},
		issue.StateDone:   {"[done]", colorBrightGreen, colorBrightGreen},
		issue.StateClosed: {"[closed]", colorGray, colorLightGray},
	}

	var activeChanges map[string]*changeEntry
	if tracker != nil {
		activeChanges = tracker.getActiveChanges()
	}

	termWidth := getTerminalWidth()

	for _, pIss := range issues {
		style := stateStyle[pIss.State]
		labels := ""
		if len(pIss.Labels) > 0 {
			labels = fmt.Sprintf(" [%s]", strings.Join(pIss.Labels, ", "))
		}

		dateSuffix := ""
		if !watchNoDate {
			dateSuffix = fmt.Sprintf(" %s", colorize(formatRelativeTime(pIss.UpdatedAt), colorGray))
		}

		title := colorize(pIss.Title, style.titleColor)
		tag := colorize(fmt.Sprintf("%-8s", style.tag), style.color)
		ref := colorize(fmt.Sprintf("%-12s", pIss.Ref()), colorCyan)
		line := fmt.Sprintf("%s %s %s%s%s", tag, ref, title, labels, dateSuffix)
		fmt.Println(truncateLine(line, termWidth))

		if entry, ok := activeChanges[pIss.FilePath]; ok {
			changeLine := fmt.Sprintf("                      %s %s", colorize("↳", colorCyan), colorize(entry.summary, colorGray))
			fmt.Println(truncateLine(changeLine, termWidth))
			if entry.aiLoading {
				aiLine := fmt.Sprintf("                      %s %s", colorize("↳", colorCyan), colorize("Loading ...", colorGray))
				fmt.Println(truncateLine(aiLine, termWidth))
			} else if entry.aiSummary != "" {
				aiLine := fmt.Sprintf("                      %s %s", colorize("↳", colorCyan), colorize(entry.aiSummary, colorMagenta))
				fmt.Println(truncateLine(aiLine, termWidth))
			}
		}
	}

	fmt.Printf("\nTotal: %d issues\n", len(issues))
}

func renderWatch(dir string, tracker *changeTracker) {
	fmt.Print("\033[H\033[2J")

	fmt.Println(colorize("Issue Monitor", colorCyan) + " " + colorize("(Press Ctrl+C to exit)", colorGray))
	fmt.Println(strings.Repeat("─", 60))

	store := issue.NewStore(dir)

	allIssues, err := store.List(issue.AllStates()...)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	stats := calculateStats(allIssues)
	printWatchStats(stats)

	fmt.Println(strings.Repeat("─", 60))

	var states []issue.State
	if watchState != "" {
		state, ok := issue.ParseState(watchState)
		if !ok {
			fmt.Printf("Invalid state: %s\n", watchState)
			return
		}
		states = []issue.State{state}
	} else if watchAll {
		states = issue.AllStates()
	} else {
		states = issue.ActiveStates()
	}

	var issues []*issue.Issue
	if watchLabel != "" {
		issues, err = store.FilterByLabel(watchLabel, states...)
	} else if watchAssignee != "" {
		issues, err = store.FilterByAssignee(watchAssignee, states...)
	} else {
		issues, err = store.List(states...)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	recentClosedDuration := getRecentClosedDuration()
	if !watchAll && watchState == "" && recentClosedDuration > 0 {
		recentIssues, err := getRecentlyClosedIssuesForWatch(store, recentClosedDuration, watchLabel, watchAssignee)
		if err == nil && len(recentIssues) > 0 {
			issues = mergeIssues(issues, recentIssues)
		}
	}

	if len(issues) == 0 {
		fmt.Println(colorize("No active issues.", colorGray))
	} else {
		sortIssuesByStateAndTime(issues)
		printWatchIssueList(issues, recentClosedDuration, tracker)
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Last updated: %s\n", colorize(time.Now().Format("15:04:05"), colorGray))
}

func printWatchStats(stats *issue.Stats) {
	parts := []string{
		fmt.Sprintf("%s: %s", "Open", colorize(fmt.Sprintf("%d", stats.ByState[issue.StateOpen]), "")),
		fmt.Sprintf("%s: %s", colorize("WIP", colorBrightYellow), colorize(fmt.Sprintf("%d", stats.ByState[issue.StateWip]), colorBrightYellow)),
		fmt.Sprintf("%s: %s", colorize("Done", colorBrightGreen), colorize(fmt.Sprintf("%d", stats.ByState[issue.StateDone]), colorBrightGreen)),
		fmt.Sprintf("%s: %s", colorize("Closed", colorGray), colorize(fmt.Sprintf("%d", stats.ByState[issue.StateClosed]), colorGray)),
	}
	fmt.Println(strings.Join(parts, " | "))
}

func printWatchIssueList(issues []*issue.Issue, recentClosedDuration time.Duration, tracker *changeTracker) {
	stateStyle := map[issue.State]struct {
		tag        string
		color      string
		titleColor string
	}{
		issue.StateOpen:   {"[open]", "", ""},
		issue.StateWip:    {"[wip]", colorBrightYellow, colorBrightYellow},
		issue.StateDone:   {"[done]", colorBrightGreen, colorBrightGreen},
		issue.StateClosed: {"[closed]", colorGray, colorLightGray},
	}

	var activeChanges map[string]*changeEntry
	if tracker != nil {
		activeChanges = tracker.getActiveChanges()
	}

	termWidth := getTerminalWidth()

	for _, iss := range issues {
		style := stateStyle[iss.State]
		labels := ""
		if len(iss.Labels) > 0 {
			labels = fmt.Sprintf(" [%s]", strings.Join(iss.Labels, ", "))
		}

		dateSuffix := ""
		if !watchNoDate {
			dateSuffix = fmt.Sprintf(" %s", colorize(formatRelativeTime(iss.UpdatedAt), colorGray))
		}

		recentlyClosed := isRecentlyClosed(iss.UpdatedAt, string(iss.State), recentClosedDuration)

		var line string
		if recentlyClosed {
			tag := colorizeWithBg(fmt.Sprintf("%-8s", style.tag), style.color, bgGray)
			titlePart := colorizeWithBg(iss.Title, style.titleColor, bgGray)
			labelsPart := colorizeWithBg(labels, "", bgGray)
			datePart := colorizeWithBg(strings.TrimPrefix(dateSuffix, " "), colorGray, bgGray)

			line = fmt.Sprintf("%s #%-4d %s", tag, iss.Number, titlePart)
			if labels != "" {
				line += " " + labelsPart
			}
			if dateSuffix != "" {
				line += " " + datePart
			}
		} else {
			title := colorize(iss.Title, style.titleColor)
			tag := colorize(fmt.Sprintf("%-8s", style.tag), style.color)
			line = fmt.Sprintf("%s #%-4d %s%s%s", tag, iss.Number, title, labels, dateSuffix)
		}
		fmt.Println(truncateLine(line, termWidth))

		if entry, ok := activeChanges[iss.FilePath]; ok {
			changeLine := fmt.Sprintf("         %s %s", colorize("↳", colorCyan), colorize(entry.summary, colorGray))
			fmt.Println(truncateLine(changeLine, termWidth))
			if entry.aiLoading {
				aiLine := fmt.Sprintf("         %s %s", colorize("↳", colorCyan), colorize("Loading ...", colorGray))
				fmt.Println(truncateLine(aiLine, termWidth))
			} else if entry.aiSummary != "" {
				aiLine := fmt.Sprintf("         %s %s", colorize("↳", colorCyan), colorize(entry.aiSummary, colorMagenta))
				fmt.Println(truncateLine(aiLine, termWidth))
			}
		}
	}

	fmt.Printf("\nTotal: %d issues\n", len(issues))
}

type changeEntry struct {
	timestamp   time.Time
	filePath    string
	issueNumber int
	summary     string
	aiSummary   string
	aiLoading   bool
}

type changeTracker struct {
	mu             sync.RWMutex
	snapshots      map[string]*issue.Issue
	changes        map[string]*changeEntry
	expiryDuration time.Duration
	aiClient       ai.Client
	renderNotify   chan struct{}
}

func newChangeTracker(expiryDuration time.Duration) *changeTracker {
	return &changeTracker{
		snapshots:      make(map[string]*issue.Issue),
		changes:        make(map[string]*changeEntry),
		expiryDuration: expiryDuration,
	}
}

func (ct *changeTracker) initAI() {
	cfg, err := ai.LoadConfig()
	if err != nil {
		return
	}

	gemini := ai.NewClient(ai.ProviderGemini, cfg)
	if gemini != nil && gemini.IsAvailable() {
		ct.aiClient = gemini
		return
	}

	claude := ai.NewClient(ai.ProviderClaude, cfg)
	if claude != nil && claude.IsAvailable() {
		ct.aiClient = claude
	}
}

func (ct *changeTracker) takeSnapshot(issues []*issue.Issue) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	for _, iss := range issues {
		ct.snapshots[iss.FilePath] = iss
	}
}

func (ct *changeTracker) processChange(filePath string) {
	newIssue, err := issue.Parse(filePath)
	if err != nil {
		return
	}

	ct.mu.Lock()

	old, exists := ct.snapshots[filePath]
	if !exists {
		entry := &changeEntry{
			timestamp:   time.Now(),
			filePath:    filePath,
			issueNumber: newIssue.Number,
			summary:     "new issue created",
			aiLoading:   ct.aiClient != nil,
		}
		ct.changes[filePath] = entry
		ct.snapshots[filePath] = newIssue
		ct.mu.Unlock()

		if ct.aiClient != nil {
			go ct.fetchAISummary(filePath, nil, newIssue)
		}
		return
	}

	summary := generateChangeSummary(old, newIssue)
	if summary != "" {
		entry := &changeEntry{
			timestamp:   time.Now(),
			filePath:    filePath,
			issueNumber: newIssue.Number,
			summary:     summary,
			aiLoading:   ct.aiClient != nil,
		}
		ct.changes[filePath] = entry
		oldCopy := *old
		ct.snapshots[filePath] = newIssue
		ct.mu.Unlock()

		if ct.aiClient != nil {
			go ct.fetchAISummary(filePath, &oldCopy, newIssue)
		}
		return
	}
	ct.snapshots[filePath] = newIssue
	ct.mu.Unlock()
}

func (ct *changeTracker) fetchAISummary(filePath string, old, new *issue.Issue) {
	prompt := buildAIPrompt(old, new)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := ct.aiClient.Complete(ctx, &ai.Request{
		Prompt:    prompt,
		MaxTokens: 200,
	})

	ct.mu.Lock()
	defer ct.mu.Unlock()

	entry, ok := ct.changes[filePath]
	if !ok {
		return
	}
	entry.aiLoading = false

	if err == nil && resp.Content != "" {
		entry.aiSummary = resp.Content
	}

	if ct.renderNotify != nil {
		select {
		case ct.renderNotify <- struct{}{}:
		default:
		}
	}
}

func buildAIPrompt(old, new *issue.Issue) string {
	var sb strings.Builder
	sb.WriteString("다음 이슈 변경 사항을 한 줄(최대 80자)로 간결하게 한국어로 요약해주세요. 설명 없이 요약만 출력하세요.\n\n")

	if old == nil {
		sb.WriteString(fmt.Sprintf("새 이슈 생성: #%d %s\n", new.Number, new.Title))
		if new.Body != "" {
			body := new.Body
			if len(body) > 500 {
				body = body[:500] + "..."
			}
			sb.WriteString("\n본문:\n")
			sb.WriteString(body)
		}
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("이슈: #%d %s\n", new.Number, new.Title))

	if old.State != new.State {
		sb.WriteString(fmt.Sprintf("상태: %s → %s\n", old.State, new.State))
	}
	if old.Title != new.Title {
		sb.WriteString(fmt.Sprintf("제목: \"%s\" → \"%s\"\n", old.Title, new.Title))
	}
	if labelDiff := diffStringSlice(old.Labels, new.Labels); labelDiff != "" {
		sb.WriteString(fmt.Sprintf("레이블: %s\n", labelDiff))
	}
	if assigneeDiff := diffStringSlice(old.Assignees, new.Assignees); assigneeDiff != "" {
		sb.WriteString(fmt.Sprintf("담당자: %s\n", assigneeDiff))
	}

	if old.Body != new.Body {
		oldBody := old.Body
		if len(oldBody) > 300 {
			oldBody = oldBody[:300] + "..."
		}
		newBody := new.Body
		if len(newBody) > 300 {
			newBody = newBody[:300] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n이전 본문:\n%s\n\n변경된 본문:\n%s", oldBody, newBody))
	}

	return sb.String()
}

func (ct *changeTracker) processRemoval(filePath string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	delete(ct.snapshots, filePath)
	delete(ct.changes, filePath)
}

func (ct *changeTracker) getActiveChanges() map[string]*changeEntry {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	active := make(map[string]*changeEntry)
	for k, entry := range ct.changes {
		if now.Sub(entry.timestamp) <= ct.expiryDuration {
			active[k] = entry
		} else {
			delete(ct.changes, k)
		}
	}
	return active
}

func generateChangeSummary(old, new *issue.Issue) string {
	var parts []string

	if old.State != new.State {
		parts = append(parts, fmt.Sprintf("state: %s → %s", old.State, new.State))
	}
	if old.Title != new.Title {
		parts = append(parts, fmt.Sprintf("title: \"%s\" → \"%s\"", old.Title, new.Title))
	}

	if labelDiff := diffStringSlice(old.Labels, new.Labels); labelDiff != "" {
		parts = append(parts, "labels: "+labelDiff)
	}
	if assigneeDiff := diffStringSlice(old.Assignees, new.Assignees); assigneeDiff != "" {
		parts = append(parts, "assignees: "+assigneeDiff)
	}

	if old.Body != new.Body {
		parts = append(parts, "body updated")
	}

	return strings.Join(parts, ", ")
}

func diffStringSlice(old, new []string) string {
	oldSet := make(map[string]bool)
	for _, s := range old {
		oldSet[s] = true
	}
	newSet := make(map[string]bool)
	for _, s := range new {
		newSet[s] = true
	}

	var diffs []string
	added := make([]string, 0)
	for _, s := range new {
		if !oldSet[s] {
			added = append(added, s)
		}
	}
	sort.Strings(added)
	for _, s := range added {
		diffs = append(diffs, "+"+s)
	}
	removed := make([]string, 0)
	for _, s := range old {
		if !newSet[s] {
			removed = append(removed, s)
		}
	}
	sort.Strings(removed)
	for _, s := range removed {
		diffs = append(diffs, "-"+s)
	}

	return strings.Join(diffs, " ")
}

func getWatchChangeDuration() time.Duration {
	if watchDuration > 0 {
		return time.Duration(watchDuration) * time.Minute
	}
	if val := os.Getenv(EnvWatchChangeMinutes); val != "" {
		if minutes, err := strconv.Atoi(val); err == nil && minutes >= 0 {
			if minutes == 0 {
				return 0
			}
			return time.Duration(minutes) * time.Minute
		}
	}
	return DefaultWatchChangeMinutes * time.Minute
}

func getRecentlyClosedIssuesForWatch(store *issue.Store, duration time.Duration, labelFilter, assigneeFilter string) ([]*issue.Issue, error) {
	closedStates := []issue.State{issue.StateDone, issue.StateClosed}

	var issues []*issue.Issue
	var err error

	if labelFilter != "" {
		issues, err = store.FilterByLabel(labelFilter, closedStates...)
	} else if assigneeFilter != "" {
		issues, err = store.FilterByAssignee(assigneeFilter, closedStates...)
	} else {
		issues, err = store.List(closedStates...)
	}

	if err != nil {
		return nil, err
	}

	var recentIssues []*issue.Issue
	for _, iss := range issues {
		if isRecentlyClosed(iss.UpdatedAt, string(iss.State), duration) {
			recentIssues = append(recentIssues, iss)
		}
	}

	return recentIssues, nil
}
