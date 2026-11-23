package schema

import "time"

// SessionState is the complete runtime state of an active lab session.
// It is serialised as JSONB and stored in PostgreSQL; the engine loads it
// on session resume and writes it back after every command that produces
// side effects.
type SessionState struct {
	// ID is the UUID that uniquely identifies this session.
	ID string `json:"id"`

	// LabID is the identifier of the Lab definition this session runs against.
	LabID string `json:"lab_id"`

	// User is the sAMAccountName of the AD principal the student is currently
	// operating as.
	User string `json:"user"`

	// Hostname is the NetBIOS name of the machine the student's shell is
	// currently anchored to.
	Hostname string `json:"hostname"`

	// CWD is the current working directory path in the simulated filesystem.
	CWD string `json:"cwd"`

	// Loot aggregates all credentials and artefacts the student has collected.
	Loot Loot `json:"loot"`

	// Tickets lists Kerberos tickets the student has obtained or forged.
	Tickets []Ticket `json:"tickets"`

	// Tokens lists OAuth/device-code tokens obtained during the session,
	// relevant for Azure-hybrid labs.
	Tokens []Token `json:"tokens"`

	// CompletedObjectives tracks which win conditions have been satisfied
	// and when.
	CompletedObjectives []CompletedObjective `json:"completed_objectives"`

	// UsedHints tracks which hint indices have been shown to the student.
	UsedHints []int `json:"used_hints"`

	// Journal is the append-only command history for this session.
	// It persists across reconnections and is used for replays and scoring.
	Journal []JournalEntry `json:"journal"`

	// StartedAt is the wall-clock time the session was first created.
	StartedAt time.Time `json:"started_at"`

	// LastActive is updated on every command dispatch, used to enforce
	// the session timeout defined in TimeConfig.
	LastActive time.Time `json:"last_active"`

	// Mode mirrors Lab.Mode and is stored here so the engine can make
	// routing decisions without re-fetching the lab definition.
	Mode string `json:"mode"`
}

// Loot holds all credential material and artefacts accumulated during a session.
// Map keys throughout are sAMAccountNames unless otherwise noted.
type Loot struct {
	// NTLMHashes maps sAMAccountName to NT hash (hex string, no colons).
	NTLMHashes map[string]string `json:"ntlm_hashes"`

	// SPNHashes maps SPN string to the Kerberoast hash (Hashcat format).
	SPNHashes map[string]string `json:"spn_hashes"`

	// PlaintextCreds maps sAMAccountName to plaintext password.
	PlaintextCreds map[string]string `json:"plaintext_creds"`

	// NTDSDump is the ordered list of "user:NTLM" lines produced by a
	// DCSync or volume-shadow NTDS.dit extraction.
	NTDSDump []string `json:"ntds_dump"`

	// PFXCerts maps certificate name to base64-encoded PFX blob,
	// used in ADCS ESC attack paths.
	PFXCerts map[string]string `json:"pfx_certs"`

	// Arbitrary holds any other loot the tool ecosystem produces that does
	// not fit a first-class field, keyed by an engine-assigned label.
	Arbitrary map[string]string `json:"arbitrary"`
}

// Ticket represents a Kerberos ticket held in the student's simulated credential store.
type Ticket struct {
	// Type classifies the ticket: "TGT", "TGS", "golden", or "silver".
	Type string `json:"type"`

	// User is the principal the ticket was issued for or forged as.
	User string `json:"user"`

	// Domain is the realm the ticket is valid in.
	Domain string `json:"domain"`

	// Service is the SPN the TGS was issued for. Empty for TGTs.
	Service string `json:"service"`

	// Hash is the Kerberos key material (RC4/AES) used to encrypt the ticket.
	// For TGTs this is the krbtgt hash; for service tickets the service account hash.
	Hash string `json:"hash"`

	// Forged is true for golden and silver tickets created offline rather than
	// requested from a KDC.
	Forged bool `json:"forged"`

	// ExpiresAt is the simulated expiry time of the ticket, derived from
	// TimeConfig.TokenTTL at issuance.
	ExpiresAt time.Time `json:"expires_at"`
}

// Token represents an OAuth2 or device-code token used in Entra ID / Azure
// hybrid attack paths.
type Token struct {
	// Type classifies the token: "access", "refresh", or "device_code".
	Type string `json:"type"`

	// Scope is the OAuth2 scope string the token was issued for
	// (e.g. "https://graph.microsoft.com/.default").
	Scope string `json:"scope"`

	// User is the UPN of the principal the token was issued to.
	User string `json:"user"`

	// AcquiredAt is the time the token was obtained.
	AcquiredAt time.Time `json:"acquired_at"`

	// ExpiresAt is the simulated expiry derived from TimeConfig.TokenTTL.
	ExpiresAt time.Time `json:"expires_at"`

	// Raw is the full token string as it would appear in a real token cache,
	// used by tools that require presenting the token verbatim.
	Raw string `json:"raw"`
}

// CompletedObjective records the moment a win condition was satisfied.
type CompletedObjective struct {
	// Label mirrors WinCondition.Label, identifying which objective was met.
	Label string `json:"label"`

	// CompletedAt is the wall-clock time the engine evaluated the condition
	// as satisfied.
	CompletedAt time.Time `json:"completed_at"`
}

// JournalEntry is a single immutable record in the session command log.
// Entries are appended in sequence and never modified.
type JournalEntry struct {
	// Sequence is a monotonically increasing integer used to order entries
	// deterministically even when wall-clock timestamps collide.
	Sequence int `json:"sequence"`

	// Input is the raw command string as typed by the student.
	Input string `json:"input"`

	// Output is the full text the engine returned for this command.
	Output string `json:"output"`

	// Timestamp is the wall-clock time the command was received.
	Timestamp time.Time `json:"timestamp"`

	// Machine is the Hostname value at the time of execution, capturing
	// lateral movement in the log.
	Machine string `json:"machine"`

	// User is the User value at the time of execution, capturing
	// privilege changes in the log.
	User string `json:"user"`
}
// CurrentUser returns the active user (alias for User).
func (s *SessionState) GetCurrentUser() string { return s.User }

// CurrentMachine returns the active hostname (alias for Hostname).
func (s *SessionState) GetCurrentMachine() string { return s.Hostname }
