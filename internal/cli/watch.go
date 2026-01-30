package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
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

var (
	watchAll      bool
	watchState    string
	watchLabel    string
	watchAssignee string
	watchNoDate   bool
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().BoolVarP(&watchAll, "all", "a", false, "Show all issues including done and closed")
	watchCmd.Flags().StringVarP(&watchState, "state", "s", "", "Filter by state (open, wip, done, closed)")
	watchCmd.Flags().StringVarP(&watchLabel, "label", "l", "", "Filter by label")
	watchCmd.Flags().StringVar(&watchAssignee, "assignee", "", "Filter by assignee")
	watchCmd.Flags().BoolVar(&watchNoDate, "no-date", false, "Hide updated time from output")
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	renderWatch(dir)

	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Print("\033[H\033[2J")
			fmt.Println("Watch mode exited.")
			return nil

		case <-ticker.C:
			renderWatch(dir)

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				renderWatch(dir)
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	renderMultiProjectWatch(multiStore)

	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Print("\033[H\033[2J")
			fmt.Println("Watch mode exited.")
			return nil

		case <-ticker.C:
			renderMultiProjectWatch(multiStore)

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				renderMultiProjectWatch(multiStore)
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)
		}
	}
}

func renderMultiProjectWatch(multiStore *project.MultiStore) {
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
		printMultiProjectWatchIssueList(projectIssues)
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Last updated: %s\n", colorize(time.Now().Format("15:04:05"), colorGray))
}

func printMultiProjectWatchIssueList(issues []*project.ProjectIssue) {
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
		fmt.Printf("%s %s %s%s%s\n", tag, ref, title, labels, dateSuffix)
	}

	fmt.Printf("\nTotal: %d issues\n", len(issues))
}

func renderWatch(dir string) {
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
		printWatchIssueList(issues, recentClosedDuration)
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

func printWatchIssueList(issues []*issue.Issue, recentClosedDuration time.Duration) {
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

		if recentlyClosed {
			tag := colorizeWithBg(fmt.Sprintf("%-8s", style.tag), style.color, bgGray)
			titlePart := colorizeWithBg(iss.Title, style.titleColor, bgGray)
			labelsPart := colorizeWithBg(labels, "", bgGray)
			datePart := colorizeWithBg(strings.TrimPrefix(dateSuffix, " "), colorGray, bgGray)

			line := fmt.Sprintf("%s #%-4d %s", tag, iss.Number, titlePart)
			if labels != "" {
				line += " " + labelsPart
			}
			if dateSuffix != "" {
				line += " " + datePart
			}
			fmt.Println(line)
		} else {
			title := colorize(iss.Title, style.titleColor)
			tag := colorize(fmt.Sprintf("%-8s", style.tag), style.color)
			fmt.Printf("%s #%-4d %s%s%s\n", tag, iss.Number, title, labels, dateSuffix)
		}
	}

	fmt.Printf("\nTotal: %d issues\n", len(issues))
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
