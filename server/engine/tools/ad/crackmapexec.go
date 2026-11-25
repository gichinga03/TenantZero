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
	if cmd.Subcommand == "" || len(cmd.Args) == 0 {
		return schema.ToolResult{Output: cmeUsage()}
	}

	protocol := strings.ToLower(cmd.Subcommand)
	target := cmd.Args[0]
	flags := cmd.Flags

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
				Effects: []schema.SideEffect{
					{Type: schema.SideEffectTypePivotContext, PivotHost: hostname, PivotIdentity: user},
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
		{Type: schema.SideEffectTypePivotContext, PivotHost: hostname, PivotIdentity: user},
		{Type: schema.SideEffectTypeSetSessionUser, NewUser: user},
	}

	return schema.ToolResult{
		Output: fmt.Sprintf("WINRM       %-15s 5985   %-15s     [+] %s\\%s:%s (Pwn3d!)\n",
			ip, hostname, domain, user, pass),
		Effects: effects,
	}
}

// resolveTarget looks up a host by hostname or IP.
func resolveTarget(target string, provider providers.Provider) (schema.Host, bool) {
	h, ok := provider.GetHost(target)
	if ok {
		return h, true
	}
	// Try IP lookup.
	for _, host := range provider.GetAllHosts() {
		if host.IP == target {
			return host, true
		}
	}
	return schema.Host{}, false
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



func cmeUsage() string {
	return `usage: crackmapexec [-h] [-t THREADS] [--timeout TIMEOUT] [--jitter INTERVAL]
                    {smb,winrm} ...

positional arguments:
  {smb,winrm}
    smb          own stuff using SMB
    winrm        own stuff using WinRM
`
}