package schema

// Engagement is the top-level structure parsed from engagement.yaml.
// It is the fictional Rules of Engagement / scoping document presented to
// the student at session start to frame the lab as a realistic client engagement.
type Engagement struct {
	// LabID ties this engagement document to a specific Lab definition.
	LabID string `yaml:"lab_id" json:"lab_id"`

	// Client describes the fictional organisation being tested.
	Client Client `yaml:"client" json:"client"`

	// Type is the engagement classification, e.g. "Internal Red Team",
	// "Assumed Breach", or "Internal Pentest".
	Type string `yaml:"type" json:"type"`

	// Mode mirrors Lab.Mode and is included here so the engagement document
	// is self-contained for display purposes.
	Mode string `yaml:"mode" json:"mode"`

	// StartingPoint defines where the student begins in "drop-in" engagements.
	// Included in the engagement doc to give the student their initial brief.
	StartingPoint StartingPoint `yaml:"starting_point" json:"starting_point"`

	// Scope defines what the student is and is not authorised to test.
	Scope Scope `yaml:"scope" json:"scope"`

	// RulesOfEngagement contains the engagement constraints and emergency
	// contact information.
	RulesOfEngagement RulesOfEngagement `yaml:"rules_of_engagement" json:"rules_of_engagement"`

	// EstimatedDuration is a human-readable time estimate (e.g. "4–6 hours").
	EstimatedDuration string `yaml:"estimated_duration" json:"estimated_duration"`

	// Window describes the authorised testing window (e.g. "24/7" or
	// "Mon–Fri 08:00–18:00 EAT").
	Window string `yaml:"window" json:"window"`
}

// Client describes the fictional organisation that has commissioned the
// simulated engagement.
type Client struct {
	// Name is the company name (e.g. "Contoso Ltd").
	Name string `yaml:"name" json:"name"`

	// Industry is the sector the client operates in (e.g. "Financial Services").
	Industry string `yaml:"industry" json:"industry"`

	// Fictional is always true in TenantZero engagement documents. It is
	// surfaced explicitly so lab authors cannot accidentally publish a
	// document that could be mistaken for a real engagement.
	Fictional bool `yaml:"fictional" json:"fictional"`
}

// Scope defines the boundary of the simulated engagement.
type Scope struct {
	// InScope lists the systems, subnets, or object classes the student is
	// authorised to target (e.g. ["10.10.10.0/24", "corp.local"]).
	InScope []string `yaml:"in_scope" json:"in_scope"`

	// OutOfScope lists anything explicitly excluded from testing.
	OutOfScope []string `yaml:"out_of_scope" json:"out_of_scope"`
}

// RulesOfEngagement contains the procedural constraints and escalation
// contacts for the simulated engagement.
type RulesOfEngagement struct {
	// Rules is an ordered list of plain-language constraints
	// (e.g. "Do not cause denial of service conditions").
	Rules []string `yaml:"rules" json:"rules"`

	// EmergencyContacts lists the fictional contacts the student should
	// notify in a simulated critical-finding scenario.
	EmergencyContacts []Contact `yaml:"emergency_contacts" json:"emergency_contacts"`
}

// Contact is a named person in the engagement's emergency contact list.
type Contact struct {
	// Name is the contact's full name.
	Name string `yaml:"name" json:"name"`

	// Role is their position within the fictional client organisation
	// (e.g. "CISO", "SOC Lead").
	Role string `yaml:"role" json:"role"`

	// Email is the fictional contact email address.
	Email string `yaml:"email" json:"email"`

	// Phone is the fictional contact phone number.
	Phone string `yaml:"phone" json:"phone"`
}