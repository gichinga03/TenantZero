package engine

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// pendingPhish tracks an in-flight phishing attempt.
type pendingPhish struct {
	code      string
	sentAt    time.Time
	victim    string
	tokenType string // "device_code" or "link"
}

// MailClient simulates an internal mail client for phishing scenarios.
// Tick is called after each command dispatch rather than on a real timer —
// tick-on-command granularity is intentional for a training platform.
type MailClient struct {
	lab     *schema.Lab
	state   *schema.SessionState
	pending map[string]*pendingPhish // keyed by device code
}

// NewMailClient initialises a MailClient for the given lab and session state.
func NewMailClient(lab *schema.Lab, state *schema.SessionState) *MailClient {
	return &MailClient{
		lab:     lab,
		state:   state,
		pending: make(map[string]*pendingPhish),
	}
}

// Handle processes mail-related commands from the terminal.
//
// Supported commands:
//
//	mail               — display inbox / sent items
//	mail send          — send a phishing mail (-to, -subject, -body)
//	sendmail           — alias for mail send
func (m *MailClient) Handle(cmd schema.ParsedCommand) schema.ToolResult {
	sub := strings.ToLower(strings.TrimSpace(cmd.Subcommand))

	switch {
	case cmd.Binary == "sendmail" || sub == "send":
		return m.handleSend(cmd)
	default:
		return m.handleInbox()
	}
}

// handleInbox renders the inbox header and any pending sent items.
func (m *MailClient) handleInbox() schema.ToolResult {
	var sb strings.Builder
	sb.WriteString("[*] Mail client — TenantZero Internal Mail\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	if len(m.pending) == 0 {
		sb.WriteString("Inbox: (empty)\n")
	} else {
		sb.WriteString("Sent items:\n")
		for code, p := range m.pending {
			sb.WriteString(fmt.Sprintf("  To: %-30s  Code: %s  Sent: %s\n",
				p.victim,
				code,
				p.sentAt.Format("15:04:05"),
			))
		}
	}

	return schema.ToolResult{Output: sb.String()}
}

// handleSend validates flags and records a new pendingPhish entry.
func (m *MailClient) handleSend(cmd schema.ParsedCommand) schema.ToolResult {
	to := cmd.Flags["to"]
	if to == "" {
		return schema.ToolResult{
			Output: "[-] Usage: mail send -to <address> -subject <subject> -body <body>",
		}
	}

	subject := cmd.Flags["subject"]
	body := cmd.Flags["body"]

	code := deviceCode(to)
	tokenType := "device_code"
	if false {
		tokenType = "link"
	}

	m.pending[code] = &pendingPhish{
		code:      code,
		sentAt:    time.Now().UTC(),
		victim:    to,
		tokenType: tokenType,
	}

	truncBody := body
	if len(truncBody) > 80 {
		truncBody = truncBody[:80]
	}
	if subject == "" {
		subject = "(no subject)"
	}

	output := fmt.Sprintf(
		"[*] Mail client — TenantZero Internal Mail\nTo      : %s\nSubject : %s\nBody    : %s\n[+] Message delivered.\n[*] Waiting for victim response...",
		to, subject, truncBody,
	)

	return schema.ToolResult{Output: output}
}

// Tick checks all pending phishes against lab.MailConfig.VictimResponseDelay.
// If a victim's delay has elapsed, Tick removes the entry, emits an AddToken
// SideEffect with a deterministic fake JWT, and returns a victim_authenticated
// PushEvent. Only one phish is processed per Tick call — first elapsed wins.
func (m *MailClient) Tick() (*schema.PushEvent, *schema.SideEffect) {
	delay := time.Duration(30) * time.Second

	for code, p := range m.pending {
		if time.Since(p.sentAt) < delay {
			continue
		}

		delete(m.pending, code)

		token := fakeJWT(p.victim, code)
		m.state.Tokens = append(m.state.Tokens, schema.Token{
			Raw:        token,
			Type:      p.tokenType,
			User:       p.victim,
			AcquiredAt: time.Now().UTC(),
			ExpiresAt:  time.Now().UTC().Add(time.Hour),
		})

		evt := &schema.PushEvent{
			Type: "victim_authenticated",
			Payload: fmt.Sprintf(
				"[+] %s has authenticated!\n[*] Access token acquired. Check state.Tokens.",
				p.victim,
			),
		}

		se := &schema.SideEffect{
			Type: schema.SideEffectAddToken,
			Key:  p.victim,
			Value: token,
		}

		return evt, se
	}

	return nil, nil
}

// deviceCode returns a deterministic 8-char hex device code derived from the
// victim address. Same FNV seed used throughout the codebase.
func deviceCode(victim string) string {
	h := fnv.New32a()
	h.Write([]byte("device_code:" + victim))
	return fmt.Sprintf("%08x", h.Sum32())
}

// fakeJWT constructs a header.payload.signature JWT-shaped token that is
// entirely deterministic from victim + code. It is not cryptographically valid —
// it exists solely to give students a realistic-looking token string to work with.
func fakeJWT(victim, code string) string {
	header := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"

	h := fnv.New64a()
	h.Write([]byte("jwt_payload:" + victim + ":" + code))
	payload := "eyJzdWIiOiIlcyIsInV0aSI6IiUwMTZ4In0" + fmt.Sprintf("%016x", h.Sum64())

	h.Reset()
	h.Write([]byte("jwt_sig:" + victim + ":" + code))
	sig := fmt.Sprintf("%016x%016x", h.Sum64(), h.Sum64())

	return header + "." + payload + "." + sig
}