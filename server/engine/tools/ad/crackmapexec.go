// Package ad implements Active Directory attack tool simulations for the TenantZero engine.
package ad

import (
	"fmt"
	"strings"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/schema"
)

// CrackMapExec simulates the CrackMapExec (CME) lateral movement tool.
// Registered under both "crackmapexec" and "cme".
type CrackMapExec struct{}

// Name returns the primary tool identifier.
func (c *CrackMapExec) Name() string { return "crackmapexec" }
func (c *CrackMapExec) Aliases() []string { return []string{"cme", "crackmapexec.py"} }

// Requires returns the minimum access conditions.
// Individual subcommands enforce further requirements.
func (c *CrackMapExec) Requires() []string { return nil }

// Execute dispatches a CrackMapExec invocation.
// Format: crackmapexec <protocol> <target> [flags]
func (c *CrackMapExec) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(cmd.Args) < 2 {
		return schema.ToolResult{Output: cmeUsage()}
	}

	protocol := strings.ToLower(cmd.Args[0])
	target := cmd.Args[1]
	flags := parseCMEFlags(cmd.Args[2:])

	switch protocol {
	case "smb":
		return c.smb(target, flags, state, provider)
	case "winrm":
		return c.winrm(target, flags, state, provider)
	default:
		return schema.ToolResult{Output: fmt.Sprintf("[-] Protocol '%s' not supported\n", protocol)}
	}
}

// smb handles SMB-based enumeration and lateral movement.
func (c *CrackMapExec) smb(
	target string,
	flags map[string]string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	computer, ok := resolveTarget(target, provider)
	if !ok {
		return schema.ToolResult{
			Output: fmt.Sprintf("SMB         %-15s 445    %-15s     [-] Connection timeout\n", target, target),
		}
	}

	domain := providers.NetBIOSName(providers.GetDomain(provider))
	ip := computer.IP
	hostname := computer.Hostname

	user := flags["u"]
	pass := flags["p"]
	hash := flags["H"]

	// Base banner — always printed regardless of auth.
	baseLine := fmt.Sprintf("SMB         %-15s 445    %-15s     [*] Windows Server 2019 Build 17763 x64 (name:%s) (domain:%s) (signing:True) (SMBv1:False)\n",
		ip, hostname, hostname, domain)

	// No credentials supplied — banner only.
	if user == "" {
		if _, hasShares := flags["--shares"]; hasShares {
			return schema.ToolResult{Output: baseLine + "[-] Credentials required for share enumeration\n"}
		}
		return schema.ToolResult{Output: baseLine}
	}

	// Validate credentials.
	authed := false
	isAdmin := false
	if hash != "" {
		authed = provider.ValidateNTLM(user, hash)
	} else {
		authed = provider.ValidateCredentials(user, pass)
	}
	if authed {
		isAdmin = provider.HasLocalAdmin(user, hostname)
	}

	credential := func() string {
		if hash != "" {
			return fmt.Sprintf("%s\\%s:%s", domain, user, hash)
		}
		return fmt.Sprintf("%s\\%s:%s", domain, user, pass)
	}

	if !authed {
		return schema.ToolResult{
			Output: baseLine + fmt.Sprintf("SMB         %-15s 445    %-15s     [-] %s STATUS_LOGON_FAILURE\n",
				ip, hostname, credential()),
		}
	}

	var sb strings.Builder
	sb.WriteString(baseLine)

	adminTag := ""
	if isAdmin {
		adminTag = " (Pwn3d!)"
	}
	fmt.Fprintf(&sb, "SMB         %-15s 445    %-15s     [+] %s%s\n",
		ip, hostname, credential(), adminTag)

	// --shares
	if _, hasShares := flags["--shares"]; hasShares {
		sb.WriteString(cmeShareTable(hostname, state))
		return schema.ToolResult{Output: sb.String()}
	}

	// -x command execution
	execCmd, hasExec := flags["x"]
	if !hasExec {
		execCmd = flags["-exec-method"]
	}
	if hasExec || execCmd != "" {
		if !isAdmin {
			fmt.Fprintf(&sb, "SMB         %-15s 445    %-15s     [-] %s\\%s Executed command via smbexec Error: access denied\n",
				ip, hostname, domain, user)
			return schema.ToolResult{Output: sb.String()}
		}
		fmt.Fprintf(&sb, "SMB         %-15s 445    %-15s     [+] %s\\%s Executed command via smbexec\n",
			ip, hostname, domain, user)
		// Only pivot session on a shell-spawn-style command.
		if isShellSpawn(execCmd) {
			return schema.ToolResult{
				Output: sb.String(),
				SideEffects: []schema.SideEffect{
					{Type: schema.SideEffectSetSessionMachine, Value: hostname},
				},
			}
		}
		return schema.ToolResult{Output: sb.String()}
	}

	return schema.ToolResult{Output: sb.String()}
}

// winrm handles WinRM-based lateral movement.
func (c *CrackMapExec) winrm(
	target string,
	flags map[string]string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	computer, ok := resolveTarget(target, provider)
	if !ok {
		return schema.ToolResult{
			Output: fmt.Sprintf("WINRM       %-15s 5985   %-15s     [-] Connection timeout\n", target, target),
		}
	}

	domain := providers.NetBIOSName(providers.GetDomain(provider))
	ip := computer.IP
	hostname := computer.Hostname
	user := flags["u"]
	pass := flags["p"]

	if user == "" || pass == "" {
		return schema.ToolResult{
			Output: fmt.Sprintf("WINRM       %-15s 5985   %-15s     [-] Credentials required\n", ip, hostname),
		}
	}

	if !provider.ValidateCredentials(user, pass) {
		return schema.ToolResult{
			Output: fmt.Sprintf("WINRM       %-15s 5985   %-15s     [-] %s\\%s:%s STATUS_LOGON_FAILURE\n",
				ip, hostname, domain, user, pass),
		}
	}

	effects := []schema.SideEffect{
		{Type: schema.SideEffectSetSessionMachine, Value: hostname},
		{Type: schema.SideEffectSetSessionUser, Value: user},
	}

	return schema.ToolResult{
		Output: fmt.Sprintf("WINRM       %-15s 5985   %-15s     [+] %s\\%s:%s (Pwn3d!)\n",
			ip, hostname, domain, user, pass),
		SideEffects: effects,
	}
}

// resolveTarget looks up a computer by hostname or IP.
func resolveTarget(target string, provider providers.Provider) (schema.Computer, bool) {
	c, ok := provider.GetComputer(target)
	if ok {
		return c, true
	}
	// Try IP lookup.
	for _, comp := range providers.GetAllComputers(provider) {
		if comp.IP == target {
			return comp, true
		}
	}
	return schema.Computer{}, false
}

// cmeShareTable formats a share listing in CME style.
func cmeShareTable(hostname string, state *schema.SessionState) string {
	type shareEntry struct{ name, perms, comment string }

	shares := []shareEntry{
		{"ADMIN$", "READ,WRITE", "Remote Admin"},
		{"C$", "READ,WRITE", "Default share"},
		{"IPC$", "READ", "Remote IPC"},
		{"NETLOGON", "READ", "Logon server share"},
		{"SYSVOL", "READ", "Logon server share"},
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "SMB         %-15s 445    %-15s     [+] Enumerated shares\n", hostname, hostname)
	sb.WriteString("SMB         " + strings.Repeat(" ", 16) + "445    " +
		strings.Repeat(" ", 16) + "     Share           Permissions     Remark\n")
	sb.WriteString("SMB         " + strings.Repeat(" ", 16) + "445    " +
		strings.Repeat(" ", 16) + "     -----           -----------     ------\n")

	for _, s := range shares {
		fmt.Fprintf(&sb, "SMB         %-15s 445    %-15s     %-15s %-15s %s\n",
			hostname, hostname, s.name, s.perms, s.comment)
	}
	return sb.String()
}

// isShellSpawn returns true for commands that imply an interactive session.
func isShellSpawn(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	shellCmds := []string{"cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh"}
	for _, sc := range shellCmds {
		if lower == sc || strings.HasPrefix(lower, sc+" ") {
			return true
		}
	}
	return false
}

// parseCMEFlags extracts -flag value and --flag pairs from args.
func parseCMEFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			key := strings.TrimPrefix(a, "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[a] = "true" // boolean flag
			}
		} else if strings.HasPrefix(a, "-") {
			key := strings.TrimPrefix(a, "-")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
		}
	}
	return flags
}

func cmeUsage() string {
	return `usage: crackmapexec [-h] [-t THREADS] [--timeout TIMEOUT] [--jitter INTERVAL]
                    {smb,winrm} ...

positional arguments:
  {smb,winrm}
    smb          own stuff using SMB
    winrm        own stuff using WinRM
`
}