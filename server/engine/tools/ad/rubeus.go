package ad

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/schema"
)

// rubeusVersion printed in the banner.
const rubeusVersion = "v2.3.2"

const rubeusBanner = `
   ______        _
  (_____ \      | |
   _____) )_   _| |__  _____ _   _  ___
  |  __  /| | | |  _ \| ___ | | | |/___)
  | |  \ \| |_| | |_) ) ____| |_| |___ |
  |_|   |_|____/|____/|_____)____/(___/

  v` + rubeusVersion + `

`

const rubeusUsage = `  Ticket requests and renewals:
    asktgt         - Request a TGT
    asktgs         - Request a TGS
    renew          - Renew/autorenew a TGT

  Constrained delegation abuse:
    s4u            - S4U2Self and S4U2Proxy abuse

  Ticket management:
    ptt            - Pass a ticket
    triage         - List current tickets
    dump           - Dump all ticket data
    describe       - Describe a ticket

  Roasting:
    kerberoast     - Perform Kerberoasting

  Ticket forgery:
    golden         - Forge a golden ticket
    silver         - Forge a silver ticket

`

// Rubeus implements a simulated Rubeus.exe tool.
type Rubeus struct{}

func (r *Rubeus) Name() string { return "rubeus" }

func (r *Rubeus) Aliases() []string { return []string{"Rubeus.exe"} }

func (r *Rubeus) Requires() []string { return nil }

func (r *Rubeus) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	// Subcommand arrives as cmd.Subcommand (parser eats the first non-flag
	// token into Subcommand). Fall back to cmd.Args[0] for forward compat.
	sub := strings.ToLower(cmd.Subcommand)
	if sub == "" && len(cmd.Args) > 0 {
		sub = strings.ToLower(cmd.Args[0])
	}
	if sub == "" {
		return schema.ToolResult{Output: rubeusBanner + rubeusUsage}
	}

	switch sub {
	case "kerberoast":
		return r.kerberoast(cmd, state, provider)
	case "asktgt":
		return r.asktgt(cmd, state, provider)
	case "s4u":
		return r.s4u(cmd, state, provider)
	case "dump":
		return r.dump(cmd, state)
	case "ptt":
		return r.ptt(cmd, state)
	case "triage":
		return r.triage(state)
	default:
		return schema.ToolResult{
			Output: rubeusBanner + rubeusUsage,
		}
	}
}

// ── kerberoast ────────────────────────────────────────────────────────────────

func (r *Rubeus) kerberoast(
	cmd schema.ParsedCommand,
	_ *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	targetUser := flagValue(cmd.Flags, "user")
	nowrap := hasFlag(cmd.Flags, "nowrap")

	candidates := provider.GetUsersWithSPN()
	if len(candidates) == 0 {
		return schema.ToolResult{
			Output: "[*] Action: Kerberoasting\r\n\r\n[-] No users found with SPNs.\r\n",
		}
	}

	// Filter by /user if specified.
	if targetUser != "" {
		var filtered []schema.User
		for _, u := range candidates {
			if strings.EqualFold(u.SAMAccountName, targetUser) {
				filtered = append(filtered, u)
				break
			}
		}
		if len(filtered) == 0 {
			return schema.ToolResult{
				Output: fmt.Sprintf(
					"[*] Action: Kerberoasting\r\n\r\n[-] User '%s' not found or has no SPN.\r\n",
					targetUser,
				),
			}
		}
		candidates = filtered
	}

	domain := providers.GetDomain(provider)

	var sb strings.Builder
	sb.WriteString("[*] Action: Kerberoasting\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] NOTICE: AES hashes will be returned for AES-enabled accounts.\r\n"))
	sb.WriteString(fmt.Sprintf("[*]         Use /ticket:X or /tgtdeleg to force RC4_HMAC for these accounts.\r\n\r\n"))
	sb.WriteString(fmt.Sprintf("[*] Target Domain          : %s\r\n", domain))
	sb.WriteString(fmt.Sprintf("[*] Searching path 'LDAP://%s/DC=%s' for '%s'\r\n\r\n",
		domain,
		dcComponents(domain),
		"(&(samAccountType=805306368)(servicePrincipalName=*)(!samAccountName=krbtgt)(!(UserAccountControl:1.2.840.113556.1.4.803:=2)))",
	))
	sb.WriteString(fmt.Sprintf("[*] Total kerberoastable users : %d\r\n\r\n\r\n", len(candidates)))

	var effects []schema.SideEffect

	for _, u := range candidates {
		spnStr := ""
		if len(u.SPNs) > 0 { spnStr = u.SPNs[0] }
		hashData := deterministicHex(fnv.New64a().Sum64()^uint64(len(u.SAMAccountName)*31+len(spnStr)*17), 32)
		h2 := fnv.New64a()
		h2.Write([]byte("krb5tgs:" + u.SAMAccountName + spnStr))
		challenge := deterministicHex(h2.Sum64(), 32)
		encrypted := deterministicHex(h2.Sum64()^0xdeadbeefcafe1234, 512)
		_ = hashData
		hash := kerberoastHash(u.SAMAccountName, domain, spnStr, challenge+"$"+encrypted)

		sb.WriteString(fmt.Sprintf("[*] SamAccountName         : %s\r\n", u.SAMAccountName))
		sb.WriteString(fmt.Sprintf("[*] DistinguishedName      : CN=%s,CN=Users,DC=%s\r\n",
			u.SAMAccountName,
			dcComponents(domain),
		))
		sb.WriteString(fmt.Sprintf("[*] ServicePrincipalName   : %s\r\n", spnStr))
		sb.WriteString(fmt.Sprintf("[*] PwdLastSet             : 11/8/2026 9:14:22 AM\r\n"))
		sb.WriteString(fmt.Sprintf("[*] Supported ETypes       : RC4_HMAC_DEFAULT\r\n"))
		if nowrap {
			sb.WriteString(fmt.Sprintf("[*] Hash                   : %s\r\n\r\n\r\n", hash))
		} else {
			// Real Rubeus wraps hash at ~80 chars per line with leading spaces.
			sb.WriteString("[*] Hash                   : ")
			sb.WriteString(wrapHash(hash, 80))
			sb.WriteString("\r\n\r\n")
		}

		effects = append(effects, schema.SideEffect{
			Type:       schema.SideEffectTypeLoot,
			LootType:   "spn_hash",
			LootValue:  hash,
			LootSource: fmt.Sprintf("spn_hashes.%s", u.SAMAccountName),
		})
	}

	return schema.ToolResult{
		Output:  sb.String(),
		Effects: effects,
	}
}

// ── asktgt ────────────────────────────────────────────────────────────────────

func (r *Rubeus) asktgt(
	cmd schema.ParsedCommand,
	_ *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	password := flagValue(cmd.Flags, "password")
	ntlm := flagValue(cmd.Flags, "ntlm")

	if user == "" || (password == "" && ntlm == "") {
		return schema.ToolResult{
			Output: "[-] /user and (/password or /ntlm) are required\r\n",
		}
	}

	domain := providers.GetDomain(provider)
	domainUpper := strings.ToUpper(domain)

	// Validate credentials.
	var valid bool
	if ntlm != "" {
		valid = provider.ValidateNTLM(user, ntlm)
	} else {
		valid = provider.ValidateCredentials(user, password)
	}

	if !valid {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[*] Action: Ask TGT\r\n\r\n[*] Using %s\r\n\r\n[-] Kerberos SessionError: KDC_ERR_PREAUTH_FAILED\r\n",
				credLine(user, domainUpper, password, ntlm),
			),
		}
	}

	blob := tgtBlob(user, domain)
	hash := ntlm
	if hash == "" {
		hash = ntlmFromPassword(password)
	}
	luid := fakeLUID()

	ticket := schema.Ticket{
		TicketType: "TGT",
		Identity:   user,
		Target:     domainUpper,
		B64:        hash,
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(10 * time.Hour),
	}
	ticketJSON, _ := json.Marshal(ticket)

	var sb strings.Builder
	sb.WriteString("[*] Action: Ask TGT\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Using %s\r\n\r\n", credLine(user, domainUpper, password, ntlm)))
	sb.WriteString(fmt.Sprintf("[*] Building AS-REQ (w/ preauth) for: '%s\\%s'\r\n", domainUpper, user))
	sb.WriteString(fmt.Sprintf("[+] TGT request successful!\r\n"))
	sb.WriteString(fmt.Sprintf("[*] base64(ticket.kirbi):\r\n\r\n"))
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  krbtgt/%s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", user))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", time.Now().Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", time.Now().Add(10*time.Hour).Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", time.Now().Add(7*24*time.Hour).Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  Flags                    :  name_canonicalize, pre_authent, initial, renewable, forwardable\r\n"))
	sb.WriteString(fmt.Sprintf("  KeyType                  :  rc4_hmac\r\n"))
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob(user)))
	sb.WriteString(fmt.Sprintf("  ASREP (key)              :  %s\r\n\r\n", strings.ToUpper(ntlmFromPassword(user))))
	doPTT := hasFlag(cmd.Flags, "ptt")

	if doPTT {
		sb.WriteString(fmt.Sprintf("[+] Ticket successfully imported into the current session!\r\n"))
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
	} else {
		fmt.Fprintf(&sb, "[*] Saving ticket to '%s.kirbi'\r\n", user)
		sb.WriteString(fmt.Sprintf("[*] Ticket written to disk!\r\n"))
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
	}

	effects := []schema.SideEffect{
		{
			Type:       schema.SideEffectTypeTicket,
			TicketType: "TGT",
			Identity:   user,
			Target:     domainUpper,
			TicketB64:  string(ticketJSON),
		},
	}
	if doPTT {
		effects = append(effects, schema.SideEffect{
			Type:    schema.SideEffectTypeSetSessionUser,
			NewUser: user,
		})
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// ── s4u ───────────────────────────────────────────────────────────────────────

func (r *Rubeus) s4u(
	cmd schema.ParsedCommand,
	_ *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	impersonate := flagValue(cmd.Flags, "impersonateuser")
	msdsspn := flagValue(cmd.Flags, "msdsspn")
	ticket := flagValue(cmd.Flags, "ticket")
	ntlm := flagValue(cmd.Flags, "ntlm")

	if user == "" || impersonate == "" || msdsspn == "" || (ticket == "" && ntlm == "") {
		return schema.ToolResult{
			Output: "[-] /user, /impersonateuser, /msdsspn, and (/ticket or /ntlm) are required\r\n",
		}
	}

	domain := providers.GetDomain(provider)
	domainUpper := strings.ToUpper(domain)

	// Extract target host from SPN (e.g. "cifs/dc01.corp.local" → "dc01")
	targetHost := spnHost(msdsspn)

	// Check delegation config.
	delegations := provider.GetDelegations()
	var found bool
	for _, d := range delegations {
		if strings.EqualFold(spnHost(d.Account), targetHost) {
			for _, allowed := range d.AllowedTargets {
				if strings.EqualFold(allowed, msdsspn) || strings.EqualFold(spnHost(allowed), targetHost) {
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] No S4U delegation path found for %s\r\n", msdsspn),
		}
	}

	blob := tgtBlob(impersonate, domain)

	var sb strings.Builder
	sb.WriteString("[*] Action: S4U\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Building S4U2self request for: '%s@%s'\r\n", user, domainUpper))
	sb.WriteString(fmt.Sprintf("[*] Using domain controller: %s\r\n", domain))
	sb.WriteString(fmt.Sprintf("[+] S4U2self success!\r\n"))
	sb.WriteString(fmt.Sprintf("[*] Got a TGS for '%s' to '%s@%s'\r\n", impersonate, user, domainUpper))
	sb.WriteString(fmt.Sprintf("[*] base64(ticket.kirbi) for SPN '%s':\r\n\r\n", msdsspn))
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Impersonating user '%s' to target SPN '%s'\r\n", impersonate, msdsspn))
	sb.WriteString(fmt.Sprintf("[*] Building S4U2proxy request for service: '%s'\r\n", msdsspn))
	sb.WriteString(fmt.Sprintf("[+] S4U2proxy success!\r\n"))
	sb.WriteString(fmt.Sprintf("[*] base64(ticket.kirbi) for SPN '%s':\r\n\r\n", msdsspn))
	sb.WriteString(wrapBase64(tgtBlob(impersonate+msdsspn, domain), 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[+] Ticket successfully imported!\r\n"))

	tkt := schema.Ticket{
		TicketType: "TGS",
		Identity:   impersonate,
		Target:     domainUpper,
		B64:        "",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(10 * time.Hour),
	}
	tktJSON, _ := json.Marshal(tkt)

	return schema.ToolResult{
		Output: sb.String(),
		Effects: []schema.SideEffect{
			{
				Type:  schema.SideEffectTypeTicket,
				LootSource: fmt.Sprintf("ticket.%s.tgs.%s", strings.ToLower(impersonate), strings.ToLower(targetHost)),
				LootValue:  string(tktJSON),
			},
		},
	}
}

// ── dump ──────────────────────────────────────────────────────────────────────

func (r *Rubeus) dump(_ schema.ParsedCommand, state *schema.SessionState) schema.ToolResult {
	if len(state.Tickets) == 0 {
		return schema.ToolResult{Output: "[*] Action: Dump Kerberos Ticket Data (All Users)\r\n\r\n[*] No tickets found.\r\n"}
	}

	var sb strings.Builder
	sb.WriteString("[*] Action: Dump Kerberos Ticket Data (All Users)\r\n\r\n")

	luid := fakeLUID()
	for i, t := range state.Tickets {
		sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", t.Identity))
		sb.WriteString(fmt.Sprintf("  Domain                   :  %s\r\n", t.Target))
		sb.WriteString(fmt.Sprintf("  LogonId                  :  %s\r\n", luid))
		sb.WriteString(fmt.Sprintf("  UserSID                  :  S-1-5-21-3623811015-3361044348-30300820-1121\r\n"))
		sb.WriteString(fmt.Sprintf("  AuthenticationPackage    :  Kerberos\r\n"))
		sb.WriteString(fmt.Sprintf("  LogonType                :  Network\r\n"))
		sb.WriteString(fmt.Sprintf("  LogonTime                :  11/22/2026 1:40:00 PM\r\n"))
		sb.WriteString(fmt.Sprintf("  LogonServer              :  %s\r\n", t.Target))
		sb.WriteString(fmt.Sprintf("  LogonServerDNSDomain     :  %s\r\n", strings.ToLower(t.Target)))
		sb.WriteString(fmt.Sprintf("  UserPrincipalName        :  %s@%s\r\n\r\n", t.Identity, strings.ToLower(t.Target)))
		sb.WriteString(fmt.Sprintf("    [%d] Kerberos\r\n\r\n", i))
		sb.WriteString(fmt.Sprintf("           ServiceName              :  %s\r\n", ticketService(t)))
		sb.WriteString(fmt.Sprintf("           ServiceRealm             :  %s\r\n", t.Target))
		sb.WriteString(fmt.Sprintf("           UserName                 :  %s\r\n", t.Identity))
		sb.WriteString(fmt.Sprintf("           UserRealm                :  %s\r\n", t.Target))
		sb.WriteString(fmt.Sprintf("           StartTime                :  %s\r\n", time.Now().Format("1/2/2006 3:04:05 PM")))
		sb.WriteString(fmt.Sprintf("           EndTime                  :  %s\r\n", t.ExpiresAt.Format("1/2/2006 3:04:05 PM")))
		sb.WriteString(fmt.Sprintf("           RenewTill                :  %s\r\n", time.Now().Add(7*24*time.Hour).Format("1/2/2006 3:04:05 PM")))
		sb.WriteString(fmt.Sprintf("           Flags                    :  name_canonicalize, pre_authent, renewable, forwarded, forwardable\r\n"))
		sb.WriteString(fmt.Sprintf("           KeyType                  :  rc4_hmac\r\n"))
		sb.WriteString(fmt.Sprintf("           Base64(key)              :  %s\r\n\r\n", shortBlob(t.Identity)))
	}

	return schema.ToolResult{Output: sb.String()}
}

// ── ptt ───────────────────────────────────────────────────────────────────────

func (r *Rubeus) ptt(cmd schema.ParsedCommand, state *schema.SessionState) schema.ToolResult {
	ticketArg := flagValue(cmd.Flags, "ticket")
	if ticketArg == "" {
		return schema.ToolResult{Output: "[-] /ticket is required\r\n"}
	}

	// Strip path, strip .kirbi — match against stored ticket identity
	name := ticketArg
	if idx := strings.LastIndexAny(name, `/\\`); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(strings.ToLower(name), ".kirbi")

	var matched *schema.Ticket
	for i := range state.Tickets {
		if strings.EqualFold(state.Tickets[i].Identity, name) && state.Tickets[i].TicketType == "TGT" {
			matched = &state.Tickets[i]
			break
		}
	}

	if matched == nil {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Ticket not found: %s\r\n[*] Use Rubeus dump or triage to list available tickets\r\n", ticketArg),
		}
	}

	var sb strings.Builder
	sb.WriteString("[*] Action: Import Ticket\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Importing ticket for %s @ %s\r\n", matched.Identity, matched.Target))
	sb.WriteString(fmt.Sprintf("[+] Ticket successfully imported!\r\n"))
	sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", fakeLUID()))

	return schema.ToolResult{
		Output: sb.String(),
		Effects: []schema.SideEffect{
			{
				Type:    schema.SideEffectTypeSetSessionUser,
				NewUser: matched.Identity,
			},
		},
	}
}

// ── triage ────────────────────────────────────────────────────────────────────

func (r *Rubeus) triage(state *schema.SessionState) schema.ToolResult {
	var sb strings.Builder
	sb.WriteString("[*] Action: Triage Kerberos Tickets (All Users)\r\n\r\n")

	if len(state.Tickets) == 0 {
		sb.WriteString("[*] No tickets found.\r\n")
		return schema.ToolResult{Output: sb.String()}
	}

	luid := fakeLUID()
	sb.WriteString(fmt.Sprintf(" %-20s  %-20s  %-40s  %-25s\r\n",
		"LUID", "UserName", "Service", "EndTime"))
	sb.WriteString(fmt.Sprintf(" %-20s  %-20s  %-40s  %-25s\r\n",
		"----", "--------", "-------", "-------"))

	for _, t := range state.Tickets {
		sb.WriteString(fmt.Sprintf(" %-20s  %-20s  %-40s  %-25s\r\n",
			luid,
			fmt.Sprintf("%s @ %s", t.Identity, t.Target),
			ticketService(t),
			t.ExpiresAt.Format("1/2/2006 3:04:05 PM"),
		))
	}

	return schema.ToolResult{Output: sb.String()}
}

// ── Hash / blob helpers ───────────────────────────────────────────────────────

// kerberoastHash produces a deterministic $krb5tgs$23$*...*$<32hex>$<256hex> string.

// tgtBlob produces a deterministic base64 string (~500 chars) simulating a kirbi blob.
func tgtBlob(username, domain string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte("tgt:" + username + "@" + domain))
	seed := h.Sum64()
	raw := deterministicBytes(seed, 375) // 375 bytes → ~500 base64 chars
	return base64.StdEncoding.EncodeToString(raw)
}

// fakeLUID returns the static LUID used throughout output.
func fakeLUID() string {
	return "0x0000000000403f65"
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// deterministicHex generates a hex string of exactly n characters using an FNV seed.
func deterministicHex(seed uint64, n int) string {
	const hexChars = "0123456789abcdef"
	buf := make([]byte, n)
	h := fnv.New64a()
	block := seed
	pos := 0
	for pos < n {
		h.Reset()
		var tmp [8]byte
		for i := range tmp {
			tmp[i] = byte(block >> (i * 8))
		}
		_, _ = h.Write(tmp[:])
		block = h.Sum64()
		for i := 0; i < 8 && pos < n; i++ {
			buf[pos] = hexChars[(block>>(i*4))&0xf]
			pos++
		}
	}
	return string(buf)
}

// deterministicBytes generates n pseudorandom bytes from a seed.
func deterministicBytes(seed uint64, n int) []byte {
	buf := make([]byte, n)
	h := fnv.New64a()
	block := seed
	pos := 0
	for pos < n {
		h.Reset()
		var tmp [8]byte
		for i := range tmp {
			tmp[i] = byte(block >> (i * 8))
		}
		_, _ = h.Write(tmp[:])
		block = h.Sum64()
		for i := 0; i < 8 && pos < n; i++ {
			buf[pos] = byte(block >> (i * 8))
			pos++
		}
	}
	return buf
}

// ntlmFromPassword returns a deterministic 32-char hex string standing in for an NTLM hash.
func ntlmFromPassword(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte("ntlm:" + s))
	return deterministicHex(h.Sum64(), 32)
}

// shortBlob returns a short (~24 char) base64 string for key fields.
func shortBlob(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte("key:" + s))
	raw := deterministicBytes(h.Sum64(), 18)
	return base64.StdEncoding.EncodeToString(raw)
}

// wrapHash wraps a hash string at width characters with leading spaces on continuation lines.
func wrapHash(hash string, width int) string {
	if len(hash) <= width {
		return hash
	}
	var sb strings.Builder
	sb.WriteString(hash[:width])
	rest := hash[width:]
	for len(rest) > 0 {
		chunk := rest
		if len(chunk) > width {
			chunk = rest[:width]
		}
		sb.WriteString("\r\n                             ")
		sb.WriteString(chunk)
		rest = rest[len(chunk):]
	}
	return sb.String()
}

// wrapBase64 wraps a base64 blob at width, indented 10 spaces.
func wrapBase64(blob string, width int) string {
	var sb strings.Builder
	for len(blob) > 0 {
		chunk := blob
		if len(chunk) > width {
			chunk = blob[:width]
		}
		sb.WriteString("          ")
		sb.WriteString(chunk)
		sb.WriteString("\r\n")
		blob = blob[len(chunk):]
	}
	return sb.String()
}

// dcComponents splits "corp.local" → ["corp","local"].

// spnHost extracts the hostname from an SPN like "cifs/dc01.corp.local" → "dc01".
func spnHost(spn string) string {
	// Strip service class prefix.
	if idx := strings.Index(spn, "/"); idx >= 0 {
		spn = spn[idx+1:]
	}
	// Strip domain suffix.
	if idx := strings.Index(spn, "."); idx >= 0 {
		spn = spn[:idx]
	}
	return strings.ToLower(spn)
}

// ticketService returns a display string for the service field in dump/triage.
func ticketService(t schema.Ticket) string {
	if t.TicketType == "TGT" {
		return fmt.Sprintf("krbtgt/%s", t.Target)
	}
	return fmt.Sprintf("cifs/%s", strings.ToLower(t.Target))
}

// credLine formats the credential line shown in asktgt output.
func credLine(user, domain, password, ntlm string) string {
	if ntlm != "" {
		return fmt.Sprintf("rc4_hmac hash     : %s", ntlm)
	}
	return fmt.Sprintf("aes256_cts_hmac_sha1 hash : %s", ntlmFromPassword(password))
}

// flagValue returns the value of flag /name or /Name (case-insensitive key lookup).

// hasFlag returns true if the flag is present (value may be empty).
