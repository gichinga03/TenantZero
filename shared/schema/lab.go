// Package schema defines the foundational types shared across the TenantZero
// platform — lab definitions, session state, command pipeline, and engagement
// documents. All other packages (server engine, tool registry, CLI client)
// import from here. No external dependencies.
package schema


// Lab is the top-level structure parsed from lab.yaml. It describes a complete
// simulated Active Directory environment including topology, users, ACLs,
// win conditions, and tooling constraints.
type Lab struct {
	// ID is the canonical unique identifier for this lab, used as a foreign
	// key in session records and engagement documents.
	ID string `yaml:"id" json:"id"`

	// Name is the human-readable display name shown in the lab browser.
	Name string `yaml:"name" json:"name"`

	// Author is the lab designer's handle or name.
	Author string `yaml:"author" json:"author"`

	// Difficulty is a free-form string (e.g. "beginner", "intermediate",
	// "advanced") used for filtering and display.
	Difficulty string `yaml:"difficulty" json:"difficulty"`

	// EstimatedTime is the expected completion time in minutes.
	EstimatedTime int `yaml:"estimated_time" json:"estimated_time"`

	// Tags are arbitrary labels used for search and categorisation
	// (e.g. ["kerberoasting", "ADCS", "lateral-movement"]).
	Tags []string `yaml:"tags" json:"tags"`

	// Mode controls the starting context. "full-chain" begins unauthenticated;
	// "drop-in" places the student at a specific machine/user combination
	// defined in StartingPoint.
	Mode string `yaml:"mode" json:"mode"`

	// Domains lists every AD domain in the simulated forest.
	Domains []Domain `yaml:"domains" json:"domains"`

	// Users lists every principal that exists in the simulated directory.
	Users []User `yaml:"users" json:"users"`

	// Groups lists every AD security and distribution group.
	Groups []Group `yaml:"groups" json:"groups"`

	// Computers lists every machine in the simulated environment.
	Computers []Computer `yaml:"computers" json:"computers"`

	// ACLs describes discretionary access control edges between objects.
	ACLs []ACL `yaml:"acls" json:"acls"`

	// Trusts describes inter-domain and inter-forest trust relationships.
	Trusts []Trust `yaml:"trusts" json:"trusts"`

	// CertTemplates lists Active Directory Certificate Services templates
	// that may be intentionally misconfigured to demonstrate ESC attack paths.
	CertTemplates []CertTemplate `yaml:"cert_templates" json:"cert_templates"`

	// Delegations lists Kerberos delegation configurations on computer accounts.
	Delegations []Delegation `yaml:"delegations" json:"delegations"`

	// FileSystem is the simulated directory tree, mapping absolute path to a
	// list of child entry names. Used by the simulated filesystem tool.
	FileSystem FileSystem `yaml:"file_system" json:"file_system"`

	// FileContents maps absolute paths to their raw string content.
	// Only paths present here are readable by the student.
	FileContents FileContents `yaml:"file_contents" json:"file_contents"`

	// Tools describes which binaries are available and any per-subcommand
	// constraints placed on their use.
	Tools ToolConfig `yaml:"tools" json:"tools"`

	// WinConditions lists the ordered set of objectives the student must
	// satisfy to complete the lab.
	WinConditions []WinCondition `yaml:"win_conditions" json:"win_conditions"`

	// Time controls session and token lifetime parameters.
	Time TimeConfig `yaml:"time" json:"time"`

	// Hints lists progressive hints and the state conditions that unlock them.
	Hints []HintConfig `yaml:"hints" json:"hints"`

	// StartingPoint defines the initial machine and user context for
	// "drop-in" mode labs.
	StartingPoint StartingPoint `yaml:"starting_point" json:"starting_point"`

	// Mail configures the simulated phishing/mail subsystem timing.
	Mail MailConfig `yaml:"mail" json:"mail"`
}

// Domain represents a single Active Directory domain within the simulated forest.
type Domain struct {
	Name            string         `yaml:"name" json:"name"`
	NetBIOS         string         `yaml:"netbios" json:"netbios"`
	DC              string         `yaml:"dc" json:"dc"`
	FunctionalLevel string         `yaml:"functional_level" json:"functional_level"`
	Users           []User         `yaml:"users" json:"users"`
	Groups          []Group        `yaml:"groups" json:"groups"`
	Computers       []Computer     `yaml:"computers" json:"computers"`
	ACLs            []ACL          `yaml:"acls" json:"acls"`
	Trusts          []Trust        `yaml:"trusts" json:"trusts"`
	CertTemplates   []CertTemplate `yaml:"cert_templates" json:"cert_templates"`
	Delegations     []Delegation   `yaml:"delegations" json:"delegations"`
}

// User represents a simulated Active Directory user account.
type User struct {
	// Username is the sAMAccountName of the account.
	Username string `yaml:"username" json:"username"`

	// Password is the account's plaintext credential as defined in the lab.
	// The engine uses this to validate authentication attempts.
	Password string `yaml:"password" json:"password"`

	// Domain is the FQDN of the domain this account belongs to.
	Domain string `yaml:"domain" json:"domain"`

	// Enabled indicates whether the account is active. Disabled accounts
	// will reject authentication.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Groups lists the names of AD groups this user is a direct member of.
	Groups []string `yaml:"groups" json:"groups"`

	// SPN lists the Service Principal Names registered on this account,
	// making it kerberoastable when non-empty.
	SPN []string `yaml:"spn" json:"spn"`

	// AdminCount mirrors the AD adminCount attribute; a value of 1 indicates
	// the account is or was protected by AdminSDHolder.
	AdminCount int `yaml:"adminCount" json:"adminCount"`

	// Description is the account's directory description field. Lab authors
	// frequently embed credentials or hints here to simulate real-world misconfigurations.
	Description string `yaml:"description" json:"description"`

	// MemberOf is the distinguished name list for reverse-membership lookups,
	// mirroring the AD memberOf attribute.
	MemberOf []string `yaml:"memberOf" json:"memberOf"`
}

// Group represents an Active Directory security or distribution group.
type Group struct {
	// Name is the sAMAccountName of the group.
	Name string `yaml:"name" json:"name"`

	// Members lists the sAMAccountNames of direct members of this group.
	Members []string `yaml:"members" json:"members"`

	// Description is the group's directory description field.
	Description string `yaml:"description" json:"description"`
}

// Computer represents a simulated machine in the AD environment.
type Computer struct {
	// Hostname is the NetBIOS name of the machine.
	Hostname string `yaml:"hostname" json:"hostname"`

	// FQDN is the fully-qualified domain name (e.g. "DC01.corp.local").
	FQDN string `yaml:"fqdn" json:"fqdn"`

	// OS is the operating system string returned by enumeration tools
	// (e.g. "Windows Server 2019").
	OS string `yaml:"os" json:"os"`

	// IP is the simulated IPv4 address of the machine.
	IP string `yaml:"ip" json:"ip"`

	// Roles lists the machine's AD roles. Valid values are "DomainController"
	// and "MemberServer".
	Roles []string `yaml:"roles" json:"roles"`

	// LocalAdmins lists the sAMAccountNames of users and groups that hold
	// local administrator rights on this machine.
	LocalAdmins []string `yaml:"local_admins" json:"local_admins"`
}

// ACL represents a discretionary access control edge between two directory objects.
// Edges are directional: From holds the right over To.
type ACL struct {
	// From is the sAMAccountName of the principal that holds the right.
	From string `yaml:"from" json:"from"`

	// To is the sAMAccountName of the target object.
	To string `yaml:"to" json:"to"`

	// Right is the specific ACE. Valid values: "GenericWrite", "WriteDACL",
	// "GenericAll", "AllExtendedRights", "ForceChangePassword", "Owns".
	Right string `yaml:"right" json:"right"`
}

// Trust describes a relationship between two AD domains.
type Trust struct {
	// FromDomain is the FQDN of the trusting domain.
	FromDomain string `yaml:"from_domain" json:"from_domain"`

	// ToDomain is the FQDN of the trusted domain.
	ToDomain string `yaml:"to_domain" json:"to_domain"`

	// Direction is "one-way" or "bidirectional".
	Direction string `yaml:"direction" json:"direction"`

	// TrustType is "external" (cross-domain, same forest) or "forest"
	// (cross-forest).
	TrustType string `yaml:"trust_type" json:"trust_type"`
}

// CertTemplate represents an ADCS certificate template. Fields map directly
// to the properties checked by ESC vulnerability conditions.
type CertTemplate struct {
	// Name is the template's display name in the ADCS console.
	Name string `yaml:"name" json:"name"`

	// Enabled indicates whether the template is published on a CA.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// EnrolleeSuppliesSubject corresponds to the CT_FLAG_ENROLLEE_SUPPLIES_SUBJECT
	// flag; true enables ESC1 when combined with other conditions.
	EnrolleeSuppliesSubject bool `yaml:"enrollee_supplies_subject" json:"enrollee_supplies_subject"`

	// RequiresManagerApproval maps to the EDITF_ATTRIBUTESUBJECTALTNAME2 flag;
	// false permits immediate issuance.
	RequiresManagerApproval bool `yaml:"requires_manager_approval" json:"requires_manager_approval"`

	// AuthorizedSignatures is the number of authorised CA manager signatures
	// required before issuance. Zero means none required.
	AuthorizedSignatures int `yaml:"authorized_signatures" json:"authorized_signatures"`

	// ESC1 through ESC8 are boolean flags indicating which known ESC
	// misconfiguration conditions are intentionally present on this template.
	ESC1 bool `yaml:"esc1" json:"esc1"`
	ESC2 bool `yaml:"esc2" json:"esc2"`
	ESC3 bool `yaml:"esc3" json:"esc3"`
	ESC4 bool `yaml:"esc4" json:"esc4"`
	ESC5 bool `yaml:"esc5" json:"esc5"`
	ESC6 bool `yaml:"esc6" json:"esc6"`
	ESC7 bool `yaml:"esc7" json:"esc7"`
	ESC8 bool `yaml:"esc8" json:"esc8"`
}

// Delegation represents a Kerberos delegation configuration on a computer account.
type Delegation struct {
	// Computer is the hostname of the machine that has delegation enabled.
	Computer string `yaml:"computer" json:"computer"`

	// Type is the delegation model: "unconstrained", "constrained", or
	// "resource-based".
	Type string `yaml:"type" json:"type"`

	// AllowedTo lists the SPNs the computer is permitted to delegate to.
	// Only meaningful when Type is "constrained".
	AllowedTo []string `yaml:"allowed_to" json:"allowed_to"`
}

// FileSystem is the simulated directory tree. Keys are absolute UNIX-style
// paths; values are slices of child entry names (files or subdirectory names).
// The engine uses this to respond to directory-listing commands.
type FileSystem map[string][]string

// FileContents maps absolute paths to their readable string content.
// Only entries present in this map are accessible via simulated file-read tools.
type FileContents map[string]string

// ToolConfig describes the tool surface exposed to the student.
type ToolConfig struct {
	// Tools lists the binary names available in the simulated shell
	// (e.g. ["Rubeus", "mimikatz", "PowerView", "net"]).
	Tools []string `yaml:"tools" json:"tools"`

	// Overrides maps binary names to per-subcommand constraint definitions.
	// A missing entry means the binary is unrestricted.
	Overrides map[string]ToolOverride `yaml:"overrides" json:"overrides"`
}

// ToolOverride defines per-subcommand execution constraints for a binary.
// Keys in the map are subcommand strings (e.g. "dcsync", "asktgt").
type ToolOverride map[string]SubcommandConstraint

// SubcommandConstraint controls whether a specific subcommand can execute
// and what session state must be present before it is permitted.
type SubcommandConstraint struct {
	// Blocked unconditionally prevents this subcommand from executing,
	// returning a simulated "access denied" or "tool not available" error.
	Blocked bool `yaml:"blocked" json:"blocked"`

	// Requires lists state condition strings that must all evaluate to true
	// before this subcommand is permitted. Format is engine-defined
	// (e.g. "loot.tickets[TGT:administrator]").
	Requires []string `yaml:"requires" json:"requires"`
}

// WinCondition defines a single lab objective that must be satisfied.
type WinCondition struct {
	// Type is the evaluation strategy:
	//   - "command_executed": a specific binary+subcommand was run.
	//   - "state_check": a key in session state matches a value.
	//   - "loot_check": a specific piece of loot was acquired.
	//   - "file_read": a specific file path was read.
	Type string `yaml:"type" json:"type"`

	// Match is the string to compare against (command string, state value,
	// loot value, or file path depending on Type).
	Match string `yaml:"match" json:"match"`

	// Key is the state or loot key to evaluate (used by "state_check" and
	// "loot_check" types).
	Key string `yaml:"key" json:"key"`

	// Value is the expected value for "state_check" conditions.
	Value string `yaml:"value" json:"value"`

	// Label is the human-readable objective description shown to the student.
	Label string `yaml:"label" json:"label"`
}

// TimeConfig controls session lifecycle and authentication prompt timing.
type TimeConfig struct {
	// SessionTimeout is the number of seconds of inactivity before the
	// session is automatically terminated.
	SessionTimeout int `yaml:"session_timeout" json:"session_timeout"`

	// TokenTTL is the simulated lifetime of Kerberos tickets and OAuth
	// tokens in seconds.
	TokenTTL int `yaml:"token_ttl" json:"token_ttl"`

	// MFAPromptAfter is the number of seconds after which the simulated
	// environment will trigger an MFA re-prompt event.
	MFAPromptAfter int `yaml:"mfa_prompt_after" json:"mfa_prompt_after"`
}

// HintConfig associates a hint text with the state condition that unlocks it.
type HintConfig struct {
	// Trigger is a state condition expression. When it evaluates true the
	// hint becomes available to the student.
	Trigger string `yaml:"trigger" json:"trigger"`

	// Text is the hint content displayed to the student.
	Text string `yaml:"text" json:"text"`
}

// StartingPoint defines the initial context for "drop-in" mode labs.
type StartingPoint struct {
	// Machine is the hostname the student begins on.
	Machine string `yaml:"machine" json:"machine"`

	// User is the sAMAccountName the student authenticates as initially.
	User string `yaml:"user" json:"user"`

	// Privilege is the access level at start: "user", "local-admin",
	// or "SYSTEM".
	Privilege string `yaml:"privilege" json:"privilege"`
}

// MailConfig controls the simulated phishing subsystem.
type MailConfig struct {
	// VictimResponseDelay is the number of seconds the engine waits after
	// a phishing payload is "sent" before marking the victim as having
	// clicked and triggering the victim_authenticated push event.
	VictimResponseDelay int `yaml:"victim_response_delay" json:"victim_response_delay"`
}