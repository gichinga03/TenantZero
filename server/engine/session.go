package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/shared/schema"
	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/server/engine/tools"
)

// Session is the runtime container for a single student lab run. It wires
// together immutable lab data, mutable session state, the AD/Azure provider,
// and the tool registry.
type Session struct {
	ID         string
	Lab        *schema.Lab
	Engagement *schema.Engagement
	State      *schema.SessionState
	Provider   providers.Provider
	Registry   *tools.Registry
}

// NewSession constructs a Session. The caller is responsible for supplying a
// correctly initialised state (via InitState) and a fully populated registry.
func NewSession(
	lab *schema.Lab,
	eng *schema.Engagement,
	state *schema.SessionState,
	provider providers.Provider,
	registry *tools.Registry,
) *Session {
	return &Session{
		ID:         state.ID,
		Lab:        lab,
		Engagement: eng,
		State:      state,
		Provider:   provider,
		Registry:   registry,
	}
}

// Apply processes every SideEffect in result against session state. Tools are
// stateless; Apply is the single write path for all state mutations.
func (s *Session) Apply(result schema.ToolResult) {
	for _, se := range result.SideEffects {
		s.applySideEffect(se)
	}
}

func (s *Session) applySideEffect(se schema.SideEffect) {
	switch se.Type {
	case schema.SideEffectSetLoot:
		// Key format: "category.username"
		// Categories: ntlm, password, certificate
		parts := strings.SplitN(se.Key, ".", 2)
		if len(parts) != 2 {
			return
		}
		category, username := parts[0], parts[1]
		switch category {
		case "ntlm":
			s.State.Loot.NTLMHashes[username] = se.Value
		case "plaintext":
			s.State.Loot.PlaintextCreds[username] = se.Value
		case "pfx":
			s.State.Loot.PFXCerts[username] = se.Value
		}

	case schema.SideEffectSetSessionUser:
		s.State.User = se.Value

	case schema.SideEffectSetSessionMachine:
		s.State.Hostname = se.Value
		s.State.CWD = `C:\`

	case schema.SideEffectAddTicket:
		var t schema.Ticket
		if err := json.Unmarshal([]byte(se.Value), &t); err != nil {
			return
		}
		s.State.Tickets = append(s.State.Tickets, t)

	case schema.SideEffectAddToken:
		var tok schema.Token
		if err := json.Unmarshal([]byte(se.Value), &tok); err != nil {
			return
		}
		s.State.Tokens = append(s.State.Tokens, tok)

	case schema.SideEffectSetCWD:
		// Filesystem.Cd already validated the path; Apply just stores it.
		s.State.CWD = se.Value
	}
}

// CheckTimeouts returns a PushEvent if the overall session has exceeded
// lab.TimeConfig.SessionTimeoutMinutes, or if any active token has exceeded
// lab.TimeConfig.TokenTTLMinutes. Returns nil when all timers are within bounds.
//
// Called by the WebSocket handler on every inbound command before dispatch.
func (s *Session) CheckTimeouts() *schema.PushEvent {
	cfg := s.Lab.Time
	now := time.Now().UTC()

	if cfg.SessionTimeout > 0 {
		deadline := s.State.StartedAt.Add(
			time.Duration(cfg.SessionTimeout) * time.Minute,
		)
		if now.After(deadline) {
			return &schema.PushEvent{
				Type:    "session_timeout",
				Payload: fmt.Sprintf("Session expired after %d minutes.", cfg.SessionTimeout),
			}
		}
	}

	if cfg.TokenTTL > 0 {
		ttl := time.Duration(cfg.TokenTTL) * time.Minute
		for _, tok := range s.State.Tokens {
			if now.After(tok.AcquiredAt.Add(ttl)) {
				return &schema.PushEvent{
					Type:    "token_expired",
					Payload: fmt.Sprintf("Token for %s has expired (TTL %d min).", tok.User, cfg.TokenTTL),
				}
			}
		}
	}

	return nil
}

// AppendJournal adds a timestamped, auto-sequenced entry to state.Journal.
func (s *Session) AppendJournal(input, output string) {
	seq := len(s.State.Journal) + 1
	s.State.Journal = append(s.State.Journal, schema.JournalEntry{
		Sequence:  seq,
		Timestamp: time.Now().UTC(),
		Input:     input,
		Output:    output,
	})
}