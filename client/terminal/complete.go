package terminal

import "strings"

// knownCommands is the list of commands offered for tab completion.
var knownCommands = []string{
	"whoami", "hostname", "ipconfig", "pwd", "ls", "dir", "cd", "cat", "type",
	"cls", "clear", "exit",
	"Get-DomainUser", "Get-DomainGroup", "Get-DomainComputer",
	"Get-DomainController", "Get-DomainTrust", "Get-DomainGroupMember",
	"Get-DomainObjectAcl", "Find-InterestingDomainAcl",
	"Set-DomainObject", "Add-DomainObjectAcl",
	"Rubeus.exe", "mimikatz.exe", "SharpHound.exe", "PowerView.ps1",
	"crackmapexec", "secretsdump.py",
}

// tabComplete attempts to complete the current input prefix.
// Returns the completed string, or "" if no unique match.
func tabComplete(prefix string) string {
	if prefix == "" {
		return ""
	}

	// Split into command part and current word being typed.
	parts := strings.Fields(prefix)
	if len(parts) == 0 {
		return ""
	}

	// If only one token and no trailing space — complete the command.
	if len(parts) == 1 && !strings.HasSuffix(prefix, " ") {
		word := parts[0]
		var matches []string
		for _, cmd := range knownCommands {
			if strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(word)) {
				matches = append(matches, cmd)
			}
		}
		if len(matches) == 1 {
			return matches[0]
		}
		return ""
	}

	// Otherwise complete flags for known tools.
	cmd := strings.ToLower(parts[0])
	current := ""
	if !strings.HasSuffix(prefix, " ") && len(parts) > 1 {
		current = parts[len(parts)-1]
	}
	if current == "" {
		return ""
	}

	flags := flagsFor(cmd)
	var matches []string
	for _, f := range flags {
		if strings.HasPrefix(strings.ToLower(f), strings.ToLower(current)) {
			matches = append(matches, f)
		}
	}
	if len(matches) == 1 {
		base := prefix[:strings.LastIndex(prefix, current)]
		return base + matches[0]
	}
	return ""
}

func flagsFor(cmd string) []string {
	switch cmd {
	case "get-domainuser":
		return []string{"-Identity", "-SPN", "-AdminCount", "-Domain"}
	case "get-domaingroup":
		return []string{"-Identity", "-Domain"}
	case "get-domainobjectacl", "find-interestingdomainacl":
		return []string{"-Identity", "-ResolveGUIDs", "-Domain"}
	case "set-domainobject":
		return []string{"-Identity", "-Set", "-Domain"}
	case "add-domainobjectacl":
		return []string{"-TargetIdentity", "-PrincipalIdentity", "-Rights", "-Domain"}
	case "rubeus.exe":
		return []string{"kerberoast", "asktgt", "s4u", "dump", "triage", "ptt",
			"/user:", "/password:", "/ntlm:", "/nowrap", "/ticket:"}
	case "mimikatz.exe":
		return []string{"sekurlsa::logonpasswords", "lsadump::dcsync",
			"lsadump::sam", "token::elevate", "kerberos::ptt"}
	case "crackmapexec":
		return []string{"smb", "winrm", "-u", "-p", "-H", "--shares", "--sam"}
	case "cd", "type", "cat":
		return []string{`C:\`, `C:\tools`, `C:\Users\jsmith\Desktop`}
	default:
		return nil
	}
}
