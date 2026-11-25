package engine

import (
	"fmt"
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// Parse tokenises raw into a ParsedCommand. It handles five input dialects:
//
//  1. Standard tools:  "rubeus.exe kerberoast /user:svc_sql /nowrap"
//  2. PowerView:       "Get-DomainUser -Identity svc_sql"  (no binary wrapper)
//  3. Mimikatz:        "mimikatz # sekurlsa::logonpasswords"
//  4. Impacket:        "secretsdump.py CORP/admin:pass@dc01"
//  5. Shell builtins:  "cd C:\Users\Public"
func Parse(raw string) schema.ParsedCommand {
	raw = strings.TrimSpace(raw)
	cmd := schema.ParsedCommand{Raw: raw}
	if raw == "" {
		return cmd
	}

	// --- mimikatz: "mimikatz # module::command [args]" ---
	if isMimikatz(raw) {
		return parseMimikatz(raw)
	}

	tokens := tokenise(raw)
	if len(tokens) == 0 {
		return cmd
	}

	first := tokens[0]

	// --- PowerView: binary is a known Get-*/Set-*/Add-* verb ---
	if isPowerViewVerb(first) {
		cmd.Executable = normaliseBinary(first)
		return parseFlags(cmd, tokens[1:])
	}

	// --- everything else: first token is the binary ---
	cmd.Executable = normaliseBinary(first)
	rest := tokens[1:]

	// Second token is subcommand if it doesn't start with - or /
	if len(rest) > 0 && !isFlagToken(rest[0]) {
		cmd.Subcommand = strings.ToLower(rest[0])
		rest = rest[1:]
	}

	return parseFlags(cmd, rest)
}

// isMimikatz returns true for the "mimikatz # …" shell-within-a-shell syntax.
func isMimikatz(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.HasPrefix(lower, "mimikatz") && strings.Contains(raw, "#")
}

// parseMimikatz handles: "mimikatz # module::command [/flag:val …]"
// Binary → "mimikatz", Subcommand → "module::command" (lowercased).
func parseMimikatz(raw string) schema.ParsedCommand {
	cmd := schema.ParsedCommand{Raw: raw}
	cmd.Executable = "mimikatz"

	hashIdx := strings.Index(raw, "#")
	if hashIdx < 0 {
		return cmd
	}

	rest := strings.TrimSpace(raw[hashIdx+1:])
	tokens := tokenise(rest)
	if len(tokens) == 0 {
		return cmd
	}

	cmd.Subcommand = strings.ToLower(tokens[0])
	return parseFlags(cmd, tokens[1:])
}

// normaliseBinary lowercases and strips the .exe / .py suffix used for
// display but not for registry lookup.
func normaliseBinary(tok string) string {
	b := strings.ToLower(tok)
	b = strings.TrimSuffix(b, ".exe")
	b = strings.TrimSuffix(b, ".py")
	return b
}

// isPowerViewVerb returns true for the PowerShell Active Directory cmdlets
// that are invoked directly without a wrapper binary.
func isPowerViewVerb(tok string) bool {
	lower := strings.ToLower(tok)
	prefixes := []string{
		"get-domain", "get-object", "get-net",
		"set-domain", "add-domain", "find-domain",
		"invoke-acl", "find-interesting",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// isFlagToken returns true if the token starts with - or /.
func isFlagToken(tok string) bool {
	return strings.HasPrefix(tok, "-") || strings.HasPrefix(tok, "/")
}

// parseFlags extracts flags and positional args from the remaining tokens into
// cmd. Mutates and returns cmd.
//
// Supported syntaxes:
//
//	/Flag:value      — slash, colon-separated (Rubeus/Certify style)
//	/Flag value      — slash, space-separated
//	--flag value     — GNU long, space-separated
//	--flag=value     — GNU long, equals-separated
//	-Flag value      — single-dash (PowerView/CME style)
//	-Flag=value      — single-dash, equals-separated
func parseFlags(cmd schema.ParsedCommand, tokens []string) schema.ParsedCommand {
	if cmd.Flags == nil {
		cmd.Flags = make(map[string]string)
	}

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		if strings.HasPrefix(tok, "--") {
			// GNU long: --flag=value or --flag value
			inner := tok[2:]
			if eq := strings.IndexByte(inner, '='); eq >= 0 {
				cmd.Flags[strings.ToLower(inner[:eq])] = inner[eq+1:]
				i++
				continue
			}
			key := strings.ToLower(inner)
			i++
			if i < len(tokens) && !isFlagToken(tokens[i]) {
				cmd.Flags[key] = tokens[i]
				i++
			} else {
				cmd.Flags[key] = "true"
			}
			continue
		}

		if strings.HasPrefix(tok, "/") {
			// Slash flag: /Flag:value or /Flag value or bare /Flag
			inner := tok[1:]
			if col := strings.IndexByte(inner, ':'); col >= 0 {
				cmd.Flags[strings.ToLower(inner[:col])] = inner[col+1:]
				i++
				continue
			}
			key := strings.ToLower(inner)
			i++
			if i < len(tokens) && !isFlagToken(tokens[i]) {
				cmd.Flags[key] = tokens[i]
				i++
			} else {
				cmd.Flags[key] = "true"
			}
			continue
		}

		if strings.HasPrefix(tok, "-") {
			// Single-dash: -Flag=value or -Flag value or bare -Flag
			inner := tok[1:]
			if eq := strings.IndexByte(inner, '='); eq >= 0 {
				cmd.Flags[strings.ToLower(inner[:eq])] = inner[eq+1:]
				i++
				continue
			}
			key := strings.ToLower(inner)
			i++
			if i < len(tokens) && !isFlagToken(tokens[i]) {
				cmd.Flags[key] = tokens[i]
				i++
			} else {
				cmd.Flags[key] = "true"
			}
			continue
		}

		// positional arg
		cmd.Args = append(cmd.Args, tok)
		i++
	}

	return cmd
}

// tokenise splits raw on whitespace, respecting double-quoted spans.
// "CORP\admin:pass@dc01" is one token; quoted strings lose their quotes.
func tokenise(raw string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false

	for _, r := range raw {
		switch {
		case r == '"':
			inQuote = !inQuote
		case (r == ' ' || r == '\t') && !inQuote:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// --- builtins ---

const (
	builtinCD    = "cd"
	builtinLS    = "ls"
	builtinDir   = "dir"
	builtinPwd   = "pwd"
	builtinCat   = "cat"
	builtinType  = "type"
	builtinCls   = "cls"
	builtinClear = "clear"
	builtinExit  = "exit"
	builtinHelp  = "help"
	builtinHint  = "hint"
)

var builtins = map[string]bool{
	builtinCD: true, builtinLS: true, builtinDir: true,
	builtinPwd: true, builtinCat: true, builtinType: true,
	builtinCls: true, builtinClear: true, builtinExit: true,
	builtinHelp: true, builtinHint: true,
	"whoami": true, "hostname": true, "ipconfig": true, "net": true,
}

// Dispatch routes a parsed command to either a shell builtin or the tool
// registry. Unknown binaries return a "not recognised" ToolResult.
func (s *Session) Dispatch(cmd schema.ParsedCommand) schema.ToolResult {
	if builtins[cmd.Executable] {
		return s.dispatchBuiltin(cmd)
	}

	return s.Registry.Dispatch(cmd, &s.State, s.Provider, s.Lab.Tools.Overrides)
}

// dispatchBuiltin handles all shell-level commands without touching the tool
// registry.
func (s *Session) dispatchBuiltin(cmd schema.ParsedCommand) schema.ToolResult {
	fs := NewFS(&s.Lab, &s.State)

	switch cmd.Executable {
	case builtinCD:
		path := ""
		if len(cmd.Args) > 0 {
			path = cmd.Args[0]
		} else if cmd.Subcommand != "" {
			path = cmd.Subcommand
		} else if p, ok := cmd.Flags["path"]; ok {
			path = p
		}
		resolved, err := fs.Cd(path)
		if err != nil {
			return schema.ToolResult{Output: err.Error()}
		}
		s.State.CWD = resolved
		return schema.ToolResult{Output: ""}

	case builtinLS, builtinDir:
		path := s.State.CWD
		if len(cmd.Args) > 0 {
			path = cmd.Args[0]
		} else if cmd.Subcommand != "" {
			path = cmd.Subcommand
		}
		return schema.ToolResult{Output: fs.Ls(path)}

	case builtinCat, builtinType:
		path := ""
		if len(cmd.Args) > 0 {
			path = cmd.Args[0]
		} else if cmd.Subcommand != "" {
			path = cmd.Subcommand
		}
		return schema.ToolResult{Output: fs.Cat(path)}

	case builtinPwd:
		return schema.ToolResult{Output: s.State.CWD}

	case builtinCls, builtinClear:
		return schema.ToolResult{Output: "\x0c"} // form-feed; client interprets as clear

	case builtinExit:
		return schema.ToolResult{
			Output: "Session ended by user.",
			Effects: []schema.SideEffect{
				{Type: "session_end"},
			},
		}

	case "whoami":
		domain := strings.ToUpper(strings.SplitN(s.Lab.Domain.Name, ".", 2)[0])
		user := s.State.ActiveUser
		if hasFlag(cmd.Flags, "all") || hasFlag(cmd.Flags, "a") {
			return schema.ToolResult{Output: whoamiAll(domain, user, s.Lab, &s.State)}
		}
		if hasFlag(cmd.Flags, "groups") {
			return schema.ToolResult{Output: whoamiGroups(domain, user, s.Lab)}
		}
		if hasFlag(cmd.Flags, "priv") || hasFlag(cmd.Flags, "privs") {
			return schema.ToolResult{Output: whoamiPrivs(user, s.Lab)}
		}
		return schema.ToolResult{Output: fmt.Sprintf(`%s\%s`, domain, user)}

	case "hostname":
		return schema.ToolResult{Output: s.State.ActiveHost}

	case "ipconfig":
		ip := ""
		for _, h := range s.Lab.Hosts {
			if strings.EqualFold(h.Hostname, s.State.ActiveHost) {
				ip = h.IP
			}
		}
		if ip == "" { ip = "10.10.10.10" }
		return schema.ToolResult{Output: fmt.Sprintf(
			"Windows IP Configuration\r\n\r\nEthernet adapter Ethernet0:\r\n\r\n   IPv4 Address. . . . . . . . . . . : %s\r\n   Subnet Mask . . . . . . . . . . . : 255.255.255.0\r\n   Default Gateway . . . . . . . . . : %s.1\r\n",
			ip, ip[:strings.LastIndex(ip, ".")],
		)}

	case "net":
		return schema.ToolResult{Output: "The syntax of this command is:\r\nNET [ ACCOUNTS | COMPUTER | CONFIG | CONTINUE | FILE | GROUP | HELP |\r\n      HELPMSG | LOCALGROUP | PAUSE | SESSION | SHARE | START |\r\n      STATISTICS | STOP | TIME | USE | USER | VIEW ]"}

	case builtinHelp:
		return schema.ToolResult{Output: s.helpText()}

	case builtinHint:
		return schema.ToolResult{Output: s.nextHint()}
	}

	return schema.ToolResult{Output: fmt.Sprintf("builtin '%s' not implemented", cmd.Executable)}
}

// helpText returns a formatted list of available tool binaries from the
// registry, plus the shell builtins.
func (s *Session) helpText() string {
	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	for _, name := range s.Registry.Names() {
		sb.WriteString("  ")
		sb.WriteString(name)
		sb.WriteString("\n")
	}
	sb.WriteString("\nShell builtins: cd, ls, dir, cat, type, pwd, cls, clear, hint, help, exit\n")
	return sb.String()
}

// nextHint returns the first unused hint whose MinElapsedMinutes threshold
// has been reached, or a no-hints message.
func (s *Session) nextHint() string {
	used := make(map[string]bool, len(s.State.UsedHints))
	for _, id := range s.State.UsedHints {
		used[id] = true
	}

	for i, h := range s.Lab.Hints {
		if used[h.ID] {
			continue
		}
		s.State.UsedHints = append(s.State.UsedHints, h.ID)
		return fmt.Sprintf("[Hint %d/%d] %s", i+1, len(s.Lab.Hints), h.Text)
	}
	return "No hints available yet. Keep trying."
}

// whoamiAll produces output matching "whoami /all" on a domain-joined Windows host.
func whoamiAll(domain, user string, lab schema.Lab, state *schema.SessionState) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\r\nUSER INFORMATION\r\n----------------\r\n\r\n"))
	sb.WriteString(fmt.Sprintf("User Name           SID\r\n"))
	sb.WriteString(fmt.Sprintf("=================== =============================================\r\n"))
	domainLower := strings.ToLower(domain)
	rid := fnvRIDCmd(user)
	sb.WriteString(fmt.Sprintf("%s\\%s S-1-5-21-%d-%d-%d-%d\r\n",
		domainLower, user, 3623811015, 3361044348, 30300820, rid))
	sb.WriteString(whoamiGroups(domain, user, lab))
	sb.WriteString(whoamiPrivs(user, lab))
	return sb.String()
}

// whoamiGroups produces output matching "whoami /groups".
func whoamiGroups(domain, user string, lab schema.Lab) string {
	var sb strings.Builder
	sb.WriteString("\r\nGROUP INFORMATION\r\n-----------------\r\n\r\n")
	sb.WriteString("Group Name                                 Type             SID                                            Attributes\r\n")
	sb.WriteString("========================================== ================ ============================================== ==================================================\r\n")

	domainUpper := strings.ToUpper(domain)
	groups := []struct{ name, typ, sid, attrs string }{
		{"Everyone",                              "Well-known group", "S-1-1-0",          "Mandatory group, Enabled by default, Enabled group"},
		{domainUpper + "\\Domain Users",          "Group",            "S-1-5-21-3623811015-3361044348-30300820-513", "Mandatory group, Enabled by default, Enabled group"},
		{"BUILTIN\\Users",                        "Alias",            "S-1-5-32-545",     "Mandatory group, Enabled by default, Enabled group"},
		{"NT AUTHORITY\\NETWORK",                 "Well-known group", "S-1-5-2",          "Mandatory group, Enabled by default, Enabled group"},
		{"NT AUTHORITY\\Authenticated Users",     "Well-known group", "S-1-5-11",         "Mandatory group, Enabled by default, Enabled group"},
		{"NT AUTHORITY\\This Organization",       "Well-known group", "S-1-5-15",         "Mandatory group, Enabled by default, Enabled group"},
		{"NT AUTHORITY\\NTLM Authentication",     "Well-known group", "S-1-5-64-10",      "Mandatory group, Enabled by default, Enabled group"},
		{"Mandatory Label\\Medium Mandatory Level","Label",           "S-1-16-8192",      "Mandatory group, Enabled by default, Enabled group"},
	}

	// Add local admin group if user has local admin
	_ = lab
	groups = append(groups, struct{ name, typ, sid, attrs string }{
		"BUILTIN\\Administrators", "Alias", "S-1-5-32-544",
		"Mandatory group, Enabled by default, Enabled group, Group owner",
	})

	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("%-42s %-16s %-46s %s\r\n", g.name, g.typ, g.sid, g.attrs))
	}
	return sb.String()
}

// whoamiPrivs produces output matching "whoami /priv".
func whoamiPrivs(user string, lab schema.Lab) string {
	_ = user
	_ = lab
	var sb strings.Builder
	sb.WriteString("\r\nPRIVILEGES INFORMATION\r\n----------------------\r\n\r\n")
	sb.WriteString("Privilege Name                            Description                                                        State\r\n")
	sb.WriteString("========================================= ================================================================== =======\r\n")

	privs := []struct{ name, desc, state string }{
		{"SeShutdownPrivilege",          "Shut down the system",                                                "Disabled"},
		{"SeChangeNotifyPrivilege",      "Bypass traverse checking",                                            "Enabled"},
		{"SeUndockPrivilege",            "Remove computer from docking station",                                "Disabled"},
		{"SeIncreaseWorkingSetPrivilege","Increase a process working set",                                      "Disabled"},
		{"SeTimeZonePrivilege",          "Change the time zone",                                                "Disabled"},
		{"SeDebugPrivilege",             "Debug programs",                                                      "Enabled"},
		{"SeImpersonatePrivilege",       "Impersonate a client after authentication",                           "Enabled"},
		{"SeCreateGlobalObjects",        "Create global objects",                                               "Enabled"},
	}
	for _, priv := range privs {
		sb.WriteString(fmt.Sprintf("%-41s %-66s %s\r\n", priv.name, priv.desc, priv.state))
	}
	return sb.String()
}

// fnvRIDCmd is a local copy of fnvRID for use outside the tools package.
func fnvRIDCmd(s string) uint32 {
	var h uint32 = 2166136261
	for _, b := range []byte(s) {
		h ^= uint32(b)
		h *= 16777619
	}
	return 1000 + (h % 9000)
}

// hasFlag checks whether a flag key exists in cmd.Flags (case-insensitive).
func hasFlag(flags map[string]string, key string) bool {
	_, ok := flags[strings.ToLower(key)]
	return ok
}

