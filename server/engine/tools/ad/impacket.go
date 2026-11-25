// Package ad implements Active Directory attack tool simulations for the TenantZero engine.
package ad

import (
	"fmt"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/schema"
)

// Impacket simulates the Impacket suite of Python tools.
// Each Impacket binary is a subcommand: secretsdump.py, psexec.py, etc.
type Impacket struct{}

// Name returns the tool identifier used in the registry.
func (i *Impacket) Name() string { return "impacket" }
func (i *Impacket) Aliases() []string { return []string{"impacket-secretsdump", "impacket-psexec"} }

// Requires returns the minimum access conditions.
func (i *Impacket) Requires() []string { return nil }

// Execute dispatches an Impacket tool invocation.
// Format: <tool>.py <domain>/<user>:<pass>@<target> [flags]
//
// The first arg is the tool name (e.g. secretsdump.py).
func (i *Impacket) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(cmd.Args) == 0 {
		return schema.ToolResult{Output: impacketUsage()}
	}

	tool := strings.ToLower(cmd.Args[0])
	remaining := cmd.Args[1:]

	switch tool {
	case "secretsdump.py":
		return i.secretsdump(remaining, state, provider)
	case "psexec.py":
		return i.psexec(remaining, state, provider)
	case "wmiexec.py":
		return i.wmiexec(remaining, state, provider)
	case "getuserspns.py":
		return i.getUserSPNs(remaining, state, provider)
	case "ticketer.py":
		return i.ticketer(remaining, state, provider)
	default:
		return schema.ToolResult{Output: fmt.Sprintf("[-] Unknown impacket tool: %s\n", tool)}
	}
}

// secretsdump simulates secretsdump.py credential extraction.
// Format: secretsdump.py <domain>/<user>:<pass>@<target>
func (i *Impacket) secretsdump(
	args []string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(args) == 0 {
		return schema.ToolResult{Output: "[-] Usage: secretsdump.py <domain>/<user>:<pass>@<target>\n"}
	}

	creds, target, ok := parseImpacketTarget(args[0])
	if !ok {
		return schema.ToolResult{Output: "[-] Usage: secretsdump.py <domain>/<user>:<pass>@<target>\n"}
	}

	if !provider.ValidateCredentials(creds.user, creds.pass) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] %s\\%s:%s - SMB SessionError: STATUS_LOGON_FAILURE\n",
				creds.domain, creds.user, creds.pass),
		}
	}

	computer, ok2 := resolveTarget(target, provider)
	if !ok2 {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] RemoteOperations failed: Can't connect to %s\n", target),
		}
	}

	if !provider.HasLocalAdmin(creds.user, computer.Hostname) {
		return schema.ToolResult{
			Output: "[-] RemoteOperations failed: DCERPC Runtime Error: code: 0x5 - rpc_s_access_denied\n",
		}
	}

	var sb strings.Builder
	domain := providers.NetBIOSName(providers.GetDomain(provider))
	fqdn := providers.GetDomain(provider)

	fmt.Fprintf(&sb, "[*] Service RemoteRegistry is in stopped state\n")
	fmt.Fprintf(&sb, "[*] Starting service RemoteRegistry\n")
	fmt.Fprintf(&sb, "[*] Target system bootKey: 0x%s\n", deterministicSysKey(computer.Hostname)[:16])
	sb.WriteString("[*] Dumping local SAM hashes (uid:rid:lmhash:nthash)\n")

	var effects []schema.SideEffect

	// SAM dump.
	localAccounts := []string{"Administrator"}
	seen := map[string]bool{}
	for _, acct := range localAccounts {
		if seen[acct] {
			continue
		}
		seen[acct] = true
		hash := fnvNTLM(acct + "@" + computer.Hostname)
		fmt.Fprintf(&sb, "%s:500:%s:%s:::\n",
			acct,
			"aad3b435b51404eeaad3b435b51404ee",
			hash,
		)
		effects = append(effects, schema.SideEffect{
			Type:       schema.SideEffectTypeLoot,
			LootType:   "hash",
			LootValue:  hash,
			LootSource: fmt.Sprintf("ntlm_hashes.%s\\%s", domain, acct),
		})
	}

	// NTDS / DC dump.
	if strings.EqualFold(computer.Role, "domain_controller") {
		sb.WriteString("\n[*] Dumping Domain Credentials (domain\\uid:rid:lmhash:nthash)\n")
		sb.WriteString("[*] Using the DRSUAPI method to get NTDS.DIT secrets\n")
		users := provider.GetUsers()
		for _, u := range users {
			hash := fnvNTLM(u.SAMAccountName + "@" + fqdn)
			rid := fnvRID(u.SAMAccountName)
			fmt.Fprintf(&sb, "%s\\%s:%d:%s:%s:::\n",
				domain, u.SAMAccountName,
				rid,
				"aad3b435b51404eeaad3b435b51404ee",
				hash,
			)
			effects = append(effects, schema.SideEffect{
				Type:       schema.SideEffectTypeLoot,
				LootType:   "hash",
				LootValue:  hash,
				LootSource: fmt.Sprintf("ntlm_hashes.%s\\%s", domain, u.SAMAccountName),
			})
		}
		// Always include krbtgt.
		krbtgtHash := fnvNTLM("krbtgt@" + fqdn)
		fmt.Fprintf(&sb, "%s\\krbtgt:502:%s:%s:::\n",
			domain,
			"aad3b435b51404eeaad3b435b51404ee",
			krbtgtHash,
		)
		effects = append(effects, schema.SideEffect{
			Type:       schema.SideEffectTypeLoot,
			LootType:   "hash",
			LootValue:  krbtgtHash,
			LootSource: "ntlm_hashes.krbtgt",
		})
	}

	sb.WriteString("\n[*] Cleaning up...\n")
	sb.WriteString("[*] Stopping service RemoteRegistry\n")

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// psexec simulates psexec.py remote shell execution.
// Lands session as SYSTEM if the user is DA, otherwise as the authenticated user.
func (i *Impacket) psexec(
	args []string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(args) == 0 {
		return schema.ToolResult{Output: "[-] Usage: psexec.py <domain>/<user>:<pass>@<target>\n"}
	}

	creds, target, ok := parseImpacketTarget(args[0])
	if !ok {
		return schema.ToolResult{Output: "[-] Usage: psexec.py <domain>/<user>:<pass>@<target>\n"}
	}

	if !provider.ValidateCredentials(creds.user, creds.pass) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] %s\\%s:%s - SMB SessionError: STATUS_LOGON_FAILURE\n",
				creds.domain, creds.user, creds.pass),
		}
	}

	computer, ok2 := resolveTarget(target, provider)
	if !ok2 {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Connection to %s failed\n", target),
		}
	}

	if !provider.HasLocalAdmin(creds.user, computer.Hostname) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] %s\\%s - SMB SessionError: STATUS_ACCESS_DENIED\n",
				creds.domain, creds.user),
		}
	}

	landAs := creds.user
	if provider.HasDomainAdmin(creds.user) {
		landAs = "SYSTEM"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Impacket v0.11.0 - Copyright 2023 Fortra\n\n")
	fmt.Fprintf(&sb, "[*] Requesting shares on %s.....\n", computer.Hostname)
	fmt.Fprintf(&sb, "[*] Found writable share ADMIN$\n")
	fmt.Fprintf(&sb, "[*] Uploading file %s.exe\n", deterministicSysKey(computer.Hostname)[:8])
	fmt.Fprintf(&sb, "[*] Opening SVCManager on %s.....\n", computer.Hostname)
	fmt.Fprintf(&sb, "[*] Creating service %s on %s.....\n",
		strings.ToUpper(deterministicSysKey(computer.Hostname)[:4]),
		computer.Hostname)
	fmt.Fprintf(&sb, "[*] Starting service %s.....\n",
		strings.ToUpper(deterministicSysKey(computer.Hostname)[:4]))
	sb.WriteString("[!] Press help for extra shell commands\n")
	fmt.Fprintf(&sb, "Microsoft Windows [Version 10.0.17763.5696]\n")
	fmt.Fprintf(&sb, "(c) 2018 Microsoft Corporation. All rights reserved.\n\n")
	fmt.Fprintf(&sb, "C:\\Windows\\system32>")

	effects := []schema.SideEffect{
		{Type: schema.SideEffectTypePivotContext, PivotHost: computer.Hostname, PivotIdentity: landAs},
		{Type: schema.SideEffectTypeSetSessionUser, NewUser: landAs},
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// wmiexec simulates wmiexec.py remote command execution.
// Lands as the authenticated user (not SYSTEM).
func (i *Impacket) wmiexec(
	args []string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(args) == 0 {
		return schema.ToolResult{Output: "[-] Usage: wmiexec.py <domain>/<user>:<pass>@<target>\n"}
	}

	creds, target, ok := parseImpacketTarget(args[0])
	if !ok {
		return schema.ToolResult{Output: "[-] Usage: wmiexec.py <domain>/<user>:<pass>@<target>\n"}
	}

	if !provider.ValidateCredentials(creds.user, creds.pass) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] %s\\%s:%s - SMB SessionError: STATUS_LOGON_FAILURE\n",
				creds.domain, creds.user, creds.pass),
		}
	}

	computer, ok2 := resolveTarget(target, provider)
	if !ok2 {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Connection to %s failed\n", target),
		}
	}

	if !provider.HasLocalAdmin(creds.user, computer.Hostname) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] %s\\%s - SMB SessionError: STATUS_ACCESS_DENIED\n",
				creds.domain, creds.user),
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Impacket v0.11.0 - Copyright 2023 Fortra\n\n")
	fmt.Fprintf(&sb, "[*] SMBv3.0 dialect used\n")
	fmt.Fprintf(&sb, "[!] Launching semi-interactive shell - Careful what you execute\n")
	fmt.Fprintf(&sb, "[!] Press help for extra shell commands\n")
	fmt.Fprintf(&sb, "C:\\>")

	effects := []schema.SideEffect{
		{Type: schema.SideEffectTypePivotContext, PivotHost: computer.Hostname, PivotIdentity: creds.user},
		{Type: schema.SideEffectTypeSetSessionUser, NewUser: creds.user},
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// getUserSPNs simulates GetUserSPNs.py Kerberoasting.
// Format: GetUserSPNs.py <domain>/<user>:<pass>
func (i *Impacket) getUserSPNs(
	args []string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(args) == 0 {
		return schema.ToolResult{Output: "[-] Usage: GetUserSPNs.py <domain>/<user>:<pass>\n"}
	}

	// GetUserSPNs doesn't take a target host — just domain/user:pass.
	creds, _, credOK := parseImpacketTarget(args[0])
	if !credOK {
		// Try without target suffix.
		creds2, ok2 := parseImpacketCreds(args[0])
		if !ok2 {
			return schema.ToolResult{Output: "[-] Usage: GetUserSPNs.py <domain>/<user>:<pass>\n"}
		}
		creds = creds2
	}

	if !provider.ValidateCredentials(creds.user, creds.pass) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] %s\\%s:%s - Kerberos SessionError: KRB_AP_ERR_SKEW\n",
				creds.domain, creds.user, creds.pass),
		}
	}

	spnUsers := provider.GetUsersWithSPN()
	if len(spnUsers) == 0 {
		return schema.ToolResult{Output: "[*] No entries found!\n"}
	}

	var sb strings.Builder
	sb.WriteString("Impacket v0.11.0 - Copyright 2023 Fortra\n\n")
	fmt.Fprintf(&sb, "ServicePrincipalName                 Name       MemberOf  PasswordLastSet             LastLogon                   Delegation\n")
	fmt.Fprintf(&sb, "-----------------------------------  ---------  --------  --------------------------  --------------------------  ----------\n")

	var effects []schema.SideEffect
	domain := providers.GetDomain(provider)

	for _, u := range spnUsers {
		fmt.Fprintf(&sb, "%-36s %-10s           %s  <never>\n",
			u.SPNs, u.SAMAccountName,
			time.Now().Add(-30*24*time.Hour).Format("2006-01-02 15:04:05.999999"),
		)
		spn := ""; if len(u.SPNs) > 0 { spn = u.SPNs[0] }
		hash := kerberoastHash(u.SAMAccountName, domain, spn, fnvNTLM(u.SAMAccountName+domain))
		effects = append(effects, schema.SideEffect{
			Type:       schema.SideEffectTypeLoot,
			LootType:   "spn_hash",
			LootValue:  hash,
			LootSource: "spn_hashes." + u.SAMAccountName,
		})
	}

	sb.WriteString("\n")
	for _, u := range spnUsers {
		spn := ""; if len(u.SPNs) > 0 { spn = u.SPNs[0] }
		hash := kerberoastHash(u.SAMAccountName, domain, spn, fnvNTLM(u.SAMAccountName+domain))
		sb.WriteString(hash + "\n")
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// kerberoastHash returns a fake $krb5tgs$23$ hash seeded from username+domain.
// Mirrors the helper in rubeus.go for consistent output.

// ticketer simulates ticketer.py golden ticket creation.
// Format: ticketer.py -nthash <hash> -domain-sid <sid> -domain <domain> <username>
func (i *Impacket) ticketer(
	args []string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	flags := parseImpacketFlags(args)
	nthash := flags["nthash"]
	domainSID := flags["domain-sid"]
	domain := flags["domain"]

	// Username is a positional arg — last non-flag arg.
	username := ""
	for j := len(args) - 1; j >= 0; j-- {
		if !strings.HasPrefix(args[j], "-") {
			username = args[j]
			break
		}
	}

	if nthash == "" || domainSID == "" || domain == "" || username == "" {
		return schema.ToolResult{
			Output: "[-] Usage: ticketer.py -nthash <hash> -domain-sid <sid> -domain <domain> <username>\n",
		}
	}

	krbtgtHash := ""
	for _, l := range state.Loot {
		if l.Type == "hash" && strings.Contains(strings.ToLower(l.Source), "krbtgt") {
			krbtgtHash = l.Value
			break
		}
	}
	if krbtgtHash != "" && !strings.EqualFold(krbtgtHash, nthash) {
		return schema.ToolResult{
			Output: "[-] Got error while trying to validate credentials and target: Kerberos SessionError: KRB_AP_ERR_MODIFIED\n",
		}
	}

	var sb strings.Builder
	sb.WriteString("Impacket v0.11.0 - Copyright 2023 Fortra\n\n")
	fmt.Fprintf(&sb, "[*] Creating basic skeleton ticket and PAC Infos\n")
	fmt.Fprintf(&sb, "[*] Signing/Encrypting final ticket\n")
	fmt.Fprintf(&sb, "[*] Saving ticket in %s.ccache\n", username)

	effects := []schema.SideEffect{
		{
			Type:       schema.SideEffectTypeTicket,
			TicketType: "TGT",
			Identity:   username,
			Target:     domain,
		},
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// impacketCreds holds parsed credential components.
type impacketCreds struct {
	domain string
	user   string
	pass   string
}

// parseImpacketTarget parses "domain/user:pass@target" into components.
// Returns false if the string doesn't match the expected format.
func parseImpacketTarget(s string) (impacketCreds, string, bool) {
	// Split on last '@'.
	atIdx := strings.LastIndex(s, "@")
	if atIdx < 0 {
		return impacketCreds{}, "", false
	}
	credStr := s[:atIdx]
	target := s[atIdx+1:]

	creds, ok := parseImpacketCreds(credStr)
	return creds, target, ok
}

// parseImpacketCreds parses "domain/user:pass" with optional domain prefix.
func parseImpacketCreds(s string) (impacketCreds, bool) {
	var domain, rest string
	if idx := strings.Index(s, "/"); idx >= 0 {
		domain = s[:idx]
		rest = s[idx+1:]
	} else {
		rest = s
	}

	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return impacketCreds{}, false
	}
	user := rest[:colonIdx]
	pass := rest[colonIdx+1:]
	return impacketCreds{domain: domain, user: user, pass: pass}, true
}

// parseImpacketFlags extracts -flag value pairs from Impacket-style args.
func parseImpacketFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for j := 0; j < len(args); j++ {
		a := args[j]
		if strings.HasPrefix(a, "-") {
			key := strings.TrimLeft(a, "-")
			if j+1 < len(args) && !strings.HasPrefix(args[j+1], "-") {
				flags[key] = args[j+1]
				j++
			} else {
				flags[key] = "true"
			}
		}
	}
	return flags
}

func impacketUsage() string {
	return `Impacket tool suite. Usage: <tool>.py [options]

Available tools:
  secretsdump.py    Dump SAM/NTDS hashes remotely
  psexec.py         Remote shell via SMB (SYSTEM)
  wmiexec.py        Remote shell via WMI (user context)
  GetUserSPNs.py    Kerberoast SPN accounts
  ticketer.py       Forge Kerberos tickets
`
}