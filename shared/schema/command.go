package schema

type RawInput struct {
	SessionID string `json:"sessionId"`
	BeaconID  string `json:"beaconId"`
	Text      string `json:"text"`
}

type ParsedCommand struct {
	Raw        string            `json:"raw"`
	Executable string            `json:"executable"`
	Subcommand string            `json:"subcommand"`
	Args       []string          `json:"args"`
	Flags      map[string]string `json:"flags"`
}

type ToolResult struct {
	Output  string       `json:"output"`
	Effects []SideEffect `json:"effects"`
	IsError bool         `json:"isError"`
}

type SideEffect struct {
	// Discriminator
	Type string `json:"type"`

	// SideEffectAddLoot — Type: "add_loot"
	LootType   string `json:"lootType,omitempty"`
	LootValue  string `json:"lootValue,omitempty"`
	LootSource string `json:"lootSource,omitempty"`

	// SideEffectAddTicket — Type: "add_ticket"
	TicketType string `json:"ticketType,omitempty"`
	Identity   string `json:"identity,omitempty"`
	Target     string `json:"target,omitempty"`
	TicketB64  string `json:"ticketB64,omitempty"`

	// SideEffectSetSessionUser — Type: "set_session_user"
	NewUser string `json:"newUser,omitempty"`

	// SideEffectAddToken — Type: "add_token"
	TokenUsername string `json:"tokenUsername,omitempty"`
	TokenDomain   string `json:"tokenDomain,omitempty"`
	TokenFake     string `json:"tokenFake,omitempty"`

	// SideEffectCompleteObjective — Type: "complete_objective"
	ObjectiveID string `json:"objectiveId,omitempty"`

	// SideEffectGrantACL — Type: "acl_granted"
	Principal string `json:"principal,omitempty"`
	ACLTarget string `json:"aclTarget,omitempty"`
	ACLRight  string `json:"aclRight,omitempty"`

	// SideEffectSpawnBeacon — Type: "spawn_beacon"
	BeaconHost     string `json:"beaconHost,omitempty"`
	BeaconIdentity string `json:"beaconIdentity,omitempty"`
	BeaconParentID string `json:"beaconParentId,omitempty"`

	// SideEffectPivotContext — Type: "pivot_context"
	PivotHost     string `json:"pivotHost,omitempty"`
	PivotIdentity string `json:"pivotIdentity,omitempty"`

	// SideEffectScopeViolation — Type: "scope_violation"
	ViolationHost string `json:"violationHost,omitempty"`
}

const (
	SideEffectTypeLoot           = "add_loot"
	SideEffectTypeTicket         = "add_ticket"
	SideEffectTypeSetSessionUser = "set_session_user"
	SideEffectTypeAddToken       = "add_token"
	SideEffectTypeCompleteObj    = "complete_objective"
	SideEffectTypeGrantACL       = "acl_granted"
	SideEffectTypeSpawnBeacon    = "spawn_beacon"
	SideEffectTypePivotContext   = "pivot_context"
	SideEffectTypeScopeViolation = "scope_violation"
)

type PushEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}