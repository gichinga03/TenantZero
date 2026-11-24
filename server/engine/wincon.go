package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/matcher"
	"github.com/0x4D67/tenantzero/shared/schema"
)

// Evaluator checks win conditions after each command dispatch.
type Evaluator struct {
	lab   *schema.Lab
	state *schema.SessionState
	p     providers.Provider
}

// NewEvaluator constructs an Evaluator for the given lab and session state.
func NewEvaluator(lab *schema.Lab, state *schema.SessionState, p providers.Provider) *Evaluator {
	return &Evaluator{lab: lab, state: state, p: p}
}

// Check evaluates all WinConditions against current state and returns PushEvents
// for any newly satisfied objectives. Already-completed objectives (tracked in
// state.CompletedObjectives) are skipped. If all objectives are now complete,
// a lab_complete PushEvent is appended as the final entry.
func (e *Evaluator) Check(lastCmd schema.ParsedCommand, lastResult schema.ToolResult) []schema.PushEvent {
	var events []schema.PushEvent

	completed := make(map[string]bool, len(e.state.CompletedObjectives))
	for _, obj := range e.state.CompletedObjectives {
		completed[obj.Label] = true
	}

	m := matcher.NewMatcher(nil, 0) // command_executed: exact+wildcard only

	for _, wc := range e.lab.WinConditions {
		if completed[wc.Label] {
			continue
		}
		if e.evaluate(wc, lastCmd, lastResult, m) {
			e.state.CompletedObjectives = append(e.state.CompletedObjectives, schema.CompletedObjective{
				Label:       wc.Label,
				CompletedAt: time.Now().UTC(),
			})
			events = append(events, schema.PushEvent{
				Type:    "objective_complete",
				Payload: fmt.Sprintf("[*] Objective complete: %s", wc.Label),
			})
		}
	}

	if len(events) > 0 && e.allComplete() {
		events = append(events, schema.PushEvent{
			Type:    "lab_complete",
			Payload: e.labCompletePayload(),
		})
	}

	return events
}

// evaluate dispatches to the appropriate check for each WinCondition type.
func (e *Evaluator) evaluate(wc schema.WinCondition, cmd schema.ParsedCommand, result schema.ToolResult, m *matcher.Matcher) bool {
	switch wc.Type {
	case "command_executed":
		return e.checkCommandExecuted(wc, cmd, m)
	case "state_check":
		return e.checkState(wc)
	case "loot_check":
		return e.checkLoot(wc)
	case "file_read":
		return e.checkFileRead(wc, cmd)
	default:
		return false
	}
}

// checkCommandExecuted matches lastCmd.Binary + lastCmd.Subcommand against
// WinCondition.Match using the shared matcher (exact → wildcard → fuzzy).
func (e *Evaluator) checkCommandExecuted(wc schema.WinCondition, cmd schema.ParsedCommand, m *matcher.Matcher) bool {
	parts := []string{cmd.Binary}
	if cmd.Subcommand != "" {
		parts = append(parts, cmd.Subcommand)
	}
	parts = append(parts, cmd.Args...)
	subject := strings.TrimSpace(strings.Join(parts, " "))
	m.AddPattern(wc.Match)
	if wc.Match == "" {
		m.AddPattern(wc.Value)
	}
	result := m.Match(subject)
	return result.Level != matcher.MatchNone
}

// checkState resolves WinCondition.Key as a dot-path into SessionState and
// compares the result to WinCondition.Value (case-insensitive).
func (e *Evaluator) checkState(wc schema.WinCondition) bool {
	val, ok := resolveStatePath(e.state, wc.Key)
	if !ok {
		return false
	}
	return strings.EqualFold(val, wc.Value)
}

// checkLoot checks whether a loot map path (ntlm_hashes.<user>, etc.) exists
// and is non-empty.
func (e *Evaluator) checkLoot(wc schema.WinCondition) bool {
	parts := strings.SplitN(wc.Key, ".", 2)
	if len(parts) != 2 {
		return false
	}
	category, key := parts[0], parts[1]
	switch category {
	case "ntlm_hashes":
		v, ok := e.state.Loot.NTLMHashes[key]
		return ok && v != ""
	case "spn_hashes":
		v, ok := e.state.Loot.SPNHashes[key]
		return ok && v != ""
	case "pfx_certs":
		v, ok := e.state.Loot.PFXCerts[key]
		return ok && v != ""
	case "plaintext_creds":
		v, ok := e.state.Loot.PlaintextCreds[key]
		return ok && v != ""
	default:
		return false
	}
}

// checkFileRead is satisfied when lastCmd was a cat/type command whose first
// argument matches WinCondition.Match (case-insensitive path comparison).
func (e *Evaluator) checkFileRead(wc schema.WinCondition, cmd schema.ParsedCommand) bool {
	if cmd.Binary != "cat" && cmd.Binary != "type" {
		return false
	}
	if len(cmd.Args) == 0 {
		return false
	}
	target := normalizePath(cmd.Args[0])
	want := normalizePath(wc.Match)
	return strings.EqualFold(target, want)
}

// normalizePath converts forward slashes to backslashes and lowercases for
// case-insensitive Windows-style path comparison.
func normalizePath(p string) string {
	return strings.ToLower(strings.ReplaceAll(p, "/", `\`))
}

// resolveStatePath resolves a dot-path into SessionState.
//
// Supported paths:
//
//	user                          → state.User
//	hostname                      → state.Hostname
//	cwd                           → state.CWD
//	loot.ntlm_hashes.<user>       → state.Loot.NTLMHashes[user]
//	loot.spn_hashes.<user>        → state.Loot.SPNHashes[user]
//	loot.pfx_certs.<user>         → state.Loot.PFXCerts[user]
//	loot.plaintext_creds.<user>   → state.Loot.PlaintextCreds[user]
func resolveStatePath(state *schema.SessionState, path string) (string, bool) {
	switch path {
	case "user":
		return state.User, state.User != ""
	case "hostname":
		return state.Hostname, state.Hostname != ""
	case "cwd":
		return state.CWD, state.CWD != ""
	}

	parts := strings.SplitN(path, ".", 3)
	if len(parts) == 3 && parts[0] == "loot" {
		category, key := parts[1], parts[2]
		switch category {
		case "ntlm_hashes":
			v, ok := state.Loot.NTLMHashes[key]
			return v, ok && v != ""
		case "spn_hashes":
			v, ok := state.Loot.SPNHashes[key]
			return v, ok && v != ""
		case "pfx_certs":
			v, ok := state.Loot.PFXCerts[key]
			return v, ok && v != ""
		case "plaintext_creds":
			v, ok := state.Loot.PlaintextCreds[key]
			return v, ok && v != ""
		}
	}

	return "", false
}

// allComplete returns true when every lab WinCondition has been satisfied.
func (e *Evaluator) allComplete() bool {
	completed := make(map[string]bool, len(e.state.CompletedObjectives))
	for _, obj := range e.state.CompletedObjectives {
		completed[obj.Label] = true
	}
	for _, wc := range e.lab.WinConditions {
		if !completed[wc.Label] {
			return false
		}
	}
	return true
}

// labCompletePayload builds the lab_complete push event payload including a
// summary of all completed objectives with their timestamps.
func (e *Evaluator) labCompletePayload() string {
	var sb strings.Builder
	sb.WriteString("[+] Lab complete! All objectives achieved.\n\n")
	for _, obj := range e.state.CompletedObjectives {
		sb.WriteString(fmt.Sprintf("  [*] %-40s  %s\n",
			obj.Label,
			obj.CompletedAt.Format("2006-01-02 15:04:05 UTC"),
		))
	}
	return sb.String()
}