// Package providers defines the Provider interface that all engine tools use
// to query tenant state. No tool ever reads *schema.SessionState or *schema.Lab
// directly — everything goes through this interface.
//
// The indirection exists for one reason: Azure. When the Azure provider lands,
// it implements this same interface. Tools written against Provider require zero
// changes to support both AD and Entra/Azure targets.
package providers

import (
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// Provider is the read-only query surface exposed to every tool in the engine.
// Implementations must be safe for concurrent reads — the engine may dispatch
// multiple tool calls against the same session simultaneously.
//
// Credential validation methods (ValidateCredentials, ValidateNTLM) also read
// from SessionState.Loot so that cracked/captured creds are usable mid-session
// without restarting the lab.
type Provider interface {
	// -------------------------------------------------------------------------
	// Users
	// -------------------------------------------------------------------------

	// GetUsers returns all domain users defined in the lab.
	GetUsers() []schema.User

	// GetUser returns the user matching username (case-insensitive).
	// The second return value is false if no match is found.
	GetUser(username string) (schema.User, bool)

	// GetUsersWithSPN returns users that have a ServicePrincipalName set.
	// These are Kerberoast candidates.
	GetUsersWithSPN() []schema.User

	// GetUsersWithAdminCount returns users where AdminCount == true.
	// Mirrors the adminCount attribute written by SDProp in real AD.
	GetUsersWithAdminCount() []schema.User

	// GetUserGroups returns all groups the user is a direct member of.
	// Does not recurse — callers that need transitive membership should
	// walk the returned groups themselves or use HasDomainAdmin.
	GetUserGroups(username string) []schema.Group

	// -------------------------------------------------------------------------
	// Groups
	// -------------------------------------------------------------------------

	// GetGroups returns all domain groups.
	GetGroups() []schema.Group

	// GetGroup returns the group matching name (case-insensitive).
	GetGroup(name string) (schema.Group, bool)

	// GetGroupMembers returns all users that are direct members of the group.
	// Does not recurse into nested groups.
	GetGroupMembers(name string) []schema.User

	// -------------------------------------------------------------------------
	// Computers
	// -------------------------------------------------------------------------

	// GetComputers returns all computers in the domain.
	GetComputers() []schema.Computer

	// GetComputer returns the computer matching hostname (case-insensitive).
	GetComputer(hostname string) (schema.Computer, bool)

	// GetDomainControllers returns computers where IsDC == true.
	GetDomainControllers() []schema.Computer

	// GetHost returns the host matching hostname or IP (case-insensitive).
	GetHost(hostname string) (schema.Host, bool)

	// GetAllHosts returns all hosts defined in the lab.
	GetAllHosts() []schema.Host

	// GetLocalAdmins returns the list of principals (users or groups) that
	// have local administrator access on the given hostname.
	GetLocalAdmins(hostname string) []string

	// -------------------------------------------------------------------------
	// ACLs
	// -------------------------------------------------------------------------

	// GetACLs returns all ACL entries in the lab.
	GetACLs() []schema.ACL

	// GetACLsFor returns all ACL entries where Target matches the given name
	// (case-insensitive). Used to enumerate rights over a specific object.
	GetACLsFor(target string) []schema.ACL

	// GetACLsFrom returns all ACL entries where Principal matches the given
	// name (case-insensitive). Used to enumerate what a principal can do.
	GetACLsFrom(principal string) []schema.ACL

	// GetAbusableACLs returns ACL entries that represent non-default,
	// exploitable rights. Filtered to: GenericAll, GenericWrite, WriteDACL,
	// WriteOwner, AllExtendedRights, Owns, ForceChangePassword.
	//
	// Azure equivalent: would filter Entra role assignments and resource-level
	// RBAC grants rather than AD DACL entries.
	GetAbusableACLs() []schema.ACL

	// -------------------------------------------------------------------------
	// Trusts
	// -------------------------------------------------------------------------

	// GetTrusts returns all domain/forest trust relationships defined in the lab.
	GetTrusts() []schema.Trust

	// -------------------------------------------------------------------------
	// Delegation
	// -------------------------------------------------------------------------

	// GetDelegations returns all delegation configurations across all computers.
	GetDelegations() []schema.Delegation

	// GetDelegation returns the delegation config for a specific account.
	GetDelegation(account string) (schema.Delegation, bool)

	// GetUnconstrainedDelegation returns computers configured for unconstrained
	// Kerberos delegation (Delegation.Type == "unconstrained").
	// Domain controllers are excluded — they are unconstrained by default and
	// are not interesting targets for the printer-bug / TGT capture technique.
	GetUnconstrainedDelegation() []schema.Computer

	// GetConstrainedDelegation returns delegation entries where Type is
	// "constrained" or "resource-based". Includes the allowed services list
	// for S4U2Proxy abuse enumeration.
	GetConstrainedDelegation() []schema.Delegation

	// -------------------------------------------------------------------------
	// Certificates
	// -------------------------------------------------------------------------

	// GetCertTemplates returns all certificate templates defined in the lab.
	GetCertTemplates() []schema.CertTemplate

	// GetVulnerableCertTemplates returns templates where at least one ESC
	// flag (ESC1 through ESC8) is set to true.
	GetVulnerableCertTemplates() []schema.CertTemplate

	// -------------------------------------------------------------------------
	// Credential validation — reads Lab + SessionState.Loot
	// -------------------------------------------------------------------------

	// ValidateCredentials returns true if username:password is valid.
	// Checks the lab's User.Password field first, then SessionState.Loot
	// PlaintextCreds so that creds captured or cracked mid-session work.
	//
	// Azure equivalent: would validate against a token/access_token in Loot
	// rather than a password, since Entra auth is token-based.
	ValidateCredentials(username, password string) bool

	// ValidateNTLM returns true if username:NTHash is a valid pair.
	// Checks SessionState.Loot.NTLMHashes. Used by pass-the-hash paths in
	// CME, Impacket, and Mimikatz tools.
	ValidateNTLM(username, hash string) bool

	// HasLocalAdmin returns true if username has local administrator rights
	// on hostname. Checks Computer.LocalAdmins and also returns true for any
	// user that satisfies HasDomainAdmin, since DA implies local admin
	// everywhere by default.
	HasLocalAdmin(username, hostname string) bool

	// HasDomainAdmin returns true if username is a member of "Domain Admins"
	// through any chain of nested group membership.
	//
	// Azure equivalent: would check for Global Administrator or equivalent
	// Entra role assignment instead of DA group membership.
	HasDomainAdmin(username string) bool

	// HasSQLAdmin returns true if username has sql_admin rights on hostname,
	// either via static host rights, S4U ticket in session state, or DA.
	HasSQLAdmin(username, hostname string) bool

	// HasReplicationRights returns true if username holds AllExtendedRights
	// or GenericAll over the domain object in the ACL list. These rights
	// enable DCSync via MS-DRSR without requiring DA.
	HasReplicationRights(username string) bool

	// -------------------------------------------------------------------------
	// Domain metadata
	// -------------------------------------------------------------------------

	// DomainName returns the FQDN of the primary domain (e.g. "corp.local").
	DomainName() string

	// DomainController returns the hostname of the primary DC.
	// When multiple DCs exist, returns the first one found with IsDC == true.
	DomainController() string
}
// ProviderHelper wraps a Provider to add convenience aliases used by tools.
// Tools should accept a Provider; call these helpers directly on it.

// GetDomain returns the domain FQDN — alias for DomainName().
func GetDomain(p Provider) string { return p.DomainName() }

// GetAllComputers returns all computers — alias for GetComputers().
func GetAllComputers(p Provider) []schema.Computer { return p.GetComputers() }

// NetBIOSName returns the NetBIOS name derived from a domain FQDN
// (the first label, uppercased — e.g. "corp.local" → "CORP").
func NetBIOSName(fqdn string) string {
	if idx := strings.Index(fqdn, "."); idx > 0 {
		return strings.ToUpper(fqdn[:idx])
	}
	return strings.ToUpper(fqdn)
}
