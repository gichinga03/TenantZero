// Package ad implements the providers.Provider interface for Active Directory
// lab environments. It wraps an immutable *schema.Lab and a mutable
// *schema.SessionState, exposing a clean query surface to all engine tools.
//
// Design constraints:
//   - Zero mutations — all methods are read-only. State writes go through
//     the engine's SideEffect apply loop, never through the provider.
//   - Case-insensitive matching throughout — AD is case-insensitive and labs
//     should behave the same way regardless of how a tool formats identifiers.
//   - No raw map access from tool code — tools call methods, never reach into
//     SessionState or Lab directly.
package ad

import (
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// abusableRights is the set of ACL rights considered non-default and
// exploitable. Anything outside this set is filtered by GetAbusableACLs.
//
// Azure equivalent: this would instead be a set of privileged Entra role
// display names and sensitive Azure RBAC role definitions.
var abusableRights = map[string]bool{
	"GenericAll":          true,
	"GenericWrite":        true,
	"WriteDACL":           true,
	"WriteOwner":          true,
	"AllExtendedRights":   true,
	"Owns":                true,
	"ForceChangePassword": true,
}

// replicationRights is the set of ACL rights on the domain object that
// grant MS-DRSR replication capability (DCSync).
var replicationRights = map[string]bool{
	"AllExtendedRights": true,
	"GenericAll":        true,
}

// domainAdminsGroup is the canonical name of the privileged group checked
// by HasDomainAdmin. Comparison is case-insensitive.
const domainAdminsGroup = "domain admins"

// ADProvider implements providers.Provider for an Active Directory lab.
// It is safe for concurrent reads — no method writes to lab or state.
type ADProvider struct {
	lab   *schema.Lab
	state *schema.SessionState
}

// New constructs an ADProvider. Both lab and state must be non-nil.
// lab is treated as immutable; state is read but never written.
func New(lab *schema.Lab, state *schema.SessionState) *ADProvider {
	return &ADProvider{lab: lab, state: state}
}

// normalize lowercases and trims s for case-insensitive comparison.
func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// =============================================================================
// Users
// =============================================================================

// GetUsers returns all users in the lab domain.
func (p *ADProvider) GetUsers() []schema.User {
	out := make([]schema.User, len(p.lab.Users))
	copy(out, p.lab.Users)
	return out
}

// GetUser returns the user whose Username matches username (case-insensitive).
func (p *ADProvider) GetUser(username string) (schema.User, bool) {
	key := normalize(username)
	for _, u := range p.lab.Users {
		if normalize(u.Username) == key {
			return u, true
		}
	}
	return schema.User{}, false
}

// GetUsersWithSPN returns users that carry a ServicePrincipalName.
// These are candidates for Kerberoasting via Rubeus or Impacket GetUserSPNs.
func (p *ADProvider) GetUsersWithSPN() []schema.User {
	var out []schema.User
	for _, u := range p.lab.Users {
		if len(u.SPN) > 0 {
			out = append(out, u)
		}
	}
	return out
}

// GetUsersWithAdminCount returns users where AdminCount is true.
// Mirrors the adminCount attribute set by SDProp on privileged accounts.
func (p *ADProvider) GetUsersWithAdminCount() []schema.User {
	var out []schema.User
	for _, u := range p.lab.Users {
		if u.AdminCount > 0 {
			out = append(out, u)
		}
	}
	return out
}

// GetUserGroups returns all groups the user is a direct member of.
// Membership is determined by scanning each group's Members slice.
// Does not recurse — transitive membership requires walking the result.
func (p *ADProvider) GetUserGroups(username string) []schema.Group {
	key := normalize(username)
	var out []schema.Group
	for _, g := range p.lab.Groups {
		for _, m := range g.Members {
			if normalize(m) == key {
				out = append(out, g)
				break
			}
		}
	}
	return out
}

// =============================================================================
// Groups
// =============================================================================

// GetGroups returns all domain groups.
func (p *ADProvider) GetGroups() []schema.Group {
	out := make([]schema.Group, len(p.lab.Groups))
	copy(out, p.lab.Groups)
	return out
}

// GetGroup returns the group matching name (case-insensitive).
func (p *ADProvider) GetGroup(name string) (schema.Group, bool) {
	key := normalize(name)
	for _, g := range p.lab.Groups {
		if normalize(g.Name) == key {
			return g, true
		}
	}
	return schema.Group{}, false
}

// GetGroupMembers returns all users that are direct members of the named group.
// Group membership strings are matched against User.Username case-insensitively.
// Nested groups are not resolved — only user objects are returned.
func (p *ADProvider) GetGroupMembers(name string) []schema.User {
	g, ok := p.GetGroup(name)
	if !ok {
		return nil
	}

	// Build a lookup set for the group's member strings.
	memberSet := make(map[string]bool, len(g.Members))
	for _, m := range g.Members {
		memberSet[normalize(m)] = true
	}

	var out []schema.User
	for _, u := range p.lab.Users {
		if memberSet[normalize(u.Username)] {
			out = append(out, u)
		}
	}
	return out
}

// =============================================================================
// Computers
// =============================================================================

// GetComputers returns all computers in the domain.
func (p *ADProvider) GetComputers() []schema.Computer {
	out := make([]schema.Computer, len(p.lab.Computers))
	copy(out, p.lab.Computers)
	return out
}

// GetComputer returns the computer matching hostname (case-insensitive).
func (p *ADProvider) GetComputer(hostname string) (schema.Computer, bool) {
	key := normalize(hostname)
	for _, c := range p.lab.Computers {
		if normalize(c.Hostname) == key {
			return c, true
		}
	}
	return schema.Computer{}, false
}

// GetDomainControllers returns all computers where IsDC is true.
func (p *ADProvider) GetDomainControllers() []schema.Computer {
	var out []schema.Computer
	for _, c := range p.lab.Computers {
		for _, r := range c.Roles {
			if strings.EqualFold(r, "DomainController") {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// GetLocalAdmins returns the LocalAdmins slice for the given hostname.
// Returns nil if the hostname is not found.
func (p *ADProvider) GetLocalAdmins(hostname string) []string {
	c, ok := p.GetComputer(hostname)
	if !ok {
		return nil
	}
	out := make([]string, len(c.LocalAdmins))
	copy(out, c.LocalAdmins)
	return out
}

// =============================================================================
// ACLs
// =============================================================================

// GetACLs returns all ACL entries in the lab.
func (p *ADProvider) GetACLs() []schema.ACL {
	out := make([]schema.ACL, len(p.lab.ACLs))
	copy(out, p.lab.ACLs)
	return out
}

// GetACLsFor returns ACL entries where Target matches the given name
// (case-insensitive).
func (p *ADProvider) GetACLsFor(target string) []schema.ACL {
	key := normalize(target)
	var out []schema.ACL
	for _, a := range p.lab.ACLs {
		if normalize(a.To) == key {
			out = append(out, a)
		}
	}
	return out
}

// GetACLsFrom returns ACL entries where Principal matches the given name
// (case-insensitive).
func (p *ADProvider) GetACLsFrom(principal string) []schema.ACL {
	key := normalize(principal)
	var out []schema.ACL
	for _, a := range p.lab.ACLs {
		if normalize(a.From) == key {
			out = append(out, a)
		}
	}
	return out
}

// GetAbusableACLs returns ACL entries with rights considered non-default and
// exploitable. Filtered to the set defined in abusableRights.
//
// Azure equivalent: would scan Entra role assignments and Azure RBAC grants
// for privileged roles (Owner, Contributor, User Access Administrator, etc.)
// rather than AD DACL right strings.
func (p *ADProvider) GetAbusableACLs() []schema.ACL {
	var out []schema.ACL
	for _, a := range p.lab.ACLs {
		if abusableRights[a.Right] {
			out = append(out, a)
		}
	}
	return out
}

// =============================================================================
// Trusts
// =============================================================================

// GetTrusts returns all trust relationships defined in the lab.
func (p *ADProvider) GetTrusts() []schema.Trust {
	out := make([]schema.Trust, len(p.lab.Trusts))
	copy(out, p.lab.Trusts)
	return out
}

// =============================================================================
// Delegation
// =============================================================================

// GetDelegations returns all Delegation structs across all computers in the lab.
func (p *ADProvider) GetDelegations() []schema.Delegation {
	return p.lab.Delegations
}

// GetUnconstrainedDelegation returns computers configured for unconstrained
// Kerberos delegation. Domain controllers are excluded — they carry this flag
// by default and are not meaningful targets for printer-bug TGT capture.
//
// Azure equivalent: unconstrained delegation has no direct Entra parallel;
// the closest concept is Managed Identity with broad resource access.
func (p *ADProvider) GetUnconstrainedDelegation() []schema.Computer {
	var out []schema.Computer
	dcSet := make(map[string]bool)
	for _, c := range p.lab.Computers {
		for _, r := range c.Roles {
			if strings.EqualFold(r, "DomainController") {
				dcSet[c.Hostname] = true
			}
		}
	}
	for _, d := range p.lab.Delegations {
		if strings.EqualFold(d.Type, "unconstrained") {
			for _, c := range p.lab.Computers {
				if strings.EqualFold(c.Hostname, d.Computer) && !dcSet[c.Hostname] {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// GetConstrainedDelegation returns Delegation entries where Type is
// "constrained" or "resource-based". The AllowedServices slice drives
// S4U2Proxy abuse enumeration in the Rubeus tool.
func (p *ADProvider) GetConstrainedDelegation() []schema.Delegation {
	var out []schema.Delegation
	for _, d := range p.lab.Delegations {
		t := normalize(d.Type)
		if t == "constrained" || t == "resource-based" {
			out = append(out, d)
		}
	}
	return out
}

// =============================================================================
// Certificates
// =============================================================================

// GetCertTemplates returns all certificate templates defined in the lab.
func (p *ADProvider) GetCertTemplates() []schema.CertTemplate {
	out := make([]schema.CertTemplate, len(p.lab.CertTemplates))
	copy(out, p.lab.CertTemplates)
	return out
}

// GetVulnerableCertTemplates returns templates where at least one ESC flag is
// true. ESC1–ESC8 are stored as individual booleans for O(1) field access.
//
// Azure equivalent: would scan Entra certificate-based authentication configs
// and app registration credential policies rather than ADCS templates.
func (p *ADProvider) GetVulnerableCertTemplates() []schema.CertTemplate {
	var out []schema.CertTemplate
	for _, ct := range p.lab.CertTemplates {
		if ct.ESC1 || ct.ESC2 || ct.ESC3 || ct.ESC4 ||
			ct.ESC5 || ct.ESC6 || ct.ESC7 || ct.ESC8 {
			out = append(out, ct)
		}
	}
	return out
}

// =============================================================================
// Credential validation
// =============================================================================

// ValidateCredentials returns true if username:password is a valid pair.
//
// Resolution order:
//  1. Lab users — checks User.Password (plaintext, set in YAML).
//  2. SessionState.Loot.PlaintextCreds — creds captured or cracked mid-session
//     (e.g. from a file on the virtual FS, a Mimikatz output, CME output).
//
// Both username and password comparisons are case-insensitive for usernames;
// passwords are compared as-is (AD passwords are case-sensitive).
//
// Azure equivalent: would validate an access_token or refresh_token stored in
// Loot rather than a plaintext password, since Entra auth is token-based.
// A "password" check in Entra context means checking for valid ROPC flow creds
// against a Loot entry — same structure, different semantic.
func (p *ADProvider) ValidateCredentials(username, password string) bool {
	key := normalize(username)

	// Check lab-defined users first.
	for _, u := range p.lab.Users {
		if normalize(u.Username) == key && u.Password == password {
			return true
		}
	}

	// Check plaintext creds captured during the session.
	if pw, ok := p.state.Loot.PlaintextCreds[key]; ok && pw == password {
		return true
	}

	return false
}

// ValidateNTLM returns true if username:NTHash is a valid pair in the session's
// loot. NTLM hashes are captured by tools like Mimikatz sekurlsa::logonpasswords
// or Impacket secretsdump and stored in SessionState.Loot.NTLMHashes.
//
// Hash comparison is case-insensitive (NT hashes are hex strings, mixed-case
// is common from different dumping tools).
//
// Azure equivalent: would not exist in this form — Entra has no NTLM. The
// equivalent loot validation would be checking for PRT (Primary Refresh Token)
// or OAuth token entries in Loot.
func (p *ADProvider) ValidateNTLM(username, hash string) bool {
	key := normalize(username)
	hashKey := normalize(hash)

	if h, ok := p.state.Loot.NTLMHashes[key]; ok && normalize(h) == hashKey {
		return true
	}
	return false
}

// HasLocalAdmin returns true if username has local administrator access on
// hostname. Two paths satisfy this:
//  1. The computer's LocalAdmins slice contains username (direct grant, or
//     a group name that includes the user — group expansion is not done here;
//     tools that need group-aware local admin checks should resolve membership
//     first via GetGroupMembers).
//  2. HasDomainAdmin(username) is true — DA implies local admin everywhere
//     under default GPO settings.
func (p *ADProvider) HasLocalAdmin(username, hostname string) bool {
	if p.HasDomainAdmin(username) {
		return true
	}

	c, ok := p.GetComputer(hostname)
	if !ok {
		return false
	}

	key := normalize(username)
	for _, a := range c.LocalAdmins {
		if normalize(a) == key {
			return true
		}
	}
	return false
}

// HasDomainAdmin returns true if username is a member of "Domain Admins"
// through any chain of nested group membership.
//
// Algorithm: BFS over the group graph starting from "Domain Admins".
// Visited set prevents infinite loops on cyclic group configs (shouldn't exist
// in a valid YAML, but we defend against it).
//
// Azure equivalent: would check for Global Administrator role assignment in
// Entra ID rather than DA group membership. The BFS isn't needed there —
// Entra roles are not nested.
func (p *ADProvider) HasDomainAdmin(username string) bool {
	userKey := normalize(username)

	// BFS queue holds group names to visit.
	visited := make(map[string]bool)
	queue := []string{domainAdminsGroup}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		g, ok := p.GetGroup(current)
		if !ok {
			continue
		}

		for _, m := range g.Members {
			mKey := normalize(m)

			// Direct user membership check.
			if mKey == userKey {
				return true
			}

			// If the member name matches a group, enqueue it for expansion.
			// We distinguish groups from users by checking the group index.
			// A member string that matches a group name is treated as a nested group.
			if _, isGroup := p.GetGroup(m); isGroup {
				if !visited[mKey] {
					queue = append(queue, mKey)
				}
			}
		}
	}

	return false
}

// HasReplicationRights returns true if username holds AllExtendedRights or
// GenericAll over the domain object in the lab's ACL list.
//
// In real AD, the domain object target is the domain DN. In the lab, ACL
// entries targeting the domain name (FQDN) represent domain-object rights.
// Tools use this to gate DCSync availability without requiring DA.
//
// Azure equivalent: would check for Directory.ReadWrite.All or similar MS Graph
// application permissions, or Global Administrator / Privileged Role Admin
// assignments, since Entra has no MS-DRSR equivalent.
func (p *ADProvider) HasReplicationRights(username string) bool {
	userKey := normalize(username)
	domainKey := normalize(p.lab.Domains[0].Name)

	for _, a := range p.lab.ACLs {
		if normalize(a.From) != userKey {
			continue
		}
		if normalize(a.To) != domainKey {
			continue
		}
		if replicationRights[a.Right] {
			return true
		}
	}
	return false
}

// =============================================================================
// Domain metadata
// =============================================================================

// DomainName returns the FQDN of the primary lab domain (e.g. "corp.local").
func (p *ADProvider) DomainName() string {
	return p.lab.Domains[0].Name
}

// DomainController returns the hostname of the first DC found in the computer
// list. Returns an empty string if no DC is defined in the lab (invalid YAML).
func (p *ADProvider) DomainController() string {
	for _, c := range p.lab.Computers {
		for _, r := range c.Roles {
			if strings.EqualFold(r, "DomainController") {
				return c.Hostname
			}
		}
	}
	return ""
}