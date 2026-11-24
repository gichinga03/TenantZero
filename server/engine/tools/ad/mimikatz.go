// Package ad implements Active Directory attack tool simulations for the TenantZero engine.
package ad

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/schema"
)

// Mimikatz simulates the mimikatz credential dumping tool.
type Mimikatz struct{}

const mimikatzBanner = `

  .#####.   mimikatz 2.2.0 (x64) #19041 Sep 19 2022 17:44:08
 .## ^ ##.  "A La Vie, A L'Amour" - (oe.eo)
 ## / \ ##  /*** Benjamin DELPY ` + "`" + `gentilkiwi` + "`" + ` ( benjamin@gentilkiwi.com )
 ## \ / ##       > https://blog.gentilkiwi.com/mimikatz
 '## v ##'       Vincent LE TOUX             ( vincent.letoux@gmail.com )
  '#####'        > https://pingcastle.com / https://mysmartlogon.com ***/

mimikatz # 

Modules available:
  crypto      Crypto Module
  kerberos    Kerberos Module
  lsadump     LsaDump Module
  sekurlsa    SekurLsa Module
  token       Token manipulation
`

// Name returns the tool identifier used in the registry.
func (m *Mimikatz) Name() string { return "mimikatz" }
func (m *Mimikatz) Aliases() []string { return []string{"mimikatz.exe"} }

// Requires returns the minimum access conditions for mimikatz.
// Individual subcommands enforce stricter checks internally.
func (m *Mimikatz) Requires() []string { return nil }

// Execute dispatches a mimikatz module::command invocation.
// Input format: mimikatz # <module>::<command> [flags]
func (m *Mimikatz) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	// Support both "mimikatz # lsadump::dcsync" (Subcommand) and direct Args.
	invocation := strings.TrimSpace(cmd.Subcommand + " " + strings.Join(cmd.Args, " "))
	if strings.TrimSpace(invocation) == "" {
		return schema.ToolResult{Output: mimikatzBanner}
	}
	parts := strings.SplitN(invocation, "::", 2)
	if len(parts) != 2 {
		return schema.ToolResult{Output: mimikatzBanner}
	}

	module := strings.ToLower(strings.TrimSpace(parts[0]))
	rest := strings.TrimSpace(parts[1])
	subcmdParts := strings.Fields(rest)
	subcmd := ""
	if len(subcmdParts) > 0 {
		subcmd = strings.ToLower(subcmdParts[0])
	}

	// Flags come from cmd.Flags (parsed by parseFlags in command.go), not subcmdParts.
	// parseMimikatzFlags expects []string of "/key:val" but cmd.Flags is map[string]string already.
	// Merge both: subcmdParts[1:] for inline args, cmd.Flags for pre-parsed flags.
	flags := parseMimikatzFlags(subcmdParts[1:])
	for k, v := range cmd.Flags {
		if _, exists := flags[strings.ToLower(k)]; !exists {
			flags[strings.ToLower(k)] = v
		}
	}

	switch module + "::" + subcmd {
	case "sekurlsa::logonpasswords":
		return m.sekurlsaLogonPasswords(state, provider)
	case "lsadump::dcsync":
		return m.lsadumpDcsync(flags, state, provider)
	case "lsadump::sam":
		return m.lsadumpSam(state, provider)
	case "kerberos::golden":
		return m.kerberosGolden(flags, state, provider)
	case "kerberos::ptt":
		return m.kerberosPtt(flags)
	default:
		return schema.ToolResult{Output: mimikatzBanner}
	}
}

// sekurlsaLogonPasswords dumps cached credentials from LSASS memory.
// Requires local admin on the current machine.
func (m *Mimikatz) sekurlsaLogonPasswords(
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if !provider.HasLocalAdmin(state.User, state.Hostname) {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_sekurlsa_acquireLSA ; Handle on memory (0x00000005)",
		}
	}

	var sb strings.Builder
	sb.WriteString("\nAuthentication Id : 0 ; 999 (00000000:000003e7)\n")
	sb.WriteString("Session           : UndefinedLogonType from 0\n")
	sb.WriteString("User Name         : SYSTEM\n")
	sb.WriteString("Domain            : NT AUTHORITY\n")

	// Collect credentials relevant to the current machine.
	type credEntry struct {
		username string
		domain   string
		ntlm     string
		password string
	}

	var entries []credEntry

	// Pull plaintext creds from loot.
	for user, pass := range state.Loot.PlaintextCreds {
		domain := providers.NetBIOSName(providers.GetDomain(provider))
		if domain == "" {
			domain = providers.GetDomain(provider)
		}
		entries = append(entries, credEntry{
			username: user,
			domain:   domain,
			password: pass,
			ntlm:     fnvNTLM(pass),
		})
	}

	// Pull NTLM-only entries from loot (skip those already in plaintext).
	for user, hash := range state.Loot.NTLMHashes {
		if _, already := state.Loot.PlaintextCreds[user]; already {
			continue
		}
		domain := providers.NetBIOSName(providers.GetDomain(provider))
		entries = append(entries, credEntry{
			username: user,
			domain:   domain,
			ntlm:     hash,
		})
	}

	// If nothing looted, emit a generic SYSTEM-only block so the machine
	// looks live but uninteresting.
	if len(entries) == 0 {
		sb.WriteString("Logon Server      : (null)\n\n")
		sb.WriteString("        msv :\n")
		sb.WriteString("        tspkg :\n")
		sb.WriteString("        wdigest :\n")
		sb.WriteString("         * Username : (null)\n")
		sb.WriteString("         * Domain   : (null)\n")
		sb.WriteString("         * Password : (null)\n")
		sb.WriteString("        kerberos :\n")
		sb.WriteString("        ssp :\n")
		sb.WriteString("        credman :\n")
		return schema.ToolResult{Output: sb.String()}
	}

	for i, e := range entries {
		fmt.Fprintf(&sb, "\nAuthentication Id : 0 ; %d (00000000:%08x)\n", 1000+i, 1000+i)
		sb.WriteString("Session           : Interactive from 1\n")
		fmt.Fprintf(&sb, "User Name         : %s\n", e.username)
		fmt.Fprintf(&sb, "Domain            : %s\n", e.domain)
		sb.WriteString("Logon Server      : DC01\n\n")
		sb.WriteString("        msv :\n")
		fmt.Fprintf(&sb, "         [00000003] Primary\n")
		fmt.Fprintf(&sb, "         * Username : %s\n", e.username)
		fmt.Fprintf(&sb, "         * Domain   : %s\n", e.domain)
		if e.ntlm != "" {
			fmt.Fprintf(&sb, "         * NTLM     : %s\n", e.ntlm)
			sb.WriteString("         * SHA1     : 0000000000000000000000000000000000000000\n")
		}
		sb.WriteString("        wdigest :\n")
		fmt.Fprintf(&sb, "         * Username : %s\n", e.username)
		fmt.Fprintf(&sb, "         * Domain   : %s\n", e.domain)
		if e.password != "" {
			fmt.Fprintf(&sb, "         * Password : %s\n", e.password)
		} else {
			sb.WriteString("         * Password : (null)\n")
		}
		sb.WriteString("        kerberos :\n")
		fmt.Fprintf(&sb, "         * Username : %s\n", e.username)
		fmt.Fprintf(&sb, "         * Domain   : %s\n", e.domain)
		if e.password != "" {
			fmt.Fprintf(&sb, "         * Password : %s\n", e.password)
		} else {
			sb.WriteString("         * Password : (null)\n")
		}
		sb.WriteString("        ssp :\n")
		sb.WriteString("        credman :\n")
	}

	return schema.ToolResult{Output: sb.String()}
}

// lsadumpDcsync simulates a DCSync attack against a target user account.
// Requires replication rights or domain admin.
func (m *Mimikatz) lsadumpDcsync(
	flags map[string]string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	targetUser, ok := flags["user"]
	if !ok || targetUser == "" {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_lsadump_dcsync ; /user:<username> required",
		}
	}

	// Check runtime-granted DCSync rights (via Add-DomainObjectAcl) first.
	hasDCSync := false
	if state.GrantedACLs != nil {
		for k, v := range state.GrantedACLs {
			if strings.HasPrefix(k, strings.ToLower(state.User)+".") && strings.EqualFold(v, "DCSync") {
				hasDCSync = true
				break
			}
		}
	}
	if !hasDCSync && !provider.HasReplicationRights(state.User) && !provider.HasDomainAdmin(state.User) {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_lsadump_dcsync ; GetNCChanges: 0x00002105 (ERROR_DS_DRA_ACCESS_DENIED)",
		}
	}

	hash := fnvNTLM(targetUser)
	sid := deterministicSID("S-1-5-21-SIMULATED", targetUser)
	guid := deterministicGUID(targetUser)
	domain := providers.GetDomain(provider)

	// krbtgt-specific flags — disabled account, no expiry
	accountType := "30000000 ( USER_OBJECT )"
	uac := "00010200 ( NORMAL_ACCOUNT DONT_EXPIRE_PASSWD )"
	if strings.ToLower(targetUser) == "krbtgt" {
		uac = "00000202 ( ACCOUNTDISABLE NORMAL_ACCOUNT )"
	}

	// Real AD domain SID — deterministic from domain name
	domainSID := deterministicDomainSID(domain)
	sid = fmt.Sprintf("%s-%d", domainSID, fnvRID(targetUser))

	// Real modern LM hash placeholder
	lmHash := "aad3b435b51404eeaad3b435b51404ee"

	// Real-looking GUID
	guid = deterministicGUID(targetUser)

	var sb strings.Builder
	fmt.Fprintf(&sb, "[DC] '%s' will be the domain\r\n", domain)
	fmt.Fprintf(&sb, "[DC] 'DC01.%s' will be the DC server\r\n", domain)
	fmt.Fprintf(&sb, "[DC] '%s' will be the user account\r\n", targetUser)
	sb.WriteString("[rpc] Service  : ldap\r\n")
	sb.WriteString("[rpc] AuthnSvc : GSS_NEGOTIATE (9)\r\n\r\n")
	fmt.Fprintf(&sb, "Object RDN           : %s\r\n\r\n", targetUser)
	sb.WriteString("** SAM ACCOUNT **\r\n\r\n")
	fmt.Fprintf(&sb, "SAM Username         : %s\r\n", targetUser)
	fmt.Fprintf(&sb, "Account Type         : %s\r\n", accountType)
	fmt.Fprintf(&sb, "User Account Control : %s\r\n", uac)
	sb.WriteString("Account expiration   :\r\n")
	fmt.Fprintf(&sb, "Password last change : %s\r\n", time.Now().Format("1/2/2006 3:04:05 PM"))
	fmt.Fprintf(&sb, "Object Security ID   : %s\r\n", sid)
	fmt.Fprintf(&sb, "Object Relative ID   : %d\r\n\r\n", fnvRID(targetUser))
	sb.WriteString("Credentials:\r\n")
	fmt.Fprintf(&sb, "  Hash NTLM: %s\r\n", hash)
	sb.WriteString("    ntlm- 0: " + hash + "\r\n")
	sb.WriteString("    lm  - 0: " + lmHash + "\r\n")
	fmt.Fprintf(&sb, "Object GUID          : %s\n", guid)

	effects := []schema.SideEffect{
		{
			Type:  schema.SideEffectSetLoot,
			Key:   "ntlm_hashes." + targetUser,
			Value: hash,
		},
	}

	return schema.ToolResult{Output: sb.String(), SideEffects: effects}
}

// lsadumpSam dumps local SAM account hashes from the current machine.
// Requires local admin.
func (m *Mimikatz) lsadumpSam(
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if !provider.HasLocalAdmin(state.User, state.Hostname) {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_lsadump_sam ; Handle on memory (0x00000005)",
		}
	}

	computer, ok := provider.GetComputer(state.Hostname)
	if !ok {
		return schema.ToolResult{
			Output: fmt.Sprintf("ERROR kuhl_m_lsadump_sam ; host '%s' not found", state.Hostname),
		}
	}

	var sb strings.Builder
	domain := providers.NetBIOSName(providers.GetDomain(provider))
	sb.WriteString("Domain : " + state.Hostname + "\n")
	sb.WriteString("SysKey : " + deterministicSysKey(state.Hostname) + "\n\n")
	sb.WriteString("Local SID : S-1-5-21-" + deterministicLocalSIDParts(state.Hostname) + "\n\n")
	sb.WriteString("SAMKey : " + deterministicSysKey(state.Hostname+"SAMKey") + "\n\n")

	var effects []schema.SideEffect

	accounts := append([]string{"Administrator"}, computer.LocalAdmins...)
	seen := map[string]bool{}
	for _, acct := range accounts {
		if seen[acct] {
			continue
		}
		seen[acct] = true

		hash := fnvNTLM(acct + "@" + state.Hostname)
		fmt.Fprintf(&sb, "RID  : %08x (%d)\n", fnvRID(acct), fnvRID(acct))
		fmt.Fprintf(&sb, "User : %s\n", acct)
		sb.WriteString("  Hash NTLM: " + hash + "\n\n")

		effects = append(effects, schema.SideEffect{
			Type:  schema.SideEffectSetLoot,
			Key:   "ntlm_hashes." + domain + `\` + acct,
			Value: hash,
		})
	}

	return schema.ToolResult{Output: sb.String(), SideEffects: effects}
}

// kerberosGolden forges a golden ticket given a valid krbtgt NTLM hash.
func (m *Mimikatz) kerberosGolden(
	flags map[string]string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flags["user"]
	domain := flags["domain"]
	sid := flags["sid"]
	krbtgt := flags["krbtgt"]

	if user == "" || domain == "" || sid == "" || krbtgt == "" {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_kerberos_golden ; /user, /domain, /sid, /krbtgt all required",
		}
	}

	expectedHash, hasKrbtgt := state.Loot.NTLMHashes["krbtgt"]
	if hasKrbtgt && !strings.EqualFold(expectedHash, krbtgt) {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_kerberos_golden ; Kerberos - krbtgt hash mismatch",
		}
	}

	var sb strings.Builder
	sb.WriteString("User      : " + user + "\n")
	sb.WriteString("Domain    : " + domain + "\n")
	sb.WriteString("SID       : " + sid + "\n")
	sb.WriteString("User Id   : 500\n")
	sb.WriteString("Groups Id : *513 512 520 518 519\n")
	sb.WriteString("ServiceKey: " + krbtgt + " - rc4_hmac_nt\n")
	sb.WriteString("Lifetime  : " + time.Now().Format("1/2/2006 3:04:05 PM") +
		" ; " + time.Now().Add(10*365*24*time.Hour).Format("1/2/2006 3:04:05 PM") +
		" ; " + time.Now().Add(10*365*24*time.Hour).Format("1/2/2006 3:04:05 PM") + "\n")
	sb.WriteString("-> Ticket : ticket.kirbi\n\n")
	sb.WriteString("* PAC generated\n")
	sb.WriteString("* PAC signed\n")
	sb.WriteString("* EncTicketPart generated\n")
	sb.WriteString("* EncTicketPart encrypted\n")
	sb.WriteString("* KrbCred generated\n\n")
	sb.WriteString("Golden ticket for '" + user + " @ " + domain + "' successfully submitted for current session\n")

	effects := []schema.SideEffect{
		{Type: schema.SideEffectAddTicket, Value: "ticket"},
	}

	return schema.ToolResult{Output: sb.String(), SideEffects: effects}
}

// kerberosPtt loads a Kerberos ticket into memory (pass-the-ticket).
// No side effect — ticket is assumed already present in state.
func (m *Mimikatz) kerberosPtt(flags map[string]string) schema.ToolResult {
	ticket, ok := flags["ticket"]
	if !ok || ticket == "" {
		return schema.ToolResult{
			Output: "ERROR kuhl_m_kerberos_ptt ; /ticket:<b64|path.kirbi> required",
		}
	}

	var output string
	if strings.HasSuffix(ticket, ".kirbi") {
		output = fmt.Sprintf("\n* File: '%s': OK\n", ticket)
	} else {
		output = "\n* Base64 of file: OK\n"
	}

	return schema.ToolResult{Output: output}
}

// parseMimikatzFlags extracts /key:value and /key (boolean) pairs from args.
func parseMimikatzFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for _, a := range args {
		a = strings.TrimPrefix(a, "/")
		if idx := strings.Index(a, ":"); idx >= 0 {
			flags[strings.ToLower(a[:idx])] = a[idx+1:]
		} else {
			flags[strings.ToLower(a)] = "true"
		}
	}
	return flags
}

// fnvNTLM returns a deterministic 32-hex-character fake NTLM hash seeded from s.
// Uses FNV-64a to match the convention established in rubeus.go.
func fnvNTLM(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	v := h.Sum64()
	// XOR upper and lower 32 bits to fill a 128-bit-looking hex string.
	hi := uint32(v >> 32)
	lo := uint32(v)
	return fmt.Sprintf("%08x%08x%08x%08x", hi, lo, hi^0xdeadbeef, lo^0xcafebabe)
}

// deterministicSID builds a fake but plausible SID for a user given a domain SID base.
func deterministicSID(domainSID, username string) string {
	if domainSID == "" {
		domainSID = "S-1-5-21-3623811015-3361044348-30300820"
	}
	return fmt.Sprintf("%s-%d", domainSID, fnvRID(username))
}

// fnvRID returns a stable relative identifier in the range [1000, 9999].
func fnvRID(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return 1000 + (h.Sum32() % 9000)
}

// deterministicGUID returns a fake GUID seeded from a username.
func deterministicGUID(s string) string {
	h := fnv.New64a()
	h.Write([]byte("guid:" + s))  // prefix prevents hash echo in first segment
	v1 := h.Sum64()
	h.Write([]byte("salt2"))
	v2 := h.Sum64()
	a := uint32(v1 >> 32)
	b := uint16(v1 & 0xffff)
	cc := uint16((v1>>16)&0x0fff) | 0x4000
	d := uint16((v2 & 0x3fff) | 0x8000)
	e := v2 >> 16
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", a, b, cc, d, e)
}

// deterministicSysKey returns a fake 32-hex-char syskey seeded from hostname.
func deterministicSysKey(hostname string) string {
	h := fnv.New64a()
	h.Write([]byte(hostname))
	v := h.Sum64()
	return fmt.Sprintf("%016x%016x", v, v^0xfeedface12345678)
}

// deterministicLocalSIDParts returns the three sub-authority components of a local SID.
func deterministicLocalSIDParts(hostname string) string {
	h := fnv.New64a()
	h.Write([]byte(hostname))
	v := h.Sum64()
	return fmt.Sprintf("%d-%d-%d", uint32(v), uint32(v>>16), uint32(v>>32))
}

// deterministicDomainSID builds a stable S-1-5-21-X-Y-Z from the domain name.
func deterministicDomainSID(domain string) string {
	h := fnv.New64a()
	h.Write([]byte(domain))
	v := h.Sum64()
	a := uint32(v & 0xffffffff)
	b := uint32((v >> 32) & 0xffffffff)
	h2 := fnv.New32a()
	h2.Write([]byte(domain + "salt"))
	c2 := h2.Sum32()
	return fmt.Sprintf("S-1-5-21-%d-%d-%d", a, b, c2)
}

