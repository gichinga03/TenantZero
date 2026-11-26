package ad

import (
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
)

type ADProvider struct {
	lab   schema.Lab
	state *schema.SessionState
}

func NewADProvider(lab schema.Lab, state *schema.SessionState) *ADProvider {
	return &ADProvider{lab: lab, state: state}
}

func (p *ADProvider) GetUser(sam string) (schema.User, bool) {
	for _, u := range p.lab.Users {
		if strings.EqualFold(u.SAMAccountName, sam) {
			return u, true
		}
	}
	return schema.User{}, false
}

func (p *ADProvider) GetGroup(name string) (schema.Group, bool) {
	for _, g := range p.lab.Groups {
		if strings.EqualFold(g.Name, name) {
			return g, true
		}
	}
	return schema.Group{}, false
}

func (p *ADProvider) GetComputer(name string) (schema.Computer, bool) {
	for _, c := range p.lab.Computers {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return schema.Computer{}, false
}

func (p *ADProvider) GetACLsFor(to string) []schema.ACL {
	var result []schema.ACL
	for _, acl := range p.lab.ACLs {
		if strings.EqualFold(acl.To, to) {
			result = append(result, acl)
		}
	}
	return result
}

func (p *ADProvider) GetAllACLs() []schema.ACL {
	return p.lab.ACLs
}

func (p *ADProvider) GetSPNUsers() []schema.User {
	var result []schema.User
	for _, u := range p.lab.Users {
		if len(u.SPNs) > 0 {
			result = append(result, u)
		}
	}
	return result
}

func (p *ADProvider) GetDelegation(account string) (schema.Delegation, bool) {
	for _, d := range p.lab.Delegations {
		if strings.EqualFold(d.Account, account) {
			return d, true
		}
	}
	return schema.Delegation{}, false
}

func (p *ADProvider) GetCertTemplates() []schema.CertTemplate {
	return p.lab.CertTemplates
}

func (p *ADProvider) GetCertTemplate(name string) (schema.CertTemplate, bool) {
	for _, ct := range p.lab.CertTemplates {
		if strings.EqualFold(ct.Name, name) {
			return ct, true
		}
	}
	return schema.CertTemplate{}, false
}

func (p *ADProvider) hasRuntimeACL(account, target, right string) bool {
	key := strings.ToLower(account) + "." + strings.ToLower(target)
	rights, ok := p.state.GrantedACLs[key]
	if !ok {
		return false
	}
	for _, r := range rights {
		if strings.EqualFold(r, right) {
			return true
		}
	}
	return false
}

func (p *ADProvider) HasReplicationRights(account string) bool {
	if p.hasRuntimeACL(account, p.lab.Domain.Name, "DCSync") {
		return true
	}
	for _, acl := range p.lab.ACLs {
		if strings.EqualFold(acl.From, account) && strings.Contains(acl.Rights, "DCSync") {
			return true
		}
	}
	return false
}

func (p *ADProvider) GetFileSystem() schema.FileSystem {
	return p.lab.FileSystem
}

func (p *ADProvider) GetFileContent(path string) (schema.FileContent, bool) {
	fc, ok := p.lab.FileContents[path]
	return fc, ok
}

func (p *ADProvider) GetDomain() schema.Domain {
	return p.lab.Domain
}

// --- Topology methods ---

func (p *ADProvider) GetHost(hostname string) (schema.Host, bool) {
	for _, h := range p.lab.Hosts {
		if strings.EqualFold(h.Hostname, hostname) {
			return h, true
		}
	}
	return schema.Host{}, false
}

func (p *ADProvider) GetAllHosts() []schema.Host {
	return p.lab.Hosts
}

func (p *ADProvider) GetReachableHosts(fromHostname string) []schema.Host {
	var result []schema.Host
	for _, h := range p.lab.Hosts {
		for _, rf := range h.ReachableFrom {
			if strings.EqualFold(rf, fromHostname) {
				result = append(result, h)
				break
			}
		}
	}
	if result == nil {
		result = []schema.Host{}
	}
	return result
}

func (p *ADProvider) GetRelationships(from string) []schema.Relationship {
	var result []schema.Relationship
	for _, r := range p.lab.Relationships {
		if strings.EqualFold(r.From, from) {
			result = append(result, r)
		}
	}
	return result
}

func (p *ADProvider) GetTrust(sourceForest, targetForest string) (schema.Trust, bool) {
	for _, t := range p.lab.Trusts {
		if strings.EqualFold(t.SourceForest, sourceForest) && strings.EqualFold(t.TargetForest, targetForest) {
			return t, true
		}
	}
	return schema.Trust{}, false
}

func (p *ADProvider) IsInScope(hostname string) bool {
	h, ok := p.GetHost(hostname)
	if !ok {
		return false
	}
	return h.InScope
}

func (p *ADProvider) GetForest(forestName string) (schema.Forest, bool) {
	for _, f := range p.lab.Forests {
		if strings.EqualFold(f.ForestName, forestName) {
			return f, true
		}
	}
	return schema.Forest{}, false
}

func (p *ADProvider) GetIdentity(account string) (schema.Identity, bool) {
	for _, id := range p.lab.Identities {
		if strings.EqualFold(id.Account, account) {
			return id, true
		}
	}
	return schema.Identity{}, false
}

func (p *ADProvider) DomainController() string {
	return p.lab.Domain.DCHostname
}

func (p *ADProvider) GetUsers() []schema.User {
	return p.lab.Users
}

func (p *ADProvider) GetUsersWithAdminCount() []schema.User {
	var out []schema.User
	for _, u := range p.lab.Users {
		if u.AdminCount {
			out = append(out, u)
		}
	}
	return out
}

func (p *ADProvider) GetUserGroups(username string) []schema.Group {
	var out []schema.Group
	for _, g := range p.lab.Groups {
		for _, m := range g.Members {
			if strings.EqualFold(m, username) {
				out = append(out, g)
				break
			}
		}
	}
	return out
}

func (p *ADProvider) GetGroups() []schema.Group {
	return p.lab.Groups
}

func (p *ADProvider) GetGroupMembers(name string) []schema.User {
	g, ok := p.GetGroup(name)
	if !ok {
		return nil
	}
	var out []schema.User
	for _, m := range g.Members {
		if u, ok := p.GetUser(m); ok {
			out = append(out, u)
		}
	}
	return out
}

func (p *ADProvider) GetComputers() []schema.Computer {
	return p.lab.Computers
}

func (p *ADProvider) GetDomainControllers() []schema.Computer {
	var out []schema.Computer
	for _, c := range p.lab.Computers {
		if strings.EqualFold(c.Name, p.lab.Domain.DCHostname) {
			out = append(out, c)
		}
	}
	return out
}

func (p *ADProvider) GetLocalAdmins(hostname string) []string {
	return []string{"Administrator"}
}



func (p *ADProvider) GetACLsFrom(principal string) []schema.ACL {
	var out []schema.ACL
	for _, a := range p.lab.ACLs {
		if strings.EqualFold(a.From, principal) {
			out = append(out, a)
		}
	}
	return out
}

func (p *ADProvider) GetAbusableACLs() []schema.ACL {
	abusable := map[string]bool{
		"GenericAll": true, "GenericWrite": true, "WriteDACL": true,
		"WriteOwner": true, "AllExtendedRights": true, "Owns": true,
		"ForceChangePassword": true,
	}
	var out []schema.ACL
	for _, a := range p.lab.ACLs {
		if abusable[a.Rights] {
			out = append(out, a)
		}
	}
	return out
}

func (p *ADProvider) GetTrusts() []schema.Trust {
	return p.lab.Trusts
}

func (p *ADProvider) GetUnconstrainedDelegation() []schema.Computer {
	var out []schema.Computer
	for _, c := range p.lab.Computers {
		if c.UnconstrainedDelegation {
			out = append(out, c)
		}
	}
	return out
}

func (p *ADProvider) GetConstrainedDelegation() []schema.Delegation {
	var out []schema.Delegation
	for _, d := range p.lab.Delegations {
		if strings.EqualFold(d.Type, "constrained") || strings.EqualFold(d.Type, "resource-based") {
			out = append(out, d)
		}
	}
	return out
}

func (p *ADProvider) GetVulnerableCertTemplates() []schema.CertTemplate {
	var out []schema.CertTemplate
	for _, ct := range p.lab.CertTemplates {
		if ct.ESC1 || ct.ESC2 || ct.ESC3 || ct.ESC4 || ct.ESC8 {
			out = append(out, ct)
		}
	}
	return out
}

func (p *ADProvider) ValidateCredentials(username, password string) bool {
	for _, u := range p.lab.Users {
		if strings.EqualFold(u.SAMAccountName, username) && u.Password == password {
			return true
		}
	}
	for _, id := range p.lab.Identities {
		if strings.EqualFold(id.Account, username) && id.Password == password {
			return true
		}
	}
	// Also check loot plaintext
	for _, l := range p.state.Loot {
		if l.Type == "plaintext" && strings.EqualFold(l.Source, username) && l.Value == password {
			return true
		}
	}
	return false
}

func (p *ADProvider) ValidateNTLM(username, hash string) bool {
	for _, u := range p.lab.Users {
		if strings.EqualFold(u.SAMAccountName, username) && strings.EqualFold(u.NTHash, hash) {
			return true
		}
	}
	for _, l := range p.state.Loot {
		if l.Type == "hash" && strings.EqualFold(l.Source, username) && strings.EqualFold(l.Value, hash) {
			return true
		}
	}
	return false
}

func (p *ADProvider) HasLocalAdmin(username, hostname string) bool {
	if p.HasDomainAdmin(username) {
		return true
	}
	// Check ACLs for local admin rights on the host
	for _, a := range p.lab.ACLs {
		if strings.EqualFold(a.From, username) && strings.EqualFold(a.To, hostname) {
			if a.Rights == "GenericAll" || a.Rights == "LocalAdmin" {
				return true
			}
		}
	}
	return false
}

func (p *ADProvider) HasSQLAdmin(username, hostname string) bool {
	if p.HasDomainAdmin(username) {
		return true
	}
	// Check tickets in state — S4U ticket for this SPN grants sql_admin
	for _, t := range p.state.Tickets {
		if strings.EqualFold(t.Identity, username) &&
			strings.Contains(strings.ToLower(t.Target), strings.ToLower(hostname)) {
			return true
		}
	}
	// Check static host rights
	h, ok := p.GetHost(hostname)
	if !ok {
		return false
	}
	barePrincipal := username
	if idx := strings.LastIndexAny(username, `\/`); idx >= 0 {
		barePrincipal = username[idx+1:]
	}
	for _, r := range h.Rights {
		rBare := r.Principal
		if idx := strings.LastIndexAny(r.Principal, `\/`); idx >= 0 {
			rBare = r.Principal[idx+1:]
		}
		if strings.EqualFold(r.Principal, username) || strings.EqualFold(rBare, barePrincipal) {
			for _, rr := range r.Rights {
				if strings.EqualFold(rr, "sql_admin") {
					return true
				}
			}
		}
	}
	return false
}

func (p *ADProvider) HasDomainAdmin(username string) bool {
	for _, g := range p.lab.Groups {
		if strings.EqualFold(g.Name, "Domain Admins") {
			for _, m := range g.Members {
				if strings.EqualFold(m, username) {
					return true
				}
			}
		}
	}
	return false
}

func (p *ADProvider) DomainName() string {
	return p.lab.Domain.Name
}

func (p *ADProvider) GetUsersWithSPN() []schema.User {
	var out []schema.User
	for _, u := range p.lab.Users {
		if len(u.SPNs) > 0 && u.SAMAccountName != "krbtgt" {
			out = append(out, u)
		}
	}
	return out
}

func (p *ADProvider) GetACLs() []schema.ACL {
	return p.lab.ACLs
}

func (p *ADProvider) GetDelegations() []schema.Delegation {
	return p.lab.Delegations
}

// ValidateCrossForestCredentials checks credentials for a foreign-forest account.
// Matches against Users with a matching forest field, or Identities.
func (p *ADProvider) ValidateCrossForestCredentials(username, password, ntlm, targetDomain string) bool {
	for _, u := range p.lab.Users {
		if !strings.EqualFold(u.SAMAccountName, username) {
			continue
		}
		if u.Forest != "" && !strings.EqualFold(u.Forest, targetDomain) {
			continue
		}
		if ntlm != "" && strings.EqualFold(u.NTHash, ntlm) {
			return true
		}
		if password != "" && u.Password == password {
			return true
		}
	}
	for _, id := range p.lab.Identities {
		if !strings.EqualFold(id.Account, username) {
			continue
		}
		if id.Forest != "" && !strings.EqualFold(id.Forest, targetDomain) {
			continue
		}
		if password != "" && id.Password == password {
			return true
		}
	}
	return false
}
