package tools

import (
	"fmt"
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
	"github.com/0x4D67/tenantzero/server/engine/providers"
)

// Registry maps every binary name and alias to its Tool implementation.
type Registry struct {
	tools map[string]Tool // keyed by lowercased Name() and all Aliases()
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register indexes the tool by its canonical name and every alias.
// Panics on collision — misconfigured registry is a programmer error, not a
// runtime error.
func (r *Registry) Register(t Tool) {
	keys := append([]string{t.Name()}, t.Aliases()...)
	for _, k := range keys {
		k = strings.ToLower(k)
		if _, exists := r.tools[k]; exists {
			continue
		}
		r.tools[k] = t
	}
}

// Get returns the Tool for a given name (case-insensitive).
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[strings.ToLower(name)]
	return t, ok
}

// Dispatch routes a parsed command through override checks then to the tool.
//
// Resolution order:
//  1. Tool lookup — unknown binary → shell-style "not recognized" error.
//  2. Override Blocked check — returns "Access is denied" if set.
//  3. Override Requires conditions — returns appropriate error if unmet.
//  4. tool.Execute().
func (r *Registry) Dispatch(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	p providers.Provider,
	overrides map[string]schema.ToolOverride,
) schema.ToolResult {
	tool, ok := r.Get(cmd.Binary)
	if !ok {
		return shellError(fmt.Sprintf(
			"'%s' is not recognized as an internal or external command, operable program or batch file.",
			cmd.Binary,
		))
	}

	// Override key: "<binary>.<subcommand>" — subcommand may be empty.
	overrideKey := cmd.Binary
	if cmd.Subcommand != "" {
		overrideKey = cmd.Binary + "." + cmd.Subcommand
	}
	if ovMap, exists := overrides[overrideKey]; exists {
		if ov, ok := ovMap[cmd.Subcommand]; ok {
			if ov.Blocked {
				return shellError("Access is denied.")
			}
			if err := checkRequires(ov.Requires, state, p); err != "" {
				return shellError(err)
			}
		}
	}

	return tool.Execute(cmd, state, p)
}

// checkRequires evaluates every Requires condition. Returns a non-empty error
// string on first failure, empty string on success.
func checkRequires(requires []string, state *schema.SessionState, p providers.Provider) string {
	for _, cond := range requires {
		switch cond {
		case "da_session":
			if !p.HasDomainAdmin(state.User) {
				return "This command requires a Domain Admin session."
			}
		case "local_admin":
			if !p.HasLocalAdmin(state.User, state.Hostname) {
				return fmt.Sprintf(
					"Access is denied. %s does not have local admin rights on %s.",
					state.User, state.Hostname,
				)
			}
		case "has_ticket":
			if len(state.Tickets) == 0 {
				return "No Kerberos tickets in memory. Import a ticket first."
			}
		case "has_ntlm":
			if len(state.Loot.NTLMHashes) == 0 {
				return "No NTLM hashes in memory."
			}
		default:
			// Unknown condition — fail safe.
			return fmt.Sprintf("Unknown session requirement: %q.", cond)
		}
	}
	return ""
}

// shellError returns a ToolResult that looks like a Windows shell error.
func shellError(msg string) schema.ToolResult {
	return schema.ToolResult{
		Output:      msg,
		IsError:     true,
		SideEffects: nil,
	}
}
// Names returns all registered tool names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for k := range r.tools {
		names = append(names, k)
	}
	return names
}

// RegisterAlias maps an additional key to an already-registered (or any) Tool.
// Unlike Register, it does not iterate Name()/Aliases() — just stores the key.
func (r *Registry) RegisterAlias(key string, t Tool) {
	r.tools[strings.ToLower(key)] = t
}
