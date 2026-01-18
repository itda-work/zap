package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/itda-work/zap/internal/issue"
	"github.com/itda-work/zap/internal/project"
	"github.com/itda-work/zap/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start web UI server",
	Long: `Start a web server to view issues in a browser.

The server provides:
- Dashboard view of all issues
- Individual issue pages with rendered markdown
- Live reload when issue files change
- Dark mode support

Examples:
  zap serve              # Start server in foreground (default port 18080)
  zap serve -D           # Start server in background (daemon mode)
  zap serve --port 3000  # Start server on custom port
  zap serve stop         # Stop the background server
  zap serve status       # Check if server is running
  zap serve logs         # View server logs (tail -f)`,
	RunE: runServe,
}

var serveStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background server",
	RunE:  runServeStop,
}

var serveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check server status",
	RunE:  runServeStatus,
}

var serveLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View server logs",
	RunE:  runServeLogs,
}

var (
	servePort      int
	serveNoBrowser bool
	serveDaemon    bool
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.AddCommand(serveStopCmd)
	serveCmd.AddCommand(serveStatusCmd)
	serveCmd.AddCommand(serveLogsCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 18080, "Port to run the server on")
	serveCmd.Flags().BoolVar(&serveNoBrowser, "no-browser", false, "Don't open browser automatically")
	serveCmd.Flags().BoolVarP(&serveDaemon, "daemon", "D", false, "Run server in background (daemon mode)")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Check for multi-project mode
	if isMultiProjectMode(cmd) {
		return runMultiProjectServe(cmd)
	}

	// Single project mode
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("issues directory not found: %s\nRun 'zap init' to create it", dir)
	}

	// Check if already running
	status, _ := web.GetDaemonStatus(dir)
	if status != nil && status.Running {
		return fmt.Errorf("server is already running (PID: %d, Port: %d)\nUse 'zap serve stop' to stop it first", status.PID, status.Port)
	}

	if serveDaemon {
		return startDaemon(cmd, dir)
	}

	return startForeground(dir)
}

func runMultiProjectServe(cmd *cobra.Command) error {
	multiStore, err := getMultiStore(cmd)
	if err != nil {
		return err
	}

	// Validate all project directories exist
	for _, proj := range multiStore.Projects() {
		if _, err := os.Stat(proj.Store.BaseDir()); os.IsNotExist(err) {
			return fmt.Errorf("issues directory not found for project %s: %s", proj.Alias, proj.Store.BaseDir())
		}
	}

	// For daemon mode with multi-project, we'd need more complex handling
	// For now, just support foreground mode
	if serveDaemon {
		return fmt.Errorf("daemon mode is not yet supported with multiple projects")
	}

	return startMultiProjectForeground(multiStore)
}

func startMultiProjectForeground(multiStore *project.MultiStore) error {
	server := web.NewMultiServer(multiStore, servePort)

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

	// Print project info
	fmt.Printf("Starting Zap web server on http://localhost:%d\n", servePort)
	fmt.Printf("Projects: ")
	for i, proj := range multiStore.Projects() {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(proj.Alias)
	}
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")

	if serveNoBrowser {
		return server.Start(ctx)
	}
	return server.StartAndOpen(ctx, "/")
}

func startForeground(dir string) error {
	store := issue.NewStore(dir)
	server := web.NewServer(store, dir, servePort)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down server...")
		web.RemovePidFile(dir)
		cancel()
	}()

	// Write PID file for foreground process too (for status check)
	web.WritePidFile(dir, os.Getpid(), servePort)
	defer web.RemovePidFile(dir)

	fmt.Printf("Starting Zap web server on http://localhost:%d\n", servePort)
	fmt.Println("Press Ctrl+C to stop")

	if serveNoBrowser {
		return server.Start(ctx)
	}
	return server.StartAndOpen(ctx, "/")
}

func startDaemon(cmd *cobra.Command, dir string) error {
	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build arguments for the child process
	var args []string

	// Add global flags first
	if d, _ := cmd.Flags().GetString("dir"); d != "" && d != ".issues" {
		args = append(args, "--dir", d)
	}
	if projects, _ := cmd.Flags().GetStringArray("project"); len(projects) > 0 {
		for _, p := range projects {
			args = append(args, "-C", p)
		}
	}

	// Add serve command and its flags
	args = append(args, "serve", "--port", fmt.Sprintf("%d", servePort), "--no-browser")

	// Open log file
	logFile := web.GetLogFilePath(dir)
	log, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Start the process
	process := exec.Command(executable, args...)
	process.Stdout = log
	process.Stderr = log
	setSysProcAttr(process)

	if err := process.Start(); err != nil {
		log.Close()
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	log.Close()

	// Wait a moment for child to start and write PID file
	// The child process writes its own PID file in startForeground()

	fmt.Printf("Server started in background (PID: %d)\n", process.Process.Pid)
	fmt.Printf("  URL: http://localhost:%d\n", servePort)
	fmt.Printf("  Log: %s\n", logFile)
	fmt.Println("\nUse 'zap serve stop' to stop the server")
	fmt.Println("Use 'zap serve logs' to view logs")

	// Open browser
	if !serveNoBrowser {
		web.OpenBrowserURL(fmt.Sprintf("http://localhost:%d", servePort))
	}

	return nil
}

func runServeStop(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	if err := web.StopDaemon(dir); err != nil {
		return err
	}

	fmt.Println("Server stopped")
	return nil
}

func runServeStatus(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	status, err := web.GetDaemonStatus(dir)
	if err != nil {
		return err
	}

	if !status.Running {
		fmt.Println("Server is not running")
		return nil
	}

	fmt.Printf("Server is running\n")
	fmt.Printf("  PID:  %d\n", status.PID)
	fmt.Printf("  Port: %d\n", status.Port)
	fmt.Printf("  URL:  http://localhost:%d\n", status.Port)
	fmt.Printf("  Log:  %s\n", web.GetLogFilePath(dir))

	return nil
}

func runServeLogs(cmd *cobra.Command, args []string) error {
	dir, err := getIssuesDir(cmd)
	if err != nil {
		return err
	}

	logFile := web.GetLogFilePath(dir)

	// Check if log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logFile)
	}

	// Use tail -f to follow the log
	tailCmd := exec.Command("tail", "-f", logFile)
	tailCmd.Stdout = os.Stdout
	tailCmd.Stderr = os.Stderr

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		tailCmd.Process.Kill()
	}()

	fmt.Printf("Following %s (Ctrl+C to stop)\n\n", logFile)

	// Also print existing content first
	file, err := os.Open(logFile)
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		file.Close()
	}

	// Start following
	return tailCmd.Run()
}
