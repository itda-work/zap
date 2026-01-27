package cli

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

// Theme represents the terminal color theme
type Theme string

const (
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

// ANSI color codes - Dark theme (default)
const (
	colorReset = "\033[0m"

	// Dark theme colors (bright colors on dark background)
	colorRedDark    = "\033[31m"
	colorGreenDark  = "\033[32m"
	colorYellowDark = "\033[33m"
	colorBlueDark   = "\033[34m"
	colorCyanDark   = "\033[36m"
	colorGrayDark   = "\033[90m"

	colorMagentaDark       = "\033[35m"
	colorBrightMagentaDark = "\033[95m"
	colorBrightGreenDark   = "\033[92m"
	colorBrightYellowDark  = "\033[93m"
	colorLightGrayDark     = "\033[37m"

	bgGrayDark = "\033[48;5;238m" // Dark gray background

	// Light theme colors (dark colors on light background)
	colorRedLight    = "\033[31m"
	colorGreenLight  = "\033[32m"
	colorYellowLight = "\033[33m"
	colorBlueLight   = "\033[34m"
	colorCyanLight   = "\033[36m"
	colorGrayLight   = "\033[90m"

	// Darker variants for light theme (more contrast)
	colorDarkGreen   = "\033[38;5;22m"  // Dark green
	colorDarkYellow  = "\033[38;5;136m" // Dark yellow/olive
	colorDarkGray    = "\033[38;5;240m" // Darker gray
	colorDarkMagenta = "\033[38;5;127m" // Dark magenta

	bgGrayLight = "\033[48;5;253m" // Light gray background for light theme

	// Invert mode (swap foreground and background)
	colorInvert = "\033[7m"
)

// Theme-aware color variables (set during init)
var (
	colorRed    string
	colorGreen  string
	colorYellow string
	colorBlue   string
	colorCyan   string
	colorGray   string

	colorMagenta      string
	colorBrightMagenta string
	colorBrightGreen  string
	colorBrightYellow string
	colorLightGray    string

	bgGray string
)

// colorEnabled indicates whether color output is supported
var colorEnabled bool

// currentTheme holds the detected or configured theme
var currentTheme Theme

func init() {
	// NO_COLOR environment variable support (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
		return
	}

	// Check if stdout is a terminal
	colorEnabled = isatty.IsTerminal(os.Stdout.Fd()) ||
		isatty.IsCygwinTerminal(os.Stdout.Fd())

	if colorEnabled {
		// Detect theme and set colors accordingly
		currentTheme = detectTheme()
		applyThemeColors(currentTheme)
	}
}

// detectTheme determines the terminal theme
func detectTheme() Theme {
	// 1. ZAP_THEME environment variable takes priority
	if theme := os.Getenv("ZAP_THEME"); theme != "" {
		switch strings.ToLower(theme) {
		case "light":
			return ThemeLight
		case "dark":
			return ThemeDark
		}
	}

	// 2. Try OSC 11 query
	if bg, err := queryTerminalBackground(); err == nil {
		if isLightColor(bg) {
			return ThemeLight
		}
		return ThemeDark
	}

	// 3. COLORFGBG fallback (format: "fg;bg" e.g., "15;0" means light fg on dark bg)
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			if bg, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				// Background values 0-6 and 8 are dark, 7 and 9-15 are light
				if bg == 7 || (bg >= 9 && bg <= 15) {
					return ThemeLight
				}
			}
		}
	}

	// 4. Default to dark theme
	return ThemeDark
}

// RGB represents a color with red, green, blue components (0-65535 range)
type RGB struct {
	R, G, B uint16
}

// queryTerminalBackground queries the terminal for its background color using OSC 11
func queryTerminalBackground() (RGB, error) {
	// Check if stdin is a terminal (required for OSC query)
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return RGB{}, os.ErrInvalid
	}

	// Save terminal state and set raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return RGB{}, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Send OSC 11 query: \033]11;?\033\\
	os.Stdout.WriteString("\033]11;?\033\\")

	// Read response with timeout
	buf := make([]byte, 64)
	responseChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- buf[:n]
	}()

	select {
	case response := <-responseChan:
		return parseOSC11Response(string(response))
	case err := <-errChan:
		return RGB{}, err
	case <-time.After(100 * time.Millisecond):
		return RGB{}, os.ErrDeadlineExceeded
	}
}

// parseOSC11Response parses the OSC 11 response
// Format: \033]11;rgb:RRRR/GGGG/BBBB\033\\ or \033]11;rgb:RRRR/GGGG/BBBB\007
func parseOSC11Response(response string) (RGB, error) {
	// Find rgb: prefix
	idx := strings.Index(response, "rgb:")
	if idx == -1 {
		return RGB{}, os.ErrInvalid
	}

	// Extract the color part
	colorPart := response[idx+4:]

	// Find the end (either \033\\ or \007 or end of string)
	endIdx := strings.IndexAny(colorPart, "\033\007")
	if endIdx != -1 {
		colorPart = colorPart[:endIdx]
	}

	// Parse RRRR/GGGG/BBBB
	parts := strings.Split(colorPart, "/")
	if len(parts) != 3 {
		return RGB{}, os.ErrInvalid
	}

	r, err := strconv.ParseUint(parts[0], 16, 16)
	if err != nil {
		return RGB{}, err
	}
	g, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return RGB{}, err
	}
	b, err := strconv.ParseUint(parts[2], 16, 16)
	if err != nil {
		return RGB{}, err
	}

	return RGB{R: uint16(r), G: uint16(g), B: uint16(b)}, nil
}

// isLightColor determines if a color is considered "light"
// Uses relative luminance formula: Y = 0.2126*R + 0.7152*G + 0.0722*B
func isLightColor(c RGB) bool {
	// Normalize to 0-1 range (from 0-65535)
	r := float64(c.R) / 65535.0
	g := float64(c.G) / 65535.0
	b := float64(c.B) / 65535.0

	// Calculate relative luminance
	luminance := 0.2126*r + 0.7152*g + 0.0722*b

	// Threshold: 0.5 is a reasonable midpoint
	return luminance > 0.5
}

// applyThemeColors sets the color variables based on the current theme
func applyThemeColors(theme Theme) {
	switch theme {
	case ThemeLight:
		colorRed = colorRedLight
		colorGreen = colorDarkGreen
		colorYellow = colorDarkYellow
		colorBlue = colorBlueLight
		colorCyan = colorCyanLight
		colorGray = colorDarkGray
		colorMagenta = colorDarkMagenta
		colorBrightMagenta = colorDarkMagenta

		colorBrightGreen = colorDarkGreen   // Use dark green for light theme
		colorBrightYellow = colorDarkYellow // Use dark yellow for light theme
		colorLightGray = colorDarkGray

		bgGray = bgGrayLight

	default: // ThemeDark
		colorRed = colorRedDark
		colorGreen = colorGreenDark
		colorYellow = colorYellowDark
		colorBlue = colorBlueDark
		colorCyan = colorCyanDark
		colorGray = colorGrayDark
		colorMagenta = colorMagentaDark
		colorBrightMagenta = colorBrightMagentaDark

		colorBrightGreen = colorBrightGreenDark
		colorBrightYellow = colorBrightYellowDark
		colorLightGray = colorLightGrayDark

		bgGray = bgGrayDark
	}
}

// GetCurrentTheme returns the currently active theme
func GetCurrentTheme() Theme {
	return currentTheme
}

// colorize wraps text with ANSI color codes if color is enabled
func colorize(text, color string) string {
	if !colorEnabled || color == "" {
		return text
	}
	return color + text + colorReset
}

// colorizeInvert wraps text with inverted colors (fg/bg swapped)
func colorizeInvert(text, color string) string {
	if !colorEnabled {
		return text
	}
	if color == "" {
		return colorInvert + text + colorReset
	}
	return color + colorInvert + text + colorReset
}

// colorizeWithBg wraps text with foreground color and background color
func colorizeWithBg(text, fgColor, bgColor string) string {
	if !colorEnabled {
		return text
	}
	return bgColor + fgColor + text + colorReset
}
