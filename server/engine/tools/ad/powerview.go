// Package ad implements simulated AD tooling against the Provider interface.
package ad

import (
	"fmt"
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
	"github.com/0x4D67/tenantzero/server/engine/providers"
)

// PowerView simulates PowerView.ps1 enumeration cmdlets.
// All methods are pure: no state mutation, SideEffects always nil.
type PowerView struct{}

func (pv *PowerView) Name() string    { return "PowerView.ps1" }
func (pv *PowerView) Aliases() []string { return []string{"powerview"} }

// Execute dispatches on cmd.Subcommand, which the engine sets to the PowerShell
// function name (e.g. "Get-DomainUser"). PowerView is imported as a PS module so
// there is no binary prefix on the wire — the engine treats the function name as
// both Binary and Subcommand during parse for module-style tools.
func (pv *PowerView) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	p providers.Provider,
) schema.ToolResult {
	fn := cmd.Subcommand
	if fn == "" {
		fn = cmd.Executable // module-style: function is the binary token
	}

	switch strings.ToLower(fn) {
	case "get-domainuser":
		return pv.getDomainUser(cmd, p)
	case "get-domaingroup":
		return pv.getDomainGroup(cmd, p)
	case "get-domaincomputer":
		return pv.getDomainComputer(cmd, p)
	case "get-domaintrust":
		return pv.getDomainTrust(cmd, p)
	case "get-domaincontroller":
		return pv.getDomainController(cmd, p)
	case "get-objectacl":
		return pv.getObjectAcl(cmd, p)
	case "find-interestingdomainacl":
		return pv.findInterestingDomainAcl(cmd, p)
	case "get-domaingroupmember":
		return pv.getDomainGroupMember(cmd, p)
	case "set-domainobject":
		return pv.setDomainObject(cmd, state, p)
	case "add-domainobjectacl":
		return pv.addDomainObjectAcl(cmd, state, p)
	case "get-domainobjectacl":
		return pv.getDomainObjectAcl(cmd, p)
	default:
		return noResult(fn, fmt.Sprintf(
			"The term '%s' is not recognized as the name of a cmdlet, function, script file, or operable program.",
			fn,
		))
	}
}

// ─── Get-DomainUser ──────────────────────────────────────────────────────────

func (pv *PowerView) getDomainUser(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	spnOnly := hasFlag(cmd.Flags, "spn")
	adminOnly := hasFlag(cmd.Flags, "admincount")

	users := p.GetUsers()
	var out []string

	for _, u := range users {
		if identity != "" && !strings.EqualFold(u.SAMAccountName, identity) {
			continue
		}
		if spnOnly && len(u.SPNs) == 0 {
			continue
		}
		if adminOnly && !u.AdminCount {
			continue
		}
		out = append(out, renderUser(u, p))
	}

	return textResult("Get-DomainUser", out)
}

func renderUser(u schema.User, p providers.Provider) string {
	domain := p.DomainName()
	domainName := domain
	upn := fmt.Sprintf("%s@%s", u.SAMAccountName, domain)

	var groups []string
	for _, g := range p.GetGroups() {
		for _, m := range g.Members {
			if strings.EqualFold(m, u.SAMAccountName) {
				groups = append(groups, fmt.Sprintf("CN=%s,CN=Users,DC=%s", g.Name, dcComponents(domainName)))
			}
		}
	}

	spns := strings.Join(u.SPNs, ", ")
	if spns == "" {
		spns = "{}"
	}

	memberOf := "{}"
	if len(groups) > 0 {
		memberOf = strings.Join(groups, ", ")
	}

	flags := "NORMAL_ACCOUNT"
	if !u.Enabled {
		flags = "ACCOUNTDISABLE, NORMAL_ACCOUNT"
	}

	return joinFields([]kv{
		{"SamAccountName", u.SAMAccountName},
		{"UserPrincipalName", upn},
		{"DisplayName", u.SAMAccountName},
		{"MemberOf", memberOf},
		{"ServicePrincipalName", spns},
		{"AdminCount", boolStr(u.AdminCount)},
		{"UserAccountControl", flags},
		{"Description", ""},
		{"PasswordLastSet", "01/01/2026 00:00:00"},
		{"LastLogonDate", "01/01/2026 00:00:00"},
		{"DistinguishedName", fmt.Sprintf("CN=%s,CN=Users,DC=%s", u.SAMAccountName, dcComponents(domain))},
	})
}

// ─── Get-DomainGroup ─────────────────────────────────────────────────────────

func (pv *PowerView) getDomainGroup(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	domainName := p.DomainName()
	groups := p.GetGroups()
	var out []string

	for _, g := range groups {
		if identity != "" && !strings.EqualFold(g.Name, identity) {
			continue
		}
		out = append(out, renderGroup(g, domainName))
	}

	return textResult("Get-DomainGroup", out)
}

func renderGroup(g schema.Group, domain string) string {
	members := "{}"
	if len(g.Members) > 0 {
		var ms []string
		for _, m := range g.Members {
			ms = append(ms, fmt.Sprintf("CN=%s,CN=Users,DC=%s", m, dcComponents(domain)))
		}
		members = strings.Join(ms, ", ")
	}
	return joinFields([]kv{
		{"SamAccountName", g.Name},
		{"GroupScope", "Global"},
		{"GroupCategory", "Security"},
		{"Members", members},
		{"DistinguishedName", fmt.Sprintf("CN=%s,CN=Users,DC=%s", g.Name, dcComponents(domain))},
	})
}

// ─── Get-DomainComputer ──────────────────────────────────────────────────────

func (pv *PowerView) getDomainComputer(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	unconstrainedOnly := hasFlag(cmd.Flags, "unconstrained")

	domainName := p.DomainName()
	computers := p.GetComputers()
	var out []string

	for _, c := range computers {
		if identity != "" && !strings.EqualFold(c.Name, identity) {
			continue
		}
		delegations := p.GetConstrainedDelegation()
		if unconstrainedOnly {
			continue
		}
		out = append(out, renderComputer(c, delegations, domainName))
	}

	return textResult("Get-DomainComputer", out)
}

func renderComputer(c schema.Computer, delegations []schema.Delegation, domain string) string {
	spn := fmt.Sprintf("HOST/%s.%s, RestrictedKrbHost/%s.%s", c.Name, domain, c.Name, domain)
	var del schema.Delegation
	for _, d := range delegations {
		if strings.EqualFold(d.Account, c.Name) {
			del = d
			break
		}
	}
	delType := "None"
	if del.Type == "unconstrained" {
		delType = "Unconstrained"
	} else if len(del.AllowedTargets) > 0 {
		delType = "Constrained"
	}
	return joinFields([]kv{
		{"SamAccountName", c.Name + "$"},
		{"DNSHostName", fmt.Sprintf("%s.%s", c.Name, domain)},
		{"OperatingSystem", c.OperatingSystem},
		{"ServicePrincipalName", spn},
		{"TrustedForDelegation", boolStr(del.Type == "unconstrained")},
		{"TrustedToAuthForDelegation", boolStr(false)},
		{"DelegationType", delType},
		{"DistinguishedName", fmt.Sprintf("CN=%s,OU=Computers,DC=%s", c.Name, dcComponents(domain))},
	})
}

// ─── Get-DomainTrust ─────────────────────────────────────────────────────────

func (pv *PowerView) getDomainTrust(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	trusts := p.GetTrusts()
	domain := p.DomainName()
	var out []string

	for _, t := range trusts {
		direction := trustDirection(t.Direction)
		out = append(out, joinFields([]kv{
			{"SourceName", domain},
			{"TargetName", t.TargetForest},
			{"TrustDirection", direction},
			{"TrustType", t.TrustType},
			{"TrustAttributes", trustAttribs(t)},
			{"Transitive", boolStr(true)},
		}))
	}
	return textResult("Get-DomainTrust", out)
}

func trustDirection(d string) string {
	switch strings.ToLower(d) {
	case "inbound":
		return "Inbound"
	case "outbound":
		return "Outbound"
	case "bidirectional", "both":
		return "Bidirectional"
	default:
		return d
	}
}

func trustAttribs(t schema.Trust) string {
	var attrs []string
	if true {
		attrs = append(attrs, "WITHIN_FOREST")
	}
	if false {
		attrs = append(attrs, "FILTER_SIDS")
	}
	if len(attrs) == 0 {
		return "NONE"
	}
	return strings.Join(attrs, ", ")
}

// ─── Get-DomainController ────────────────────────────────────────────────────

func (pv *PowerView) getDomainController(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	domainName := p.DomainName()
	var out []string

	for _, c := range p.GetDomainControllers() {
		out = append(out, joinFields([]kv{
			{"Name", fmt.Sprintf("%s.%s", c.Name, domainName)},
			{"Domain", domainName},
			{"Forest", domainName},
			{"IPAddress", p.DomainController()},
			{"OSVersion", c.OperatingSystem},
			{"Roles", "SchemaRole, NamingRole, PdcRole, RidRole, InfrastructureRole"},
		}))
	}

	return textResult("Get-DomainController", out)
}

// ─── Get-ObjectAcl ───────────────────────────────────────────────────────────

func (pv *PowerView) getObjectAcl(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	if identity == "" {
		return noResult("Get-ObjectAcl", "Get-ObjectAcl : -Identity is required.")
	}

	acls := p.GetACLsFor(identity)
	domain := p.DomainName()
	var out []string

	for _, a := range acls {
		objectDN := fmt.Sprintf("CN=%s,CN=Users,DC=%s", identity, dcComponents(domain))
		trusteeSID := resolveSID(a.From, "")
		out = append(out, joinFields([]kv{
			{"ObjectDN", objectDN},
			{"ObjectSID", resolveSID(identity, "")},
			{"ActiveDirectoryRights", a.Rights},
			{"SecurityIdentifier", trusteeSID},
			{"IdentityReferenceName", a.From},
			{"AceType", "AccessAllowed"},
		}))
	}

	return textResult("Get-ObjectAcl", out)
}

// ─── Find-InterestingDomainAcl ───────────────────────────────────────────────

var abusableRights = map[string]bool{
	"GenericWrite":        true,
	"WriteDACL":           true,
	"GenericAll":          true,
	"AllExtendedRights":   true,
	"Owns":                true,
	"ForceChangePassword": true,
	"WriteOwner":          true,
}

func (pv *PowerView) findInterestingDomainAcl(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	domain := p.DomainName()
	allACLs := p.GetAbusableACLs()
	var out []string

	for _, a := range allACLs {
		objectDN := fmt.Sprintf("CN=%s,CN=Users,DC=%s", a.To, dcComponents(domain))
		out = append(out, joinFields([]kv{
			{"ObjectDN", objectDN},
			{"ActiveDirectoryRights", a.Rights},
			{"IdentityReferenceName", a.From},
			{"SecurityIdentifier", resolveSID(a.From, "")},
			{"AceType", "AccessAllowed"},
		}))
	}

	return textResult("Find-InterestingDomainAcl", out)
}

// ─── Get-DomainGroupMember ───────────────────────────────────────────────────

func (pv *PowerView) getDomainGroupMember(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	if identity == "" {
		return noResult("Get-DomainGroupMember", "Get-DomainGroupMember : -Identity is required.")
	}

	members := p.GetGroupMembers(identity)
	domain := p.DomainName()
	var out []string

	for _, m := range members {
		out = append(out, joinFields([]kv{
			{"GroupName", identity},
			{"MemberName", m.SAMAccountName},
			{"MemberDomain", domain},
			{"MemberDistinguishedName", fmt.Sprintf("CN=%s,CN=Users,DC=%s", m.SAMAccountName, dcComponents(domain))},
			{"MemberObjectClass", "user"},
		}))
	}

	return textResult("Get-DomainGroupMember", out)
}

// ─── Render helpers ──────────────────────────────────────────────────────────

type kv struct {
	Key   string
	Value string
}

func joinFields(fields []kv) string {
	var sb strings.Builder
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("%-30s : %s\n", f.Key, f.Value))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// textResult formats a slice of rendered objects as PowerView output.
// Objects are separated by blank lines. Empty slice → "No results found".
func textResult(fn string, objects []string) schema.ToolResult {
	if len(objects) == 0 {
		return noResult(fn, fmt.Sprintf("%s : No results found.", fn))
	}
	return schema.ToolResult{
		Output:      strings.Join(objects, "\n\n"),
		IsError:     false,
		Effects: nil,
	}
}

func noResult(fn, msg string) schema.ToolResult {
	return schema.ToolResult{
		Output:      msg,
		IsError:     false,
		Effects: nil,
	}
}

// flagValue returns the value for the first matching flag key, case-insensitive.
// Expects flags map keys to already be lowercased by the parser; we lowercase
// candidates here defensively.


func boolStr(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// dcComponents converts "corp.local" → "DC=corp,DC=local"

// resolveSID builds a plausible SID from name + domain SID prefix.
// Real SIDs come from the lab YAML; this is a fallback for display.
func resolveSID(name, domainSID string) string {
	if domainSID == "" {
		domainSID = "S-1-5-21-0000000000-0000000000-0000000000"
	}
	// Deterministic RID from name hash — good enough for display parity.
	var h uint32 = 2166136261
	for _, c := range name {
		h ^= uint32(c)
		h *= 16777619
	}
	rid := 1000 + (h % 9000)
	return fmt.Sprintf("%s-%d", domainSID, rid)
}
// PowerViewVerbs returns all PowerShell function names this tool handles.
// Used to register each verb directly in the tool registry so the parser's
// cmd.Executable (the verb itself) resolves to this tool.
func PowerViewVerbs() []string {
	return []string{
		"get-domainuser",
		"get-domaingroup",
		"get-domaincomputer",
		"get-domaintrust",
		"get-domaincontroller",
		"get-objectacl",
		"find-interestingdomainacl",
		"get-domaingroupmember",
		"set-domainobject",
		"add-domainobjectacl",
		"get-domainobjectacl",
	}
}

// ── Set-DomainObject ──────────────────────────────────────────────────────────

func (pv *PowerView) setDomainObject(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	p providers.Provider,
) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	if identity == "" {
		return noResult("Set-DomainObject", "[-] -Identity is required")
	}

	// Check caller has GenericWrite or GenericAll over the target.
	caller := state.ActiveUser
	var hasWrite bool
	for _, a := range p.GetACLs() {
		if strings.EqualFold(a.From, caller) && strings.EqualFold(a.To, identity) {
			if a.Rights == "GenericWrite" || a.Rights == "GenericAll" || a.Rights == "AllExtendedRights" {
				hasWrite = true
				break
			}
		}
	}
	if !hasWrite {
		return noResult("Set-DomainObject", fmt.Sprintf(
			"[-] Access denied: %s does not have GenericWrite over %s", caller, identity,
		))
	}

	// Accept the set operation — in a real sim we'd mutate state, but
	// since SPNs are static in the YAML we just acknowledge and emit
	// a loot side-effect so win conditions can fire.
	return schema.ToolResult{
		Output: fmt.Sprintf("[*] Set-DomainObject executed on '%s'\r\n", identity),
		Effects: []schema.SideEffect{
			{Type: "set_domain_object", LootSource: identity, LootValue: "modified"},
		},
	}
}

// ── Add-DomainObjectAcl ───────────────────────────────────────────────────────

func (pv *PowerView) addDomainObjectAcl(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	p providers.Provider,
) schema.ToolResult {
	target := flagValue(cmd.Flags, "targetidentity")
	principal := flagValue(cmd.Flags, "principalidentity")
	rights := flagValue(cmd.Flags, "rights")

	if target == "" || principal == "" {
		return noResult("Add-DomainObjectAcl", "[-] -TargetIdentity and -PrincipalIdentity are required")
	}
	if rights == "" {
		rights = "DCSync"
	}

	// Check caller has WriteDACL over target.
	// Effective caller is ActiveUser, but also check impersonated tokens/tickets.
	// Normalise "DC=meridian,DC=local" → "meridian.local" for ACL table lookup.
	callers := effectiveCallers(state)
	aclTarget := normDN(target)
	var hasWriteDACL bool
	for _, caller := range callers {
		for _, a := range p.GetACLs() {
			if strings.EqualFold(a.From, caller) && (strings.EqualFold(a.To, aclTarget) || strings.EqualFold(a.To, target)) {
				if a.Rights == "WriteDACL" || a.Rights == "GenericAll" || a.Rights == "WriteOwner" {
					hasWriteDACL = true
					break
				}
			}
		}
		if hasWriteDACL {
			break
		}
	}
	if !hasWriteDACL {
		return noResult("Add-DomainObjectAcl", fmt.Sprintf(
			"[-] Access denied: %s does not have WriteDACL over %s", state.ActiveUser, target,
		))
	}

	return schema.ToolResult{
		Output: fmt.Sprintf(
			"[*] Add-DomainObjectAcl: granted '%s' rights '%s' on '%s'\r\n",
			principal, rights, target,
		),
		Effects: []schema.SideEffect{
			{Type: schema.SideEffectTypeGrantACL, Principal: principal, ACLTarget: normDN(target), ACLRight: rights},
		},
	}
}

// ── Get-DomainObjectAcl ───────────────────────────────────────────────────────

func (pv *PowerView) getDomainObjectAcl(cmd schema.ParsedCommand, p providers.Provider) schema.ToolResult {
	identity := flagValue(cmd.Flags, "identity")
	resolveGUIDs := hasFlag(cmd.Flags, "resolveguids")
	_ = resolveGUIDs

	acls := p.GetACLs()
	var out []string
	for _, a := range acls {
		if identity != "" {
			// match plain name OR DC=x,DC=y form
			plain := strings.EqualFold(a.To, identity)
			dn := strings.EqualFold("DC="+strings.ReplaceAll(a.To, ".", ",DC="), identity)
			if !plain && !dn {
				continue
			}
		}
		domain := p.DomainName()
		out = append(out, fmt.Sprintf(
			"ObjectDN                       : CN=%s,CN=Users,DC=%s\r\nActiveDirectoryRights          : %s\r\nIdentityReferenceName          : %s\r\nAceType                        : AccessAllowed",
			a.To, dcComponents(domain), a.Rights, a.From,
		))
	}
	return textResult("Get-DomainObjectAcl", out)
}

// normDN converts "DC=meridian,DC=local" → "meridian.local" for ACL table lookups.
func normDN(dn string) string {
	dn = strings.ToLower(strings.TrimSpace(dn))
	if !strings.HasPrefix(dn, "dc=") {
		return dn
	}
	parts := strings.Split(dn, ",")
	var labels []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "dc=") {
			labels = append(labels, strings.TrimPrefix(p, "dc="))
		}
	}
	return strings.Join(labels, ".")
}

