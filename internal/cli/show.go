package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/fsnotify/fsnotify"
	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/web"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:               "show <number>",
	Aliases:           []string{"s"},
	Short:             "Show issue details",
	Long:              `Show detailed information about a specific issue.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueNumber,
	RunE:              runShow,
}

var (
	showRaw    bool
	showRefs   bool
	showWatch  bool
	showNotify bool
	showWeb    bool
	showPort   int
)

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Show raw markdown content")
	showCmd.Flags().BoolVar(&showRefs, "refs", false, "Show referenced issues graph")
	showCmd.Flags().BoolVarP(&showWatch, "watch", "w", false, "Watch for file changes (like tail -f)")
	showCmd.Flags().BoolVar(&showNotify, "notify", false, "Send system notification when state changes to done (requires -w)")
	showCmd.Flags().BoolVar(&showWeb, "web", false, "Open issue in web browser")
	showCmd.Flags().IntVar(&showPort, "port", 18080, "Port for web server (used with --web)")
}

func runShow(cmd *cobra.Command, args []string) error {
	number, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}
	store := issue.NewStore(dir)

	iss, err := store.Get(number)
	if err != nil {
		return err
	}

	if showWeb {
		return showIssueInBrowser(store, dir, iss.Number)
	}

	if showWatch {
		return watchIssue(store, iss)
	}

	return displayIssue(store, iss)
}

func showIssueInBrowser(store *issue.Store, dir string, number int) error {
	server := web.NewServer(store, dir, showPort)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down server...")
		cancel()
	}()

	fmt.Printf("Starting Zap web server on http://localhost:%d\n", showPort)
	fmt.Println("Press Ctrl+C to stop")

	return server.StartAndOpen(ctx, fmt.Sprintf("/issues/%d/view", number))
}

func displayIssue(store *issue.Store, iss *issue.Issue) error {
	if showRaw {
		printRawIssue(iss)
	} else {
		printIssueDetail(iss)
	}

	if showRefs {
		printRefsGraph(store, iss.Number)
	}

	return nil
}

func watchIssue(store *issue.Store, iss *issue.Issue) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(iss.FilePath); err != nil {
		return fmt.Errorf("failed to watch file: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	clearScreen()
	if err := displayIssue(store, iss); err != nil {
		return err
	}
	printWatchHint()

	debounce := time.NewTimer(0)
	debounce.Stop()
	defer debounce.Stop()

	prevState := iss.State

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				debounce.Reset(50 * time.Millisecond)
			}

			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				time.Sleep(100 * time.Millisecond)
				if _, err := os.Stat(iss.FilePath); err == nil {
					watcher.Add(iss.FilePath)
				} else {
					fmt.Println("\nFile was removed. Stopping watch.")
					return nil
				}
			}

		case <-debounce.C:
			updated, err := issue.Parse(iss.FilePath)
			if err != nil {
				continue
			}

			clearScreen()
			if err := displayIssue(store, updated); err != nil {
				fmt.Fprintf(os.Stderr, "Error displaying issue: %v\n", err)
			}

			if prevState != issue.StateDone && updated.State == issue.StateDone {
				notifyDone(updated)
			}
			prevState = updated.State

			printWatchHint()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)

		case <-sigChan:
			fmt.Println("\nStopping watch...")
			return nil
		}
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func printWatchHint() {
	fmt.Println()
	fmt.Println(colorize("Watching for changes... (Ctrl+C to exit)", colorGray))
}

func notifyDone(iss *issue.Issue) {
	// Terminal bell
	fmt.Print("\a")

	// Visual notification
	fmt.Println()
	fmt.Println(colorize("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", colorGreen))
	fmt.Println(colorize(fmt.Sprintf("✓ Issue #%d marked as done!", iss.Number), colorGreen))
	fmt.Println(colorize("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", colorGreen))

	// System notification (if --notify flag is set)
	if showNotify {
		sendSystemNotification(
			"Issue Completed",
			fmt.Sprintf("#%d: %s", iss.Number, iss.Title),
		)
	}
}

func sendSystemNotification(title, message string) {
	if runtime.GOOS != "darwin" {
		return
	}

	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	exec.Command("osascript", "-e", script).Run()
}

func printIssueDetail(iss *issue.Issue) {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Issue #%d: %s\n", iss.Number, iss.Title)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("State:    %s\n", iss.State)

	if len(iss.Labels) > 0 {
		fmt.Printf("Labels:   %s\n", strings.Join(iss.Labels, ", "))
	}

	if len(iss.Assignees) > 0 {
		fmt.Printf("Assignee: %s\n", strings.Join(iss.Assignees, ", "))
	}

	fmt.Printf("Created:  %s\n", iss.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Updated:  %s\n", iss.UpdatedAt.Format("2006-01-02 15:04"))

	if iss.ClosedAt != nil {
		fmt.Printf("Closed:   %s\n", iss.ClosedAt.Format("2006-01-02 15:04"))
	}

	fmt.Printf("File:     %s\n", iss.FilePath)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if iss.Body != "" {
		rendered, err := renderMarkdown(iss.Body)
		if err != nil {
			fmt.Printf("\n%s\n", iss.Body)
		} else {
			fmt.Print(rendered)
		}
	}
}

func renderMarkdown(content string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
		glamour.WithStylesFromJSONBytes([]byte(compactStyle)),
	)
	if err != nil {
		return "", err
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return "", err
	}
	// 빈 줄 모두 제거
	rendered = removeBlankLines(rendered)
	// glamour는 끝에 개행을 추가하므로 제거
	return strings.TrimSuffix(rendered, "\n"), nil
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

// 모든 마크다운 요소의 여백을 제거한 컴팩트 스타일
// glamour StyleConfig의 모든 여백 관련 요소를 0으로 설정
const compactStyle = `{
	"document": {
		"margin": 0,
		"block_prefix": "",
		"block_suffix": ""
	},
	"block_quote": {
		"margin": 0,
		"indent": 1,
		"indent_token": "│ "
	},
	"paragraph": {
		"margin": 0
	},
	"list": {
		"margin": 0,
		"level_indent": 2
	},
	"heading": {
		"margin": 0,
		"block_suffix": ""
	},
	"h1": {
		"margin": 0,
		"block_suffix": "",
		"prefix": "# "
	},
	"h2": {
		"margin": 0,
		"block_suffix": "",
		"prefix": "## "
	},
	"h3": {
		"margin": 0,
		"block_suffix": "",
		"prefix": "### "
	},
	"h4": {
		"margin": 0,
		"block_suffix": "",
		"prefix": "#### "
	},
	"h5": {
		"margin": 0,
		"block_suffix": "",
		"prefix": "##### "
	},
	"h6": {
		"margin": 0,
		"block_suffix": "",
		"prefix": "###### "
	},
	"hr": {
		"format": "--------"
	},
	"item": {
		"block_prefix": "• "
	},
	"enumeration": {
		"block_prefix": ". "
	},
	"task": {
		"ticked": "[x] ",
		"unticked": "[ ] "
	},
	"code": {
		"margin": 0
	},
	"code_block": {
		"margin": 0
	},
	"table": {
		"margin": 0
	},
	"definition_list": {
		"margin": 0
	},
	"definition_description": {
		"block_prefix": ""
	},
	"html_block": {
		"margin": 0
	},
	"html_span": {
		"margin": 0
	}
}`

func printRawIssue(iss *issue.Issue) {
	data, err := issue.Serialize(iss)
	if err != nil {
		fmt.Printf("Error serializing issue: %v\n", err)
		return
	}
	fmt.Print(string(data))
}

func printRefsGraph(store *issue.Store, issueNum int) {
	graph, err := store.BuildRefGraph()
	if err != nil {
		fmt.Printf("Error building reference graph: %v\n", err)
		return
	}

	tree := graph.BuildTree(issueNum)
	if len(tree) == 0 {
		return
	}

	fmt.Println()
	fmt.Println()
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println("Referenced Issues:")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	printRefTree(tree, "", true)
	fmt.Println()
	fmt.Println(colorize("(→: mentions, ←: mentioned by)", colorGray))
}

func printRefTree(nodes []*issue.TreeNode, prefix string, isRoot bool) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// Determine connector
		var connector string
		if isRoot {
			if isLast {
				connector = "└── "
			} else {
				connector = "├── "
			}
		} else {
			if isLast {
				connector = "└── "
			} else {
				connector = "├── "
			}
		}

		// Direction arrow
		arrow := "→"
		if node.Direction == issue.RefMentionedBy {
			arrow = "←"
		}

		// State tag and color
		stateTag := fmt.Sprintf("[%s]", node.Issue.State)
		color := stateColor(node.Issue.State)

		// Print node with state-based coloring
		issueInfo := fmt.Sprintf("%s #%d %s %s", arrow, node.Issue.Number, node.Issue.Title, stateTag)
		fmt.Printf("%s%s%s\n", prefix, connector, colorize(issueInfo, color))

		// Calculate new prefix for children
		var childPrefix string
		if isRoot {
			if isLast {
				childPrefix = prefix + "    "
			} else {
				childPrefix = prefix + "│   "
			}
		} else {
			if isLast {
				childPrefix = prefix + "    "
			} else {
				childPrefix = prefix + "│   "
			}
		}

		// Print children
		if len(node.Children) > 0 {
			printRefTree(node.Children, childPrefix, false)
		}
	}
}

func stateColor(s issue.State) string {
	switch s {
	case issue.StateInProgress:
		return colorYellow
	case issue.StateDone:
		return colorGreen
	case issue.StateClosed:
		return colorGray
	default:
		return ""
	}
}
