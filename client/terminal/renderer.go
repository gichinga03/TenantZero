package terminal

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes.
const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
)

// Renderer manages terminal output for a TenantZero session.
type Renderer struct {
	hostname string
	username string
	cwd      string
}

// NewRenderer constructs a Renderer with initial context.
func NewRenderer(hostname, username, cwd string) *Renderer {
	return &Renderer{hostname: hostname, username: username, cwd: cwd}
}

// Prompt returns a PowerShell-style prompt string.
// Format: "PS <cwd>> "
func (r *Renderer) Prompt() string {
	return fmt.Sprintf("PS %s> ", r.cwd)
}

// Print writes s to stdout followed by a CRLF-aware newline.
func (r *Renderer) Print(s string) {
	fmt.Fprintln(os.Stdout, s)
}

// PrintEvent prints a push event in a visually distinct style.
//
// Event types:
//   - objective_complete — green "[*]" prefix
//   - lab_complete       — green banner box
//   - token_expired      — yellow warning
//   - victim_authenticated — cyan notification
func (r *Renderer) PrintEvent(eventType, payload string) {
	switch eventType {
	case "objective_complete":
		fmt.Fprintf(os.Stdout, "%s%s[*] %s%s\n", ansiGreen, ansiBold, payload, ansiReset)

	case "lab_complete":
		width := 60
		border := strings.Repeat("=", width)
		msg := " LAB COMPLETE "
		pad := (width - len(msg)) / 2
		centered := strings.Repeat("=", pad) + msg + strings.Repeat("=", width-pad-len(msg))
		fmt.Fprintf(os.Stdout, "\n%s%s%s\n%s\n%s%s\n\n",
			ansiGreen, ansiBold,
			border,
			centered,
			border,
			ansiReset,
		)
		fmt.Fprintf(os.Stdout, "%s%s%s%s\n", ansiGreen, ansiBold, payload, ansiReset)

	case "token_expired":
		fmt.Fprintf(os.Stdout, "%s%s[!] %s%s\n", ansiYellow, ansiBold, payload, ansiReset)

	case "victim_authenticated":
		fmt.Fprintf(os.Stdout, "%s%s[+] %s%s\n", ansiCyan, ansiBold, payload, ansiReset)

	default:
		fmt.Fprintf(os.Stdout, "[event:%s] %s\n", eventType, payload)
	}
}

// UpdateContext updates hostname, username, and cwd after a session context change.
func (r *Renderer) UpdateContext(hostname, username, cwd string) {
	if hostname != "" {
		r.hostname = hostname
	}
	if username != "" {
		r.username = username
	}
	if cwd != "" {
		r.cwd = cwd
	}
}

// ClearScreen clears the terminal using ANSI escape codes.
func (r *Renderer) ClearScreen() {
	fmt.Fprint(os.Stdout, "\033[2J\033[H")
}