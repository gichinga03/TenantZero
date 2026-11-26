package ad

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/schema"
)

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

type Rubeus struct{}

func (r *Rubeus) Name() string    { return "rubeus" }
func (r *Rubeus) Aliases() []string { return []string{"Rubeus.exe"} }
func (r *Rubeus) Requires() []string { return nil }

func (r *Rubeus) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
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
	case "asktgs":
		return r.asktgs(cmd, state, provider)
	case "renew":
		return r.renew(cmd, state)
	case "s4u":
		return r.s4u(cmd, state, provider)
	case "ptt":
		return r.ptt(cmd, state)
	case "triage":
		return r.triage(state)
	case "dump":
		return r.dump(cmd, state)
	case "describe":
		return r.describe(cmd, state)
	case "golden":
		return r.golden(cmd, state, provider)
	case "silver":
		return r.silver(cmd, state, provider)
	default:
		return schema.ToolResult{Output: rubeusBanner + rubeusUsage}
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
	sb.WriteString("[*] NOTICE: AES hashes will be returned for AES-enabled accounts.\r\n")
	sb.WriteString("[*]         Use /ticket:X or /tgtdeleg to force RC4_HMAC for these accounts.\r\n\r\n")
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
		if len(u.SPNs) > 0 {
			spnStr = u.SPNs[0]
		}
		h2 := fnv.New64a()
		h2.Write([]byte("krb5tgs:" + u.SAMAccountName + spnStr))
		challenge := deterministicHex(h2.Sum64(), 32)
		encrypted := deterministicHex(h2.Sum64()^0xdeadbeefcafe1234, 512)
		hash := kerberoastHash(u.SAMAccountName, domain, spnStr, challenge+"$"+encrypted)

		sb.WriteString(fmt.Sprintf("[*] SamAccountName         : %s\r\n", u.SAMAccountName))
		sb.WriteString(fmt.Sprintf("[*] DistinguishedName      : CN=%s,CN=Users,DC=%s\r\n",
			u.SAMAccountName, dcComponents(domain)))
		sb.WriteString(fmt.Sprintf("[*] ServicePrincipalName   : %s\r\n", spnStr))
		sb.WriteString("[*] PwdLastSet             : 11/8/2026 9:14:22 AM\r\n")
		sb.WriteString("[*] Supported ETypes       : RC4_HMAC_DEFAULT\r\n")
		if nowrap {
			sb.WriteString(fmt.Sprintf("[*] Hash                   : %s\r\n\r\n\r\n", hash))
		} else {
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

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// ── asktgt ────────────────────────────────────────────────────────────────────

func (r *Rubeus) asktgt(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	password := flagValue(cmd.Flags, "password")
	ntlm := flagValue(cmd.Flags, "ntlm")
	rc4 := flagValue(cmd.Flags, "rc4")
	if rc4 != "" && ntlm == "" {
		ntlm = rc4
	}
	doPTT := hasFlag(cmd.Flags, "ptt")

	if user == "" || (password == "" && ntlm == "") {
		return schema.ToolResult{
			Output: "[-] /user and (/password or /rc4 or /ntlm) are required\r\n",
		}
	}

	domain := providers.GetDomain(provider)
	domainUpper := strings.ToUpper(domain)

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

	now := time.Now()
	expiresAt := now.Add(10 * time.Hour)
	renewTill := now.Add(7 * 24 * time.Hour)
	luid := generateLUID(user + domain + now.Format("150405"))
	blob := tgtBlob(user, domain)

	hashVal := ntlm
	if hashVal == "" {
		hashVal = ntlmFromPassword(password)
	}
	asrepKey := strings.ToUpper(ntlmFromPassword(user + domain))

	var sb strings.Builder
	sb.WriteString("[*] Action: Ask TGT\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Using %s\r\n\r\n", credLine(user, domainUpper, password, ntlm)))
	sb.WriteString(fmt.Sprintf("[*] Building AS-REQ (w/ preauth) for: '%s\\%s'\r\n", domainUpper, user))
	sb.WriteString("[+] TGT request successful!\r\n")
	sb.WriteString("[*] base64(ticket.kirbi):\r\n\r\n")
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  krbtgt/%s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", user))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", now.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", expiresAt.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", renewTill.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, initial, renewable, forwardable\r\n")
	sb.WriteString("  KeyType                  :  rc4_hmac\r\n")
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob(user)))
	sb.WriteString(fmt.Sprintf("  ASREP (key)              :  %s\r\n\r\n", asrepKey))

	effects := []schema.SideEffect{
		{
			Type:            schema.SideEffectTypeTicket,
			TicketType:      "TGT",
			Identity:        user,
			Target:          domainUpper,
			TicketB64:       blob,
			TicketLUID:      luid,
			TicketExpiresAt: expiresAt,
			TicketRenewTill: renewTill,
		},
	}

	if doPTT {
		sb.WriteString("[+] Ticket successfully imported into the current session!\r\n")
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
		effects = append(effects, schema.SideEffect{
			Type:    schema.SideEffectTypeSetSessionUser,
			NewUser: user,
		})
	} else {
		sb.WriteString(fmt.Sprintf("[*] Saving ticket to '%s.kirbi'\r\n", user))
		sb.WriteString("[*] Ticket written to disk!\r\n")
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
	}

	_ = state
	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// ── asktgs ────────────────────────────────────────────────────────────────────

func (r *Rubeus) asktgs(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	service := flagValue(cmd.Flags, "service")
	ticketArg := flagValue(cmd.Flags, "ticket")

	if service == "" {
		return schema.ToolResult{Output: "[-] /service is required (e.g. /service:cifs/dc01.meridian.local)\r\n"}
	}

	domain := providers.GetDomain(provider)
	domainUpper := strings.ToUpper(domain)

	// Resolve identity
	identity := user
	if identity == "" && ticketArg != "" {
		name := ticketArg
		if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
			name = name[idx+1:]
		}
		identity = strings.TrimSuffix(strings.ToLower(name), ".kirbi")
	}
	if identity == "" {
		identity = state.ActiveUser
		if idx := strings.LastIndexAny(identity, `\/`); idx >= 0 {
			identity = identity[idx+1:]
		}
	}

	// Must have a TGT in cache (or /ticket supplied)
	var tgt *schema.Ticket
	for i := range state.Tickets {
		t := &state.Tickets[i]
		if t.TicketType == "TGT" && strings.EqualFold(t.Identity, identity) {
			tgt = t
			break
		}
	}
	if tgt == nil && ticketArg == "" {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] No TGT found for '%s' — run asktgt first or supply /ticket\r\n", identity),
		}
	}

	// Enforce expiry
	if tgt != nil && time.Now().After(tgt.ExpiresAt) {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[-] TGT for '%s' has expired (EndTime: %s) — request a new one with asktgt\r\n",
				identity, tgt.ExpiresAt.Format("1/2/2006 3:04:05 PM"),
			),
		}
	}

	now := time.Now()
	expiresAt := now.Add(10 * time.Hour)
	renewTill := now.Add(7 * 24 * time.Hour)
	luid := generateLUID(identity + service + now.Format("150405"))
	blob := tgtBlob(identity+service, domain)
	targetHost := spnHost(service)

	var sb strings.Builder
	sb.WriteString("[*] Action: Ask TGS\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Using existing TGT for '%s'\r\n\r\n", identity))
	sb.WriteString(fmt.Sprintf("[*] Building TGS-REQ request for: '%s'\r\n", service))
	sb.WriteString("[+] TGS request successful!\r\n")
	sb.WriteString("[*] base64(ticket.kirbi):\r\n\r\n")
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  %s\r\n", service))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", identity))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", now.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", expiresAt.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", renewTill.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, renewable, forwardable\r\n")
	sb.WriteString("  KeyType                  :  aes256_cts_hmac_sha1\r\n")
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob(identity+service)))
	sb.WriteString(fmt.Sprintf("[*] Saving ticket to '%s_%s.kirbi'\r\n", identity, targetHost))
	sb.WriteString("[*] Ticket written to disk!\r\n")
	sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))

	return schema.ToolResult{
		Output: sb.String(),
		Effects: []schema.SideEffect{
			{
				Type:            schema.SideEffectTypeTicket,
				TicketType:      "TGS",
				Identity:        identity,
				Target:          service,
				TicketB64:       blob,
				TicketLUID:      luid,
				TicketExpiresAt: expiresAt,
				TicketRenewTill: renewTill,
			},
		},
	}
}

// ── renew ─────────────────────────────────────────────────────────────────────

func (r *Rubeus) renew(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
) schema.ToolResult {
	ticketArg := flagValue(cmd.Flags, "ticket")
	autorenew := hasFlag(cmd.Flags, "autorenew")

	name := ticketArg
	if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(strings.ToLower(name), ".kirbi")

	var matched *schema.Ticket
	for i := range state.Tickets {
		if state.Tickets[i].TicketType == "TGT" &&
			strings.EqualFold(state.Tickets[i].Identity, name) {
			matched = &state.Tickets[i]
			break
		}
	}
	if matched == nil {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Ticket not found: %s\r\n[*] Use triage to list available tickets\r\n", ticketArg),
		}
	}

	// Enforce RenewTill — can't renew past this window
	if time.Now().After(matched.RenewTill) {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[*] Action: Renew TGT\r\n\r\n[*] Using existing TGT for '%s'\r\n\r\n[-] KRB_AP_ERR_TKT_EXPIRED — ticket is past its RenewTill time (%s)\r\n[-] Request a new TGT with asktgt\r\n",
				matched.Identity,
				matched.RenewTill.Format("1/2/2006 3:04:05 PM"),
			),
		}
	}

	now := time.Now()
	newExpiry := now.Add(10 * time.Hour)
	blob := tgtBlob(matched.Identity+"renewed:"+now.Format("150405"), strings.ToLower(matched.Target))

	// Mutate in place — same LUID, same RenewTill, new ExpiresAt
	matched.ExpiresAt = newExpiry
	matched.B64 = blob

	var sb strings.Builder
	sb.WriteString("[*] Action: Renew TGT\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Using existing TGT for '%s'\r\n\r\n", matched.Identity))
	if autorenew {
		sb.WriteString("[*] Auto-renew mode — will renew every 30 minutes\r\n\r\n")
	}
	sb.WriteString("[*] Sending renewal request...\r\n")
	sb.WriteString("[+] TGT renewal successful!\r\n")
	sb.WriteString("[*] base64(ticket.kirbi):\r\n\r\n")
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  krbtgt/%s\r\n", matched.Target))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", matched.Target))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", matched.Identity))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", matched.Target))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", now.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", newExpiry.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", matched.RenewTill.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, initial, renewable, forwardable\r\n")
	sb.WriteString("  KeyType                  :  rc4_hmac\r\n")
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob(matched.Identity+"renewed")))
	sb.WriteString(fmt.Sprintf("[*] Saving ticket to '%s.kirbi'\r\n", matched.Identity))
	sb.WriteString("[*] Ticket written to disk!\r\n")

	return schema.ToolResult{Output: sb.String()}
}

// ── ptt ───────────────────────────────────────────────────────────────────────

func (r *Rubeus) ptt(cmd schema.ParsedCommand, state *schema.SessionState) schema.ToolResult {
	ticketArg := flagValue(cmd.Flags, "ticket")
	if ticketArg == "" {
		return schema.ToolResult{Output: "[-] /ticket is required\r\n"}
	}

	// Support raw base64 blobs as well as filenames
	isBase64 := !strings.Contains(ticketArg, `\`) &&
		!strings.Contains(ticketArg, "/") &&
		!strings.HasSuffix(strings.ToLower(ticketArg), ".kirbi") &&
		len(ticketArg) > 100

	var matched *schema.Ticket

	if isBase64 {
		// Match by blob value
		for i := range state.Tickets {
			if state.Tickets[i].B64 == ticketArg {
				matched = &state.Tickets[i]
				break
			}
		}
		if matched == nil {
			// Accept any base64 blob — treat as an externally supplied TGT
			// We can't validate it so we just import it and set identity from current user
			luid := generateLUID(ticketArg[:16] + time.Now().Format("150405"))
			var sb strings.Builder
			sb.WriteString("[*] Action: Import Ticket\r\n\r\n")
			sb.WriteString("[*] Importing ticket from base64 blob\r\n")
			sb.WriteString("[+] Ticket successfully imported!\r\n")
			sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
			return schema.ToolResult{Output: sb.String()}
		}
	} else {
		name := ticketArg
		if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
			name = name[idx+1:]
		}
		name = strings.TrimSuffix(strings.ToLower(name), ".kirbi")

		// Prefer TGT match, fall back to any ticket for that identity
		for i := range state.Tickets {
			t := &state.Tickets[i]
			if strings.EqualFold(t.Identity, name) {
				if t.TicketType == "TGT" {
					matched = t
					break
				}
				if matched == nil {
					matched = t
				}
			}
		}
		// Also match TGS tickets by sanitized filename stem:
		// "administrator_sqlsvr01_meridian_local_1433" -> identity=Administrator target=MSSQLSvc/SQLSVR01...
		if matched == nil {
			for i := range state.Tickets {
				t := &state.Tickets[i]
				expected := strings.ToLower(t.Identity) + "_" + sanitizeSPN(t.Target)
				if strings.EqualFold(expected, name) {
					matched = t
					break
				}
			}
		}
	}

	if matched == nil {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[-] Ticket not found: %s\r\n[*] Use Rubeus triage to list available tickets\r\n",
				ticketArg,
			),
		}
	}

	// Enforce expiry
	if time.Now().After(matched.ExpiresAt) {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[-] Ticket for '%s' has expired (EndTime: %s)\r\n[-] Use Rubeus renew or request a new ticket\r\n",
				matched.Identity,
				matched.ExpiresAt.Format("1/2/2006 3:04:05 PM"),
			),
		}
	}

	luid := matched.LUID
	if luid == "" {
		luid = generateLUID(matched.Identity + matched.Target)
		matched.LUID = luid
	}

	var sb strings.Builder
	sb.WriteString("[*] Action: Import Ticket\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Importing ticket for %s @ %s\r\n", matched.Identity, matched.Target))
	sb.WriteString("[+] Ticket successfully imported!\r\n")
	sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))

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

	sb.WriteString(fmt.Sprintf(" %-20s  %-30s  %-44s  %-25s\r\n",
		"LUID", "UserName", "Service", "EndTime"))
	sb.WriteString(fmt.Sprintf(" %-20s  %-30s  %-44s  %-25s\r\n",
		"----", "--------", "-------", "-------"))

	for i := range state.Tickets {
		t := &state.Tickets[i]
		luid := t.LUID
		if luid == "" {
			luid = generateLUID(t.Identity + t.Target + t.IssuedAt.Format("150405"))
			t.LUID = luid
		}
		svc := ticketService(*t)
		username := fmt.Sprintf("%s @ %s", t.Identity, t.Target)
		endTime := t.ExpiresAt.Format("1/2/2006 3:04:05 PM")
		sb.WriteString(fmt.Sprintf(" %-20s  %-30s  %-44s  %-25s\r\n",
			luid, username, svc, endTime))
	}

	return schema.ToolResult{Output: sb.String()}
}

// ── dump ──────────────────────────────────────────────────────────────────────

func (r *Rubeus) dump(_ schema.ParsedCommand, state *schema.SessionState) schema.ToolResult {
	if len(state.Tickets) == 0 {
		return schema.ToolResult{
			Output: "[*] Action: Dump Kerberos Ticket Data (All Users)\r\n\r\n[*] No tickets found.\r\n",
		}
	}

	var sb strings.Builder
	sb.WriteString("[*] Action: Dump Kerberos Ticket Data (All Users)\r\n\r\n")

	for i, t := range state.Tickets {
		luid := t.LUID
		if luid == "" {
			luid = generateLUID(t.Identity + t.Target + t.IssuedAt.Format("150405"))
			state.Tickets[i].LUID = luid
		}
		domain := t.Target
		if strings.Contains(domain, "/") {
			// TGS — target is SPN, extract domain from it
			parts := strings.SplitN(domain, "/", 2)
			if len(parts) > 1 {
				host := parts[1]
				if idx := strings.Index(host, "."); idx >= 0 {
					domain = host[idx+1:]
				}
			}
		}
		domainLower := strings.ToLower(domain)

		sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", t.Identity))
		sb.WriteString(fmt.Sprintf("  Domain                   :  %s\r\n", strings.ToUpper(domain)))
		sb.WriteString(fmt.Sprintf("  LogonId                  :  %s\r\n", luid))
		sb.WriteString("  UserSID                  :  S-1-5-21-3623811015-3361044348-30300820-1121\r\n")
		sb.WriteString("  AuthenticationPackage    :  Kerberos\r\n")
		sb.WriteString("  LogonType                :  Network\r\n")
		sb.WriteString(fmt.Sprintf("  LogonTime                :  %s\r\n", t.IssuedAt.Format("1/2/2006 3:04:05 PM")))
		sb.WriteString(fmt.Sprintf("  LogonServer              :  %s\r\n", strings.ToUpper(domain)))
		sb.WriteString(fmt.Sprintf("  LogonServerDNSDomain     :  %s\r\n", domainLower))
		sb.WriteString(fmt.Sprintf("  UserPrincipalName        :  %s@%s\r\n\r\n", t.Identity, domainLower))
		sb.WriteString(fmt.Sprintf("    [%d] Kerberos\r\n\r\n", i))
		sb.WriteString(fmt.Sprintf("           ServiceName              :  %s\r\n", ticketService(t)))
		sb.WriteString(fmt.Sprintf("           ServiceRealm             :  %s\r\n", strings.ToUpper(domain)))
		sb.WriteString(fmt.Sprintf("           UserName                 :  %s\r\n", t.Identity))
		sb.WriteString(fmt.Sprintf("           UserRealm                :  %s\r\n", strings.ToUpper(domain)))
		sb.WriteString(fmt.Sprintf("           StartTime                :  %s\r\n", t.IssuedAt.Format("1/2/2006 3:04:05 PM")))
		sb.WriteString(fmt.Sprintf("           EndTime                  :  %s\r\n", t.ExpiresAt.Format("1/2/2006 3:04:05 PM")))
		renewTill := t.RenewTill
		if renewTill.IsZero() {
			renewTill = t.IssuedAt.Add(7 * 24 * time.Hour)
		}
		sb.WriteString(fmt.Sprintf("           RenewTill                :  %s\r\n", renewTill.Format("1/2/2006 3:04:05 PM")))
		sb.WriteString("           Flags                    :  name_canonicalize, pre_authent, renewable, forwarded, forwardable\r\n")
		sb.WriteString("           KeyType                  :  rc4_hmac\r\n")
		sb.WriteString(fmt.Sprintf("           Base64(key)              :  %s\r\n\r\n", shortBlob(t.Identity)))
	}

	return schema.ToolResult{Output: sb.String()}
}

// ── describe ──────────────────────────────────────────────────────────────────

func (r *Rubeus) describe(cmd schema.ParsedCommand, state *schema.SessionState) schema.ToolResult {
	ticketArg := flagValue(cmd.Flags, "ticket")
	if ticketArg == "" {
		return schema.ToolResult{Output: "[-] /ticket is required\r\n"}
	}

	name := ticketArg
	if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(strings.ToLower(name), ".kirbi")

	var matched *schema.Ticket
	for i := range state.Tickets {
		if strings.EqualFold(state.Tickets[i].Identity, name) {
			matched = &state.Tickets[i]
			break
		}
	}
	if matched == nil {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[-] Ticket not found: %s\r\n[*] Use triage to list available tickets\r\n",
				ticketArg,
			),
		}
	}

	renewTill := matched.RenewTill
	if renewTill.IsZero() {
		renewTill = matched.IssuedAt.Add(7 * 24 * time.Hour)
	}

	var sb strings.Builder
	sb.WriteString("[*] Action: Describe Ticket\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  %s\r\n", ticketService(*matched)))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", matched.Target))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", matched.Identity))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", matched.Target))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", matched.IssuedAt.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", matched.ExpiresAt.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", renewTill.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, initial, renewable, forwardable\r\n")
	sb.WriteString("  KeyType                  :  rc4_hmac\r\n")
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob(matched.Identity)))
	sb.WriteString(fmt.Sprintf("  Ticket type              :  %s\r\n", matched.TicketType))

	return schema.ToolResult{Output: sb.String()}
}

// ── golden ────────────────────────────────────────────────────────────────────

func (r *Rubeus) golden(
	cmd schema.ParsedCommand,
	_ *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	ntlm := flagValue(cmd.Flags, "rc4")
	aes256 := flagValue(cmd.Flags, "aes256")
	sid := flagValue(cmd.Flags, "sid")
	id := flagValue(cmd.Flags, "id")
	groups := flagValue(cmd.Flags, "groups")
	doPTT := hasFlag(cmd.Flags, "ptt")

	if user == "" {
		return schema.ToolResult{Output: "[-] /user is required\r\n"}
	}
	if ntlm == "" && aes256 == "" {
		return schema.ToolResult{Output: "[-] /rc4 or /aes256 (krbtgt hash) is required\r\n"}
	}

	domain := providers.GetDomain(provider)
	domainUpper := strings.ToUpper(domain)

	if sid == "" {
		sid = "S-1-5-21-3623811015-3361044348-30300820"
	}
	if id == "" {
		id = "500"
	}
	if groups == "" {
		groups = "520,512,519,544,513,516,527,498,518"
	}

	keyType := "rc4_hmac"
	keyHash := ntlm
	if aes256 != "" {
		keyType = "aes256_cts_hmac_sha1"
		keyHash = aes256
	}

	now := time.Now()
	// Golden tickets: 10 year lifetime, same for RenewTill
	expiresAt := now.Add(10 * 365 * 24 * time.Hour)
	renewTill := expiresAt
	luid := generateLUID("golden:" + user + now.Format("150405"))
	blob := tgtBlob("golden:"+user, domain)

	var sb strings.Builder
	sb.WriteString("[*] Action: Build TGT\r\n\r\n")
	sb.WriteString("[*] Building KERB-CRED for the Golden Ticket\r\n")
	sb.WriteString(fmt.Sprintf("[*] Domain         : %s (%s)\r\n", domainUpper, domain))
	sb.WriteString(fmt.Sprintf("[*] SID            : %s\r\n", sid))
	sb.WriteString(fmt.Sprintf("[*] UserId         : %s\r\n", id))
	sb.WriteString(fmt.Sprintf("[*] Groups         : %s\r\n", groups))
	sb.WriteString(fmt.Sprintf("[*] EncryptionKey  : %s\r\n\r\n", keyHash))
	sb.WriteString(fmt.Sprintf("[*] Lifetime       : %s to %s\r\n",
		now.Format("1/2/2006 3:04:05 PM"),
		expiresAt.Format("1/2/2006 3:04:05 PM"),
	))
	sb.WriteString("[*] Generated KERB-CRED\r\n\r\n")
	sb.WriteString("[*] base64(ticket.kirbi):\r\n\r\n")
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  krbtgt/%s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", user))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", now.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", expiresAt.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", renewTill.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, initial, renewable, forwardable\r\n")
	sb.WriteString(fmt.Sprintf("  KeyType                  :  %s\r\n", keyType))
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob("golden:"+user)))

	effects := []schema.SideEffect{
		{
			Type:            schema.SideEffectTypeTicket,
			TicketType:      "TGT",
			Identity:        user,
			Target:          domainUpper,
			TicketB64:       blob,
			TicketLUID:      luid,
			TicketExpiresAt: expiresAt,
			TicketRenewTill: renewTill,
		},
	}

	if doPTT {
		sb.WriteString("[+] Golden ticket successfully imported!\r\n")
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
		effects = append(effects, schema.SideEffect{
			Type:    schema.SideEffectTypeSetSessionUser,
			NewUser: user,
		})
	} else {
		sb.WriteString(fmt.Sprintf("[*] Saving ticket to '%s_golden.kirbi'\r\n", user))
		sb.WriteString("[*] Ticket written to disk!\r\n")
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// ── silver ────────────────────────────────────────────────────────────────────

func (r *Rubeus) silver(
	cmd schema.ParsedCommand,
	_ *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	service := flagValue(cmd.Flags, "service")
	ntlm := flagValue(cmd.Flags, "rc4")
	aes256 := flagValue(cmd.Flags, "aes256")
	sid := flagValue(cmd.Flags, "sid")
	id := flagValue(cmd.Flags, "id")
	doPTT := hasFlag(cmd.Flags, "ptt")

	if user == "" || service == "" {
		return schema.ToolResult{Output: "[-] /user and /service are required\r\n"}
	}
	if ntlm == "" && aes256 == "" {
		return schema.ToolResult{Output: "[-] /rc4 or /aes256 (service account machine hash) is required\r\n"}
	}

	domain := providers.GetDomain(provider)
	domainUpper := strings.ToUpper(domain)
	targetHost := spnHost(service)

	if sid == "" {
		sid = "S-1-5-21-3623811015-3361044348-30300820"
	}
	if id == "" {
		id = "500"
	}

	keyType := "rc4_hmac"
	keyHash := ntlm
	if aes256 != "" {
		keyType = "aes256_cts_hmac_sha1"
		keyHash = aes256
	}

	now := time.Now()
	expiresAt := now.Add(10 * time.Hour)
	renewTill := now.Add(7 * 24 * time.Hour)
	luid := generateLUID("silver:" + user + service + now.Format("150405"))
	blob := tgtBlob("silver:"+user+service, domain)

	var sb strings.Builder
	sb.WriteString("[*] Action: Build Service Ticket\r\n\r\n")
	sb.WriteString("[*] Building KERB-CRED for the Silver Ticket\r\n")
	sb.WriteString(fmt.Sprintf("[*] Domain         : %s (%s)\r\n", domainUpper, domain))
	sb.WriteString(fmt.Sprintf("[*] SID            : %s\r\n", sid))
	sb.WriteString(fmt.Sprintf("[*] UserId         : %s\r\n", id))
	sb.WriteString(fmt.Sprintf("[*] ServiceName    : %s\r\n", service))
	sb.WriteString(fmt.Sprintf("[*] EncryptionKey  : %s\r\n\r\n", keyHash))
	sb.WriteString("[*] Generated KERB-CRED\r\n\r\n")
	sb.WriteString("[*] base64(ticket.kirbi):\r\n\r\n")
	sb.WriteString(wrapBase64(blob, 80))
	sb.WriteString("\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  %s\r\n", service))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", user))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", domainUpper))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", now.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", expiresAt.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", renewTill.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, renewable, forwardable\r\n")
	sb.WriteString(fmt.Sprintf("  KeyType                  :  %s\r\n", keyType))
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", shortBlob("silver:"+user+service)))

	effects := []schema.SideEffect{
		{
			Type:            schema.SideEffectTypeTicket,
			TicketType:      "TGS",
			Identity:        user,
			Target:          service,
			TicketB64:       blob,
			TicketLUID:      luid,
			TicketExpiresAt: expiresAt,
			TicketRenewTill: renewTill,
		},
	}

	if doPTT {
		sb.WriteString("[+] Silver ticket successfully imported!\r\n")
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
		effects = append(effects, schema.SideEffect{
			Type:    schema.SideEffectTypeSetSessionUser,
			NewUser: user,
		})
	} else {
		sb.WriteString(fmt.Sprintf("[*] Saving ticket to '%s_%s.kirbi'\r\n", user, targetHost))
		sb.WriteString("[*] Ticket written to disk!\r\n")
		sb.WriteString(fmt.Sprintf("[*] LUID                   :  %s\r\n", luid))
	}

	return schema.ToolResult{Output: sb.String(), Effects: effects}
}

// ── s4u ───────────────────────────────────────────────────────────────────────

func (r *Rubeus) s4u(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	user := flagValue(cmd.Flags, "user")
	impersonate := flagValue(cmd.Flags, "impersonateuser")
	msdsspn := flagValue(cmd.Flags, "msdsspn")
	ntlmFlag := flagValue(cmd.Flags, "ntlm")
	if ntlmFlag == "" {
		ntlmFlag = flagValue(cmd.Flags, "rc4")
	}
	_ = ntlmFlag

	if user == "" || impersonate == "" || msdsspn == "" {
		return schema.ToolResult{
			Output: "[*] Action: S4U\r\n\r\n[-] /user, /impersonateuser, /msdsspn are all required.\r\n",
		}
	}

	deleg, ok := provider.GetDelegation(user)
	if !ok || strings.ToLower(deleg.Type) != "constrained" {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[*] Action: S4U\r\n\r\n[-] %s does not have constrained delegation configured.\r\n",
				user,
			),
		}
	}

	allowed := false
	for _, t := range deleg.AllowedTargets {
		if strings.EqualFold(t, msdsspn) {
			allowed = true
			break
		}
	}
	if !allowed {
		return schema.ToolResult{
			Output: fmt.Sprintf(
				"[*] Action: S4U\r\n\r\n[-] %s is not in %s allowed delegation targets.\r\n",
				msdsspn, user,
			),
		}
	}

	domain := providers.GetDomain(provider)
	now := time.Now()
	endTime := now.Add(10 * time.Hour)
	luid := pseudoLUID(user + impersonate + msdsspn)
	b64 := pseudoB64(impersonate + msdsspn + user)

	var sb strings.Builder
	sb.WriteString("[*] Action: S4U\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Using existing TGT for '%s'\r\n\r\n", user))
	sb.WriteString("[*] Building S4U2self request for: '" + user + "@" + strings.ToUpper(domain) + "'\r\n")
	sb.WriteString("[+] S4U2self success!\r\n\r\n")
	sb.WriteString(fmt.Sprintf("[*] Impersonating user '%s' to target SPN '%s'\r\n", impersonate, msdsspn))
	sb.WriteString("[*] Building S4U2proxy request for service: '" + msdsspn + "'\r\n")
	sb.WriteString("[+] S4U2proxy success!\r\n\r\n")
	sb.WriteString("[*] base64(ticket.kirbi) for SPN '" + msdsspn + "':\r\n\r\n")
	sb.WriteString("          " + wrapB64(b64, 80) + "\r\n\r\n")
	sb.WriteString(fmt.Sprintf("  ServiceName              :  %s\r\n", msdsspn))
	sb.WriteString(fmt.Sprintf("  ServiceRealm             :  %s\r\n", strings.ToUpper(domain)))
	sb.WriteString(fmt.Sprintf("  UserName                 :  %s\r\n", strings.ToLower(impersonate)))
	sb.WriteString(fmt.Sprintf("  UserRealm                :  %s\r\n", strings.ToUpper(domain)))
	sb.WriteString(fmt.Sprintf("  StartTime                :  %s\r\n", now.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  EndTime                  :  %s\r\n", endTime.Format("1/2/2006 3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("  RenewTill                :  %s\r\n", now.Add(7*24*time.Hour).Format("1/2/2006 3:04:05 PM")))
	sb.WriteString("  Flags                    :  name_canonicalize, pre_authent, renewable, forwardable, ok_as_delegate\r\n")
	sb.WriteString("  KeyType                  :  aes256_cts_hmac_sha1\r\n")
	sb.WriteString(fmt.Sprintf("  Base64(key)              :  %s\r\n", pseudoB64(user+"s4ukey")[:24]))
	sb.WriteString("[+] Ticket successfully imported!\r\n")
	sb.WriteString(fmt.Sprintf("[*] LUID                   :  0x%016x\r\n", luid))
	filename := strings.ToLower(impersonate) + "_" + sanitizeSPN(msdsspn) + ".kirbi"
	sb.WriteString(fmt.Sprintf("[*] Saving ticket to '%s'\r\n", filename))
	sb.WriteString("[*] Ticket written to disk!\r\n")

	effects := []schema.SideEffect{
		{
			Type:       schema.SideEffectTypeTicket,
			TicketType: "TGS",
			Identity:   impersonate,
			Target:     msdsspn,
		},
	}
	return schema.ToolResult{Output: sb.String(), Effects: effects}
}


func generateLUID(seed string) string {
	h := fnv.New32a()
	h.Write([]byte(seed))
	return fmt.Sprintf("0x%016x", uint64(h.Sum32()))
}

func tgtBlob(username, domain string) string {
	h := fnv.New64a()
	h.Write([]byte("tgt:" + username + "@" + domain))
	raw := deterministicBytes(h.Sum64(), 375)
	return base64.StdEncoding.EncodeToString(raw)
}

func ntlmFromPassword(s string) string {
	h := fnv.New64a()
	h.Write([]byte("ntlm:" + s))
	return deterministicHex(h.Sum64(), 32)
}

func shortBlob(s string) string {
	h := fnv.New64a()
	h.Write([]byte("key:" + s))
	raw := deterministicBytes(h.Sum64(), 18)
	return base64.StdEncoding.EncodeToString(raw)
}

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
		h.Write(tmp[:])
		block = h.Sum64()
		for i := 0; i < 8 && pos < n; i++ {
			buf[pos] = hexChars[(block>>(i*4))&0xf]
			pos++
		}
	}
	return string(buf)
}

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
		h.Write(tmp[:])
		block = h.Sum64()
		for i := 0; i < 8 && pos < n; i++ {
			buf[pos] = byte(block >> (i * 8))
			pos++
		}
	}
	return buf
}

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

func spnHost(spn string) string {
	if idx := strings.Index(spn, "/"); idx >= 0 {
		spn = spn[idx+1:]
	}
	if idx := strings.Index(spn, "."); idx >= 0 {
		spn = spn[:idx]
	}
	return strings.ToLower(spn)
}

func ticketService(t schema.Ticket) string {
	if t.TicketType == "TGT" {
		return fmt.Sprintf("krbtgt/%s", t.Target)
	}
	// For TGS, Target is already the full SPN
	return t.Target
}

func credLine(user, domain, password, ntlm string) string {
	if ntlm != "" {
		return fmt.Sprintf("rc4_hmac hash     : %s", ntlm)
	}
	return fmt.Sprintf("aes256_cts_hmac_sha1 hash : %s", ntlmFromPassword(password))
}

// pseudoLUID returns a stable fake LUID from a seed string.
func pseudoLUID(seed string) uint64 {
	h := fnv.New64a()
	h.Write([]byte("luid:" + seed))
	return h.Sum64() & 0x00ffffffffffffff
}

// sanitizeSPN converts "MSSQLSvc/SQLSVR01.meridian.local:1433" → "sqlsvr01_1433"
// for use in filenames.
func sanitizeSPN(spn string) string {
	s := strings.ToLower(spn)
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}

// wrapB64 wraps a base64 string at width chars with 10-space indent.
func wrapB64(s string, width int) string {
	var out strings.Builder
	for len(s) > width {
		out.WriteString(s[:width] + "\r\n          ")
		s = s[width:]
	}
	out.WriteString(s)
	return out.String()
}

// pseudoB64 returns a deterministic base64-looking string seeded from s.
func pseudoB64(s string) string {
	h := fnv.New64a()
	h.Write([]byte("b64:" + s))
	v1 := h.Sum64()
	h.Write([]byte("salt"))
	v2 := h.Sum64()
	h.Write([]byte("salt2"))
	v3 := h.Sum64()
	h.Write([]byte("salt3"))
	v4 := h.Sum64()
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(
		"%016x%016x%016x%016x%016x%016x%016x%016x",
		v1, v2, v3, v4, v1^v2, v2^v3, v3^v4, v4^v1,
	)))
}
