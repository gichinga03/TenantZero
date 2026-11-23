package tools

import (
	"github.com/0x4D67/tenantzero/shared/schema"
	"github.com/0x4D67/tenantzero/server/engine/providers"
)

// Tool is the interface every simulated binary implements.
type Tool interface {
	Name()    string
	Aliases() []string
	Execute(cmd schema.ParsedCommand, state *schema.SessionState, p providers.Provider) schema.ToolResult
}

// SideEffectType constants mirror schema.SideEffect.Type values.
const (
	SetLoot            = "set_loot"
	SetSessionUser     = "set_session_user"
	SetSessionMachine  = "set_session_machine"
	AddTicket          = "add_ticket"
	AddToken           = "add_token"
	SetCwd             = "set_cwd"
)