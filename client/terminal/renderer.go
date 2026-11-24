package terminal

import (
	"fmt"
	"os"
	"strings"
)

const (
	reset    = "\033[0m"
	bold     = "\033[1m"
	cyan     = "\033[36m"
	white    = "\033[97m"
	darkGray = "\033[90m"
	yellow   = "\033[33m"
)

// Renderer manages terminal output for a TenantZero session.
type Renderer struct {
	hostname string
	username string
	cwd      string
}

func NewRenderer(hostname, username, cwd string) *Renderer {
	return &Renderer{hostname: hostname, username: username, cwd: cwd}
}

// Prompt returns a colored PowerShell-style prompt.
func (r *Renderer) Prompt() string {
	return fmt.Sprintf("\r%sPS %s%s%s>%s ", yellow, white, r.cwd, yellow, reset)
}

// Print writes output with a blank line after, like real PowerShell.
func (r *Renderer) Print(output string) {
	if output == "" {
		return
	}
	output = strings.ReplaceAll(output, "\r\n", "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	os.Stdout.WriteString(output)
}

// ClearScreen clears the terminal and shows the PowerShell header.
func (r *Renderer) ClearScreen() {
	os.Stdout.WriteString("\033[2J\033[H")
	os.Stdout.WriteString(darkGray + "Windows PowerShell\n")
	os.Stdout.WriteString("Copyright (C) Microsoft Corporation. All rights reserved.\n\n")
	os.Stdout.WriteString("Try the new cross-platform PowerShell https://aka.ms/pscore6\n" + reset + "\n")
}

// UpdateContext updates hostname, username, and cwd.
func (r *Renderer) UpdateContext(hostname, username, cwd string) {
	if hostname != "" {
		r.hostname = hostname
	}
	if username != "" {
		r.username = username
	}
	if cwd != "" {
		r.cwd = cwd
		fmt.Fprintf(os.Stdout, "\033]0;%s\007", cwd)
	}
}

// PrintEvent prints a system event message.
func (r *Renderer) PrintEvent(eventType, msg string) {
	os.Stdout.WriteString("\n" + cyan + bold + "[!] " + reset + msg + "\n\n")
}

// PrintBanner prints the session start banner.
func PrintBanner(labName string) {
	fmt.Printf("%s%s[*]%s Starting offline session: %s%s%s\n\n",
		bold, cyan, reset, white, labName, reset)
}
