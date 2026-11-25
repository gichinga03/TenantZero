package schema

import "time"

type Beacon struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parentId"`
	Host        string    `json:"host"`
	Identity    string    `json:"identity"`
	Active      bool      `json:"active"`
	CheckedInAt time.Time `json:"checkedInAt"`
	Tasks       []Task    `json:"tasks"`
}

type Task struct {
	ID          string    `json:"id"`
	Input       string    `json:"input"`
	Output      string    `json:"output"`
	IssuedAt    time.Time `json:"issuedAt"`
	CompletedAt time.Time `json:"completedAt"`
	Status      string    `json:"status"` // "pending", "running", "complete", "error"
}

type SessionState struct {
	LabID               string              `json:"labId"`
	UserID              string              `json:"userId"`
	ActiveHost          string              `json:"activeHost"`
	ActiveUser          string              `json:"activeUser"`
	ActiveBeaconID      string              `json:"activeBeaconId"`
	CWD                 string              `json:"cwd"`
	BeaconTree          []Beacon            `json:"beaconTree"`
	GrantedACLs         map[string][]string `json:"grantedAcls"`
	Loot                []Loot              `json:"loot"`
	Tickets             []Ticket            `json:"tickets"`
	Tokens              []Token             `json:"tokens"`
	CompletedObjectives []CompletedObjective `json:"completedObjectives"`
	Journal             []JournalEntry      `json:"journal"`
	UsedHints           []string            `json:"usedHints"`
	StartedAt           time.Time           `json:"startedAt"`
	LastActivityAt      time.Time           `json:"lastActivityAt"`
}

type Loot struct {
	Type       string    `json:"type"`
	Value      string    `json:"value"`
	Source     string    `json:"source"`
	CapturedAt time.Time `json:"capturedAt"`
}

type Ticket struct {
	TicketType string    `json:"ticketType"` // "TGT", "TGS", "ST"
	Identity   string    `json:"identity"`
	Target     string    `json:"target"`
	B64        string    `json:"b64"`
	IssuedAt   time.Time `json:"issuedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type Token struct {
	Username string    `json:"username"`
	Domain   string    `json:"domain"`
	Fake     string    `json:"fake"`
	IssuedAt time.Time `json:"issuedAt"`
}

type CompletedObjective struct {
	ID          string    `json:"id"`
	CompletedAt time.Time `json:"completedAt"`
	Points      int       `json:"points"`
}

type JournalEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	Identity  string    `json:"identity"`
	RawInput  string    `json:"rawInput"`
	Output    string    `json:"output"`
}