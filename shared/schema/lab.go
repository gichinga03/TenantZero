package schema

type Lab struct {
	LabID         string         `yaml:"lab_id" json:"labId"`
	Domain        Domain         `yaml:"domain" json:"domain"`
	Forests       []Forest       `yaml:"forests" json:"forests"`
	Trusts        []Trust        `yaml:"trusts" json:"trusts"`
	Hosts         []Host         `yaml:"hosts" json:"hosts"`
	Identities    []Identity     `yaml:"identities" json:"identities"`
	Relationships []Relationship `yaml:"relationships" json:"relationships"`
	Users         []User         `yaml:"users" json:"users"`
	Groups        []Group        `yaml:"groups" json:"groups"`
	Computers     []Computer     `yaml:"computers" json:"computers"`
	ACLs          []ACL          `yaml:"acls" json:"acls"`
	TrustsList    []TrustEntry   `yaml:"trusts_list" json:"trustsList"`
	CertTemplates []CertTemplate `yaml:"cert_templates" json:"certTemplates"`
	Delegations   []Delegation   `yaml:"delegations" json:"delegations"`
	FileSystem    FileSystem     `yaml:"filesystem" json:"filesystem"`
	FileContents  FileContents   `yaml:"file_contents" json:"fileContents"`
	Tools         ToolConfig     `yaml:"tools" json:"tools"`
	WinConditions []WinCondition `yaml:"win_conditions" json:"winConditions"`
	Time          TimeConfig     `yaml:"time" json:"time"`
	Hints         []HintConfig   `yaml:"hints" json:"hints"`
	StartingPoint StartingPoint  `yaml:"starting_point" json:"startingPoint"`
	Mail          MailConfig     `yaml:"mail" json:"mail"`
}

type Domain struct {
	Name       string `yaml:"name" json:"name"`
	NetBIOS    string `yaml:"netbios" json:"netbios"`
	Functional string `yaml:"functional_level" json:"functionalLevel"`
	DCHostname string `yaml:"dc_hostname" json:"dcHostname"`
	DCIP       string `yaml:"dc_ip" json:"dcIp"`
}

type Forest struct {
	ForestName string   `yaml:"forest_name" json:"forestName"`
	Domains    []string `yaml:"domains" json:"domains"`
	Subnets    []string `yaml:"subnets" json:"subnets"`
}

type Trust struct {
	SourceForest     string `yaml:"source_forest" json:"sourceForest"`
	TargetForest     string `yaml:"target_forest" json:"targetForest"`
	Direction        string `yaml:"direction" json:"direction"`
	TrustType        string `yaml:"trust_type" json:"trustType"`
	SIDFiltering     bool   `yaml:"sid_filtering" json:"sidFiltering"`
	FilteringEnabled bool   `yaml:"filtering_enabled" json:"filteringEnabled"`
}

type Host struct {
	Hostname      string   `yaml:"hostname" json:"hostname"`
	Forest        string   `yaml:"forest" json:"forest"`
	IP            string   `yaml:"ip" json:"ip"`
	Role          string   `yaml:"role" json:"role"`
	InScope       bool     `yaml:"in_scope" json:"inScope"`
	StagedTools   []string `yaml:"staged_tools" json:"stagedTools"`
	ReachableFrom []string `yaml:"reachable_from" json:"reachableFrom"`
	OS            string   `yaml:"os" json:"os"`
}

type Identity struct {
	Account  string   `yaml:"account" json:"account"`
	Forest   string   `yaml:"forest" json:"forest"`
	Groups   []string `yaml:"groups" json:"groups"`
	SPNs     []string `yaml:"spns" json:"spns"`
	Password string   `yaml:"password" json:"password"`
	NTHash   string   `yaml:"nt_hash" json:"ntHash"`
}

type Relationship struct {
	Type            string `yaml:"type" json:"type"`
	From            string `yaml:"from" json:"from"`
	To              string `yaml:"to" json:"to"`
	RequiresAccount string `yaml:"requires_account" json:"requiresAccount"`
	TargetAccount   string `yaml:"target_account" json:"targetAccount"`
	TrustRef        string `yaml:"trust_ref" json:"trustRef"`
}

type User struct {
	SAMAccountName          string   `yaml:"samaccountname" json:"samAccountName"`
	DisplayName             string   `yaml:"displayname" json:"displayName"`
	Password                string   `yaml:"password" json:"password"`
	NTHash                  string   `yaml:"nt_hash" json:"ntHash"`
	Groups                  []string `yaml:"groups" json:"groups"`
	SPNs                    []string `yaml:"spns" json:"spns"`
	Enabled                 bool     `yaml:"enabled" json:"enabled"`
	AdminCount              bool     `yaml:"admincount" json:"adminCount"`
	UnconstrainedDelegation bool     `yaml:"unconstrained_delegation" json:"unconstrainedDelegation"`
	PasswordNeverExpires    bool     `yaml:"password_never_expires" json:"passwordNeverExpires"`
}

type Group struct {
	Name    string   `yaml:"name" json:"name"`
	Members []string `yaml:"members" json:"members"`
}

type Computer struct {
	Name                    string `yaml:"name" json:"name"`
	DNSHostname             string `yaml:"dns_hostname" json:"dnsHostname"`
	UnconstrainedDelegation bool   `yaml:"unconstrained_delegation" json:"unconstrainedDelegation"`
	OperatingSystem         string `yaml:"operating_system" json:"operatingSystem"`
}

type ACL struct {
	From   string `yaml:"from" json:"from"`
	To     string `yaml:"to" json:"to"`
	Rights string `yaml:"rights" json:"rights"`
}

type TrustEntry struct {
	Source    string `yaml:"source" json:"source"`
	Target    string `yaml:"target" json:"target"`
	Direction string `yaml:"direction" json:"direction"`
	TrustType string `yaml:"trust_type" json:"trustType"`
}

type CertTemplate struct {
	Name                    string   `yaml:"name" json:"name"`
	SubjectAltNameEnabled   bool     `yaml:"subject_alt_name_enabled" json:"subjectAltNameEnabled"`
	EnrolleeSuppliesSAN     bool     `yaml:"enrollee_supplies_san" json:"enrolleeSuppliesSAN"`
	LowPrivEnroll           bool     `yaml:"low_priv_enroll" json:"lowPrivEnroll"`
	ManagerApprovalRequired bool     `yaml:"manager_approval_required" json:"managerApprovalRequired"`
	ESC1                    bool     `yaml:"esc1" json:"esc1"`
	ESC2                    bool     `yaml:"esc2" json:"esc2"`
	ESC3                    bool     `yaml:"esc3" json:"esc3"`
	ESC4                    bool     `yaml:"esc4" json:"esc4"`
	ESC8                    bool     `yaml:"esc8" json:"esc8"`
	AllowedEnrollees        []string `yaml:"allowed_enrollees" json:"allowedEnrollees"`
}

type Delegation struct {
	Account        string   `yaml:"account" json:"account"`
	Type           string   `yaml:"type" json:"type"`
	AllowedTargets []string `yaml:"allowed_targets" json:"allowedTargets"`
}

type FileSystem map[string][]string
type FileContents map[string]FileContent

type FileContent struct {
	Data   string `yaml:"data" json:"data"`
	DAOnly bool   `yaml:"da_only" json:"daOnly"`
}

type ToolConfig struct {
	Available []string                `yaml:"available" json:"available"`
	Overrides map[string]ToolOverride `yaml:"overrides" json:"overrides"`
}

type ToolOverride struct {
	Disabled bool   `yaml:"disabled" json:"disabled"`
	Reason   string `yaml:"reason" json:"reason"`
}

type WinCondition struct {
	ID          string            `yaml:"id" json:"id"`
	Description string            `yaml:"description" json:"description"`
	Type        string            `yaml:"type" json:"type"`
	Params      map[string]string `yaml:"params" json:"params"`
	Points      int               `yaml:"points" json:"points"`
	Required    bool              `yaml:"required" json:"required"`
}

type TimeConfig struct {
	SessionTimeoutMinutes int `yaml:"session_timeout_minutes" json:"sessionTimeoutMinutes"`
	TokenTTLMinutes       int `yaml:"token_ttl_minutes" json:"tokenTTLMinutes"`
}

type HintConfig struct {
	ID                string `yaml:"id" json:"id"`
	MinElapsedMinutes int    `yaml:"min_elapsed_minutes" json:"minElapsedMinutes"`
	Text              string `yaml:"text" json:"text"`
}

type StartingPoint struct {
	Machine   string `yaml:"machine" json:"machine"`
	User      string `yaml:"user" json:"user"`
	Privilege string `yaml:"privilege" json:"privilege"`
}

type MailConfig struct {
	VictimEmail         string `yaml:"victim_email" json:"victimEmail"`
	VictimResponseDelay int    `yaml:"victim_response_delay" json:"victimResponseDelay"`
	LinkPhish           bool   `yaml:"link_phish" json:"linkPhish"`
}