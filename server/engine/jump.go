package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// jumpMethods maps method name → required right on the target host
var jumpMethods = map[string]string{
	"psexec": "local_admin",
	"winrm":  "winrm_access",
	"wmi":    "local_admin",
}

// handleJump parses and executes:  jump <method> <host> [listener]
// Returns (output, sideEffects, error)
func (s *Session) handleJump(args []string) (string, []schema.SideEffect, error) {
	if len(args) < 2 {
		return "", nil, fmt.Errorf("usage: jump <psexec|winrm|wmi> <host> [listener]")
	}

	method := strings.ToLower(args[0])
	targetHost := args[1]
	// listener is optional in the sim — default to "default"
	listener := "default"
	if len(args) >= 3 {
		listener = args[2]
	}

	// 1. Validate method
	requiredRight, ok := jumpMethods[method]
	if !ok {
		return "", nil, fmt.Errorf("unknown jump method %q — valid: psexec, winrm, wmi", method)
	}

	// 2. Validate target host exists in the lab
	host, found := s.findHost(targetHost)
	if !found {
		return "", nil, fmt.Errorf("host %q not found in lab scope", targetHost)
	}

	// 3. Validate can_pivot_to relationship from current host
	if !s.canPivotTo(s.State.ActiveHost, targetHost) {
		return "", nil, fmt.Errorf(
			"no pivot path from %s to %s — enumerate reachable hosts first",
			s.State.ActiveHost, targetHost,
		)
	}

	// 4. Check runtime ACL or static lab right for any effective caller
	// (active user + impersonated tokens/tickets acquired this session).
	callers := s.effectiveCallers()
	hasRight := false
	for _, caller := range callers {
		if s.HasRuntimeACL(caller, targetHost, requiredRight) ||
			s.labGrantsRight(caller, targetHost, requiredRight) {
			hasRight = true
			break
		}
	}

	if !hasRight {
		return "", nil, fmt.Errorf(
			"access denied — %s does not have %s on %s",
			s.State.ActiveUser, requiredRight, targetHost,
		)
	}

	// 5. Build side effects: spawn beacon + pivot context
	newBeaconParentID := s.State.ActiveBeaconID
	effects := []schema.SideEffect{
		{
			Type:           schema.SideEffectTypeSpawnBeacon,
			BeaconHost:     host.Hostname,
			BeaconIdentity: s.State.ActiveUser,
			BeaconParentID: newBeaconParentID,
		},
		{
			Type:         schema.SideEffectTypePivotContext,
			PivotHost:    host.Hostname,
			PivotIdentity: s.State.ActiveUser,
		},
	}

	// 6. Build realistic CS-style output
	pid := pseudoPID(host.Hostname)
	output := buildJumpOutput(method, host.Hostname, s.State.ActiveUser, listener, pid)

	return output, effects, nil
}

// findHost looks up a host by hostname (case-insensitive) in the lab.
func (s *Session) findHost(name string) (schema.Host, bool) {
	for _, h := range s.Lab.Hosts {
		if strings.EqualFold(h.Hostname, name) {
			return h, true
		}
	}
	return schema.Host{}, false
}

// canPivotTo checks whether a can_pivot_to relationship exists in the lab YAML.
func (s *Session) canPivotTo(from, to string) bool {
	for _, rel := range s.Lab.Relationships {
		if rel.Type == "can_pivot_to" &&
			strings.EqualFold(rel.From, from) &&
			strings.EqualFold(rel.To, to) {
			return true
		}
	}
	return false
}

// labGrantsRight checks static rights declared on a host in the lab YAML.
// Expects host.Rights to be a map[identity][]string — adapt to your schema shape.
func (s *Session) labGrantsRight(principal, targetHost, right string) bool {
	host, found := s.findHost(targetHost)
	if !found {
		return false
	}
	// Normalise principal to bare username for comparison so
	// "awalker" matches "MERIDIAN\awalker" and vice versa.
	barePrincipal := principal
	if idx := strings.LastIndexAny(principal, `\/`); idx >= 0 {
		barePrincipal = principal[idx+1:]
	}
	for _, r := range host.Rights {
		rBare := r.Principal
		if idx := strings.LastIndexAny(r.Principal, `\/`); idx >= 0 {
			rBare = r.Principal[idx+1:]
		}
		if strings.EqualFold(r.Principal, principal) || strings.EqualFold(rBare, barePrincipal) {
			for _, rr := range r.Rights {
				if strings.EqualFold(rr, right) {
					return true
				}
			}
		}
	}
	return false
}

// pseudoPID generates a stable-looking PID from the hostname so output looks consistent.
func pseudoPID(hostname string) int {
	sum := 0
	for _, c := range hostname {
		sum += int(c)
	}
	return 1000 + (sum % 64000)
}

func buildJumpOutput(method, host, user, listener string, pid int) string {
	ts := time.Now().Format("01/02 15:04:05")
	switch method {
	case "psexec":
		return fmt.Sprintf(
			`[*] Tasked beacon to run windows/beacon_bind_pipe (%s)
[+] host called home, sent: 220444 bytes
[+] established link to child beacon: %s
[*] Started service 7f3a2c1d on %s (pid: %d)
[+] Beacon session opened at %s — %s (%s)`,
			listener, host, host, pid, ts, host, user,
		)
	case "winrm":
		return fmt.Sprintf(
			`[*] Tasked beacon to run windows/beacon_bind_pipe (%s) via WinRM
[+] Connected to %s:5985
[+] Beacon session opened at %s — %s (%s) [pid: %d]`,
			listener, host, ts, host, user, pid,
		)
	case "wmi":
		return fmt.Sprintf(
			`[*] Tasked beacon to spawn %s via WMI
[*] Win32_Process::Create on %s
[+] Beacon session opened at %s — %s (%s) [pid: %d]`,
			listener, host, ts, host, user, pid,
		)
	default:
		return fmt.Sprintf("[+] Beacon on %s (%s)", host, user)
	}
}
// DebugJump prints internal jump state — remove before release.
func (s *Session) DebugJump(method, target string) string {
	callers := s.effectiveCallers()
	requiredRight := jumpMethods[strings.ToLower(method)]
	canPivot := s.canPivotTo(s.State.ActiveHost, target)
	host, found := s.findHost(target)
	var rights []schema.HostRight
	if found {
		rights = host.Rights
	}
	var aclResults []string
	for _, c := range callers {
		lab := s.labGrantsRight(c, target, requiredRight)
		acl := s.HasRuntimeACL(c, target, requiredRight)
		aclResults = append(aclResults, fmt.Sprintf("  caller=%q lab=%v acl=%v", c, lab, acl))
	}
	return fmt.Sprintf(
		"ActiveHost=%q ActiveUser=%q\ncanPivot=%v hostFound=%v\nHostRights=%+v\nCallers+ACL:\n%s",
		s.State.ActiveHost, s.State.ActiveUser,
		canPivot, found, rights,
		strings.Join(aclResults, "\n"),
	)
}
