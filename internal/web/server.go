package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/itda-work/zap/internal/issue"
)

// Server is the web server for the issue viewer
type Server struct {
	store   *issue.Store
	baseDir string
	port    int
	handler *Handler

	// SSE client management
	clients   map[chan string]bool
	clientsMu sync.RWMutex

	// File watcher
	watcher *fsnotify.Watcher

	// Request counter for logging
	requestCount uint64
}

// NewServer creates a new web server
func NewServer(store *issue.Store, baseDir string, port int) *Server {
	return &Server{
		store:   store,
		baseDir: baseDir,
		port:    port,
		handler: NewHandler(store),
		clients: make(map[chan string]bool),
	}
}

// Start starts the web server
func (s *Server) Start(ctx context.Context) error {
	// Setup file watcher
	var err error
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer s.watcher.Close()

	// Watch the issues directory
	if err := s.watcher.Add(s.baseDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Start watching for file changes
	go s.watchFiles()

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/issues", s.handler.ListIssues)
	mux.HandleFunc("/issues/", s.handleIssues)
	mux.HandleFunc("/events", s.handleSSE)

	// Wrap with logging middleware
	handler := s.loggingMiddleware(mux)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer for SSE
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("Starting server on http://localhost:%d", s.port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// StartAndOpen starts the server and opens the browser
func (s *Server) StartAndOpen(ctx context.Context, path string) error {
	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Start(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Check if server started successfully
	select {
	case err := <-errChan:
		return err
	default:
		// Server is running, open browser
		url := fmt.Sprintf("http://localhost:%d%s", s.port, path)
		if err := OpenBrowserURL(url); err != nil {
			log.Printf("Could not open browser: %v", err)
			log.Printf("Please open %s manually", url)
		}

		// Wait for server to finish
		return <-errChan
	}
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.handler.Dashboard(w, r)
}

func (s *Server) handleIssues(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /issues/:number/view - HTML view
	if len(path) > 8 && path[len(path)-5:] == "/view" {
		s.handler.ViewIssue(w, r)
		return
	}

	// /issues/:number - JSON
	if len(path) > 8 {
		s.handler.GetIssue(w, r)
		return
	}

	// /issues - JSON list
	s.handler.ListIssues(w, r)
}

// handleSSE handles Server-Sent Events for live reload
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a channel for this client
	clientChan := make(chan string, 10)

	// Register client
	s.clientsMu.Lock()
	s.clients[clientChan] = true
	clientCount := len(s.clients)
	s.clientsMu.Unlock()

	log.Printf("SSE client connected (total: %d)", clientCount)

	// Unregister client on disconnect
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientChan)
		clientCount := len(s.clients)
		s.clientsMu.Unlock()
		close(clientChan)
		log.Printf("SSE client disconnected (total: %d)", clientCount)
	}()

	// Get the request context
	ctx := r.Context()

	// Send initial connection message
	fmt.Fprintf(w, "data: connected\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Heartbeat ticker to keep connection alive
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Listen for events
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			// Send heartbeat to keep connection alive
			fmt.Fprintf(w, ": heartbeat\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// watchFiles watches for file changes and notifies SSE clients
func (s *Server) watchFiles() {
	debounce := time.NewTimer(0)
	debounce.Stop()
	defer debounce.Stop()

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only react to write/create/remove events on issue files (.md, .rst)
			// Skip hidden files and other files to avoid infinite reload loops
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				filename := filepath.Base(event.Name)
				isIssueFile := strings.HasSuffix(filename, ".md") || strings.HasSuffix(filename, ".rst")
				if !strings.HasPrefix(filename, ".") && isIssueFile {
					debounce.Reset(100 * time.Millisecond)
				}
			}

		case <-debounce.C:
			// Broadcast reload to all clients
			s.broadcast("reload")

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// broadcast sends a message to all connected SSE clients
func (s *Server) broadcast(msg string) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for clientChan := range s.clients {
		select {
		case clientChan <- msg:
		default:
			// Client buffer is full, skip
		}
	}
}

// OpenBrowserURL opens the specified URL in the default browser
func OpenBrowserURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.port
}

// GetClientCount returns the number of connected SSE clients
func (s *Server) GetClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := atomic.AddUint64(&s.requestCount, 1)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Log format: [req#] METHOD PATH STATUS DURATION [SSE:N]
		sseInfo := ""
		if r.URL.Path == "/events" {
			sseInfo = fmt.Sprintf(" [SSE clients: %d]", s.GetClientCount())
		}

		log.Printf("[%d] %s %s %d %v%s",
			reqID,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			sseInfo,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Implement http.Flusher for SSE support
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
