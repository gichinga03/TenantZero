package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/server/engine/tools"
	"github.com/0x4D67/tenantzero/shared/schema"
)

type Session struct {
	ID        string
	Lab       schema.Lab
	State     schema.SessionState
	StartedAt time.Time
	Registry  *tools.Registry
	Provider  providers.Provider
}

func NewSession(lab schema.Lab, userID string) *Session {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		panic(err)
	}
	sessionID := hex.EncodeToString(idBytes)

	now := time.Now()

	return &Session{
		ID:        sessionID,
		Lab:       lab,
		StartedAt: now,
		State: schema.SessionState{
			LabID:               lab.LabID,
			UserID:              userID,
			ActiveHost:          lab.StartingPoint.Machine,
			ActiveUser:          lab.StartingPoint.User,
			ActiveBeaconID:      "b1a2c3d4e5f67890",
			CWD:                 `C:\Users\` + lab.StartingPoint.User,
			BeaconTree:          []schema.Beacon{
				{
					ID:          "b1a2c3d4e5f67890",
					Host:        lab.StartingPoint.Machine,
					Identity:    lab.StartingPoint.User,
					Active:      true,
					CheckedInAt: now,
				},
			},
			GrantedACLs:         map[string][]string{},
			Loot:                []schema.Loot{},
			Tickets:             []schema.Ticket{},
			Tokens:              []schema.Token{},
			CompletedObjectives: []schema.CompletedObjective{},
			Journal:             []schema.JournalEntry{},
			UsedHints:           []string{},
			StartedAt:           now,
			LastActivityAt:      now,
		},
	}
}

func (s *Session) ApplySideEffects(effects []schema.SideEffect) {
	for _, e := range effects {
		s.applySideEffect(e)
	}
}

func (s *Session) applySideEffect(e schema.SideEffect) {
	switch e.Type {

	case schema.SideEffectTypeLoot:
		s.State.Loot = append(s.State.Loot, schema.Loot{
			Type:       e.LootType,
			Value:      e.LootValue,
			Source:     e.LootSource,
			CapturedAt: time.Now(),
		})

	case schema.SideEffectTypeTicket:
		tkt := schema.Ticket{
			TicketType: e.TicketType,
			Identity:   e.Identity,
			Target:     e.Target,
			B64:        e.TicketB64,
			LUID:       e.TicketLUID,
			IssuedAt:   time.Now(),
			ExpiresAt:  time.Now().Add(10 * time.Hour),
			RenewTill:  time.Now().Add(7 * 24 * time.Hour),
		}
		if !e.TicketExpiresAt.IsZero() {
			tkt.ExpiresAt = e.TicketExpiresAt
		}
		if !e.TicketRenewTill.IsZero() {
			tkt.RenewTill = e.TicketRenewTill
		}
		s.State.Tickets = append(s.State.Tickets, tkt)

	case schema.SideEffectTypeSetSessionUser:
		s.State.ActiveUser = e.NewUser

	case schema.SideEffectTypeAddToken:
		s.State.Tokens = append(s.State.Tokens, schema.Token{
			Username: e.TokenUsername,
			Domain:   e.TokenDomain,
			Fake:     e.TokenFake,
			IssuedAt: time.Now(),
		})

	case schema.SideEffectTypeCompleteObj:
		for _, c := range s.State.CompletedObjectives {
			if c.ID == e.ObjectiveID {
				return
			}
		}
		s.State.CompletedObjectives = append(s.State.CompletedObjectives, schema.CompletedObjective{
			ID:          e.ObjectiveID,
			CompletedAt: time.Now(),
		})

	case schema.SideEffectTypeGrantACL:
		if s.State.GrantedACLs == nil {
			s.State.GrantedACLs = map[string][]string{}
		}
		key := e.Principal + "." + e.ACLTarget
		for _, existing := range s.State.GrantedACLs[key] {
			if existing == e.ACLRight {
				return
			}
		}
		s.State.GrantedACLs[key] = append(s.State.GrantedACLs[key], e.ACLRight)

	case schema.SideEffectTypeSpawnBeacon:
		newID := randomHex(8)
		beacon := schema.Beacon{
			ID:          newID,
			ParentID:    e.BeaconParentID,
			Host:        e.BeaconHost,
			Identity:    e.BeaconIdentity,
			Active:      true,
			CheckedInAt: time.Now(),
			Tasks:       []schema.Task{},
		}
		s.State.BeaconTree = append(s.State.BeaconTree, beacon)
		s.State.ActiveBeaconID = newID

	case schema.SideEffectTypePivotContext:
		s.State.ActiveHost = e.PivotHost
		s.State.ActiveUser = e.PivotIdentity

	case schema.SideEffectTypeScopeViolation:
		_ = fmt.Sprintf("scope violation attempted: %s", e.ViolationHost)
	}
}

func (s *Session) ActiveBeacon() (schema.Beacon, bool) {
	if s.State.ActiveBeaconID == "" {
		return schema.Beacon{}, false
	}
	for _, b := range s.State.BeaconTree {
		if b.ID == s.State.ActiveBeaconID {
			return b, true
		}
	}
	return schema.Beacon{}, false
}

func (s *Session) HasRuntimeACL(principal, target, right string) bool {
	key := principal + "." + target
	rights, ok := s.State.GrantedACLs[key]
	if !ok {
		return false
	}
	for _, r := range rights {
		if r == right {
			return true
		}
	}
	return false
}

func (s *Session) AppendTask(beaconID, input, output string) {
	for i, b := range s.State.BeaconTree {
		if b.ID == beaconID {
			task := schema.Task{
				ID:          randomHex(4),
				Input:       input,
				Output:      output,
				IssuedAt:    time.Now(),
				CompletedAt: time.Now(),
				Status:      "complete",
			}
			s.State.BeaconTree[i].Tasks = append(s.State.BeaconTree[i].Tasks, task)
			return
		}
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *Session) CheckTimeouts() bool {
	return time.Since(s.State.LastActivityAt) > time.Duration(s.Lab.Time.SessionTimeoutMinutes)*time.Minute
}

func (s *Session) AppendJournal(raw, output string) {
	s.State.Journal = append(s.State.Journal, schema.JournalEntry{
		Timestamp: time.Now(),
		Host:      s.State.ActiveHost,
		Identity:  s.State.ActiveUser,
		RawInput:  raw,
		Output:    output,
	})
	s.State.LastActivityAt = time.Now()
}
// effectiveCallers returns all identities this session can act as —
// the active user plus any tokens and ticket identities acquired.
// Both "DOMAIN\user" and bare "user" forms are included.
func (s *Session) effectiveCallers() []string {
	seen := map[string]bool{}
	var out []string

	add := func(id string) {
		if id == "" {
			return
		}
		key := strings.ToLower(id)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, id)
		// also add bare username so "MERIDIAN\awalker" matches "awalker"
		if idx := strings.LastIndexAny(id, `\/`); idx >= 0 {
			bare := id[idx+1:]
			bkey := strings.ToLower(bare)
			if !seen[bkey] {
				seen[bkey] = true
				out = append(out, bare)
			}
		}
	}

	add(s.State.ActiveUser)

	for _, t := range s.State.Tokens {
		if t.Domain != "" {
			add(t.Domain + `\` + t.Username)
		} else {
			add(t.Username)
		}
	}

	for _, tk := range s.State.Tickets {
		add(tk.Identity)
		if !strings.ContainsAny(tk.Identity, `\/`) && tk.Target != "" {
			domain := tk.Target
			if idx := strings.Index(domain, "."); idx >= 0 {
				domain = domain[:idx]
			}
			add(domain + `\` + tk.Identity)
		}
	}

	return out
}
