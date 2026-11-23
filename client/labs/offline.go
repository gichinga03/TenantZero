package labs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/client/journal"
	"github.com/0x4D67/tenantzero/client/terminal"
	"github.com/0x4D67/tenantzero/server/engine"
	"github.com/0x4D67/tenantzero/server/engine/providers/ad"
	"github.com/0x4D67/tenantzero/server/engine/tools"
	adtools "github.com/0x4D67/tenantzero/server/engine/tools/ad"
	"github.com/0x4D67/tenantzero/shared/schema"
)

type OfflineRunner struct {
	labID    string
	cacheDir string
}

func NewOfflineRunner(labID, cacheDir string) *OfflineRunner {
	return &OfflineRunner{labID: labID, cacheDir: cacheDir}
}

func (r *OfflineRunner) Run() error {
	labPath := filepath.Join(r.cacheDir, r.labID)
	lab, eng, err := engine.LoadLab(labPath)
	if err != nil {
		return fmt.Errorf("offline: load lab: %w", err)
	}

	statePath := filepath.Join(r.cacheDir, r.labID, "state.json")
	var state *schema.SessionState

	data, err := os.ReadFile(statePath)
	if err == nil {
		var s schema.SessionState
		if json.Unmarshal(data, &s) == nil && s.LabID != "" {
			state = &s
		}
	}
	if state == nil {
		state = engine.InitState(lab, r.labID)
	}

	registry := tools.NewRegistry()
	registry.Register(&adtools.Rubeus{})
	registry.Register(&adtools.PowerView{})
	registry.Register(&adtools.Mimikatz{})
	registry.Register(&adtools.CrackMapExec{})
	registry.Register(&adtools.Impacket{})
	registry.Register(&adtools.Certify{})

	provider := ad.New(lab, state)
	sess := engine.NewSession(lab, eng, state, provider, registry)

	hostname := "WORKSTATION"
	if state.Hostname != "" {
		hostname = state.Hostname
	}

	renderer := terminal.NewRenderer(hostname, state.User, `C:\`)
	renderer.ClearScreen()
	fmt.Printf("[*] Starting offline session: %s\n\n", r.labID)

	journalDir := filepath.Join(r.cacheDir, "..", "..", "journals")
	j, err := journal.New(journalDir, r.labID)
	if err != nil {
		return fmt.Errorf("offline: journal: %w", err)
	}
	defer j.Close()

	var history []string

	for {
		input, err := terminal.ReadLine(renderer.Prompt(), history)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, terminal.ErrInterrupt) {
				continue
			}
			return err
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		history = append(history, input)

		if input == "clear" || input == "cls" {
			renderer.ClearScreen()
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}

		parsed := engine.Parse(input)
		result := sess.Dispatch(parsed)
		sess.Apply(result)

		for _, eff := range result.SideEffects {
			switch eff.Type {
			case schema.SideEffectSetSessionMachine:
				renderer.UpdateContext(eff.Value, "", "")
			case schema.SideEffectSetSessionUser:
				renderer.UpdateContext("", eff.Value, "")
			case schema.SideEffectSetCWD:
				renderer.UpdateContext("", "", eff.Value)
			}
		}

		renderer.Print(result.Output)

		_ = j.Append(journal.Entry{
			Input:     input,
			Output:    result.Output,
			Machine:   state.Hostname,
			Username:  state.User,
			Timestamp: time.Now(),
		})
	}

	saveState(statePath, state)
	return nil
}

func saveState(path string, state *schema.SessionState) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}
