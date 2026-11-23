package schema

import "time"

// RawInput is the first representation of a student command — exactly what
// arrived over the WebSocket before any parsing. The session ID is embedded
// so the command can be routed to the correct engine instance without
// additional context.
type RawInput struct {
	// SessionID identifies the session this command belongs to.
	SessionID string `json:"session_id"`

	// Raw is the unmodified command string as typed by the student.
	Raw string `json:"raw"`

	// Timestamp is the server-side arrival time, used for journal ordering.
	Timestamp time.Time `json:"timestamp"`
}

// ParsedCommand is the structured form of a command after the parser has
// tokenised the raw input. The tool registry dispatches on Binary + Subcommand.
type ParsedCommand struct {
	// Binary is the first token — the tool name (e.g. "Rubeus", "net").
	Binary string `json:"binary"`

	// Subcommand is the second token where the tool follows a
	// binary-subcommand pattern (e.g. "asktgt", "user"). Empty for tools
	// that do not use subcommands.
	Subcommand string `json:"subcommand"`

	// Flags maps flag names (without leading dashes) to their values.
	// Boolean flags have an empty string value.
	Flags map[string]string `json:"flags"`

	// Args holds positional arguments that follow the subcommand and
	// are not flag values.
	Args []string `json:"args"`

	// Raw is preserved from RawInput to allow the engine to write the
	// original command string into the journal without reformatting.
	Raw string `json:"raw"`
}

// ToolResult is what a tool handler returns to the engine after execution.
// The engine applies SideEffects to the session state before writing the
// Output back to the student.
type ToolResult struct {
	// Output is the text to display to the student verbatim.
	Output string `json:"output"`

	// SideEffects is an ordered list of state mutations the tool requests.
	// The engine applies them atomically after validating each one.
	SideEffects []SideEffect `json:"side_effects"`

	// Success indicates whether the command completed without a simulated
	// error. The engine uses this to decide whether to evaluate win conditions.
	Success bool `json:"success"`

	// IsError flags a hard error (bad syntax, missing required arg).
	IsError bool `json:"is_error"`
}

// SideEffect is a single declarative state mutation emitted by a tool handler.
// Using a discriminated union of typed effects keeps tool handlers decoupled
// from the session state structure — handlers never write to SessionState
// directly.
type SideEffect struct {
	// Type identifies the mutation:
	//   - "set_loot":             write Key/Value into the appropriate Loot field.
	//   - "set_session_user":     change SessionState.User to Value.
	//   - "set_session_machine":  change SessionState.Hostname to Value.
	//   - "add_ticket":           append a Ticket (Value is JSON-encoded Ticket).
	//   - "add_token":            append a Token (Value is JSON-encoded Token).
	//   - "set_cwd":              change SessionState.CWD to Value.
	Type string `json:"type"`

	// Key is the secondary discriminator used by "set_loot" to identify
	// which Loot sub-map to write into (e.g. "ntlm", "spn", "plaintext",
	// "pfx", "arbitrary").
	Key string `json:"key"`

	// Value is the data payload. For simple types it is the literal value;
	// for "add_ticket" and "add_token" it is a JSON-encoded struct.
	Value string `json:"value"`
}

// PushEvent is a server-initiated message sent to the client over the
// WebSocket outside the normal request/response cycle. The client renders
// these as banner notifications.
type PushEvent struct {
	// Type identifies the event:
	//   - "objective_complete":    a WinCondition was satisfied.
	//   - "lab_complete":          all WinConditions are satisfied.
	//   - "token_expired":         a Ticket or Token has passed its ExpiresAt.
	//   - "victim_authenticated":  the simulated phishing victim clicked.
	//   - "hint":                  a hint was unlocked.
	Type string `json:"type"`

	// Payload is event-specific data. For "objective_complete" it is the
	// WinCondition.Label; for "hint" it is HintConfig.Text; for
	// "token_expired" it is the ticket/token identifier.
	Payload string `json:"payload"`
}
// SideEffect type constants
const (
	SideEffectSetLoot           = "set_loot"
	SideEffectSetSessionUser    = "set_session_user"
	SideEffectSetSessionMachine = "set_session_machine"
	SideEffectAddTicket         = "add_ticket"
	SideEffectAddToken          = "add_token"
	SideEffectSetCWD            = "set_cwd"
)

