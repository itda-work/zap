package cli

import (
	"os"

	"github.com/mattn/go-isatty"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// colorEnabled indicates whether color output is supported
var colorEnabled bool

func init() {
	// NO_COLOR environment variable support (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
		return
	}

	// Check if stdout is a terminal
	colorEnabled = isatty.IsTerminal(os.Stdout.Fd()) ||
		isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// colorize wraps text with ANSI color codes if color is enabled
func colorize(text, color string) string {
	if !colorEnabled || color == "" {
		return text
	}
	return color + text + colorReset
}
