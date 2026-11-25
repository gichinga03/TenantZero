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
		completed[obj.ID] = true
	}

	m := matcher.NewMatcher(nil, 0) // command_executed: exact+wildcard only

	for _, wc := range e.lab.WinConditions {
		if completed[wc.ID] {
			continue
		}
		if e.evaluate(wc, lastCmd, lastResult, m) {
			e.state.CompletedObjectives = append(e.state.CompletedObjectives, schema.CompletedObjective{
				ID:          wc.ID,
				CompletedAt: time.Now().UTC(),
			})
			events = append(events, schema.PushEvent{
				Type:    "objective_complete",
				Payload: fmt.Sprintf("[*] Objective complete: %s", wc.ID),
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
	parts := []string{cmd.Executable}
	if cmd.Subcommand != "" {
		parts = append(parts, cmd.Subcommand)
	}
	parts = append(parts, cmd.Args...)
	subject := strings.TrimSpace(strings.Join(parts, " "))
	m.AddPattern(wc.Params["match"])
	if wc.Params["match"] == "" {
		m.AddPattern(wc.Params["value"])
	}
	result := m.Match(subject)
	return result.Level != matcher.MatchNone
}

// checkState resolves WinCondition.Key as a dot-path into SessionState and
// compares the result to WinCondition.Value (case-insensitive).
func (e *Evaluator) checkState(wc schema.WinCondition) bool {
	val, ok := resolveStatePath(e.state, wc.Params["key"])
	if !ok {
		return false
	}
	return strings.EqualFold(val, wc.Params["value"])
}

// checkLoot checks whether the loot slice contains an entry matching
// WinCondition.Params (loot_type and loot_value_contains).
func (e *Evaluator) checkLoot(wc schema.WinCondition) bool {
	wantType := wc.Params["loot_type"]
	wantContains := wc.Params["loot_value_contains"]
	for _, l := range e.state.Loot {
		if wantType != "" && !strings.EqualFold(l.Type, wantType) {
			continue
		}
		if wantContains != "" && !strings.Contains(strings.ToLower(l.Value+l.Source), strings.ToLower(wantContains)) {
			continue
		}
		return true
	}
	return false
}

// checkFileRead is satisfied when lastCmd was a cat/type command whose first
// argument matches WinCondition.Match (case-insensitive path comparison).
func (e *Evaluator) checkFileRead(wc schema.WinCondition, cmd schema.ParsedCommand) bool {
	if cmd.Executable != "cat" && cmd.Executable != "type" {
		return false
	}
	if len(cmd.Args) == 0 {
		return false
	}
	target := normalizePath(cmd.Args[0])
	want := normalizePath(wc.Params["match"])
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
	case "user", "active_user":
		return state.ActiveUser, state.ActiveUser != ""
	case "hostname", "active_host":
		return state.ActiveHost, state.ActiveHost != ""
	case "cwd":
		return state.CWD, state.CWD != ""
	}

	parts := strings.SplitN(path, ".", 3)
	if len(parts) == 3 && parts[0] == "loot" {
		category, key := parts[1], parts[2]
		for _, l := range state.Loot {
			if strings.EqualFold(l.Type, category) && strings.EqualFold(l.Source, key) {
				return l.Value, l.Value != ""
			}
		}
	}

	return "", false
}

// allComplete returns true when every lab WinCondition has been satisfied.
func (e *Evaluator) allComplete() bool {
	completed := make(map[string]bool, len(e.state.CompletedObjectives))
	for _, obj := range e.state.CompletedObjectives {
		completed[obj.ID] = true
	}
	for _, wc := range e.lab.WinConditions {
		if !completed[wc.ID] {
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
			obj.ID,
			obj.CompletedAt.Format("2006-01-02 15:04:05 UTC"),
		))
	}
	return sb.String()
}