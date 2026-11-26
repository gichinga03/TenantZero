// Package engine is the TenantZero lab runtime. It owns lab loading, session
// lifecycle, command dispatch, filesystem simulation, and win condition
// evaluation.
package engine

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// LoadLab reads lab.yaml and engagement.yaml from dir, unmarshals both, and
// validates the lab for required fields. engagement.yaml is optional — if
// absent, a zero Engagement is returned with no error.
func LoadLab(dir string) (*schema.Lab, *schema.Engagement, error) {
	labPath := filepath.Join(dir, "lab.yaml")
	labData, err := os.ReadFile(labPath)
	if err != nil {
		return nil, nil, fmt.Errorf("lab.go: read %s: %w", labPath, err)
	}

	var lab schema.Lab
	if err := yaml.Unmarshal(labData, &lab); err != nil {
		return nil, nil, fmt.Errorf("lab.go: parse %s: %w", labPath, err)
	}

	// Normalize filesystem keys to lowercase so normPath lookups match.
	if lab.FileSystem != nil {
		normalized := make(schema.FileSystem, len(lab.FileSystem))
		for k, v := range lab.FileSystem {
			k = strings.ReplaceAll(k, `\\`, `\`)
			k = strings.ReplaceAll(k, "/", `\`)
			k = strings.ToLower(k)
			normalized[k] = v
		}
		lab.FileSystem = normalized
	}
	if lab.FileContents != nil {
		normalized := make(schema.FileContents, len(lab.FileContents))
		for k, v := range lab.FileContents {
			// Collapse double backslashes (from YAML escaping) then lowercase.
			k = strings.ReplaceAll(k, `\\`, `\`)
			k = strings.ReplaceAll(k, "/", `\`)
			k = strings.ToLower(k)
			normalized[k] = v
		}
		lab.FileContents = normalized
	}

	if err := validateLab(&lab); err != nil {
		return nil, nil, fmt.Errorf("lab.go: validate %s: %w", labPath, err)
	}

	engPath := filepath.Join(dir, "engagement.yaml")
	engData, err := os.ReadFile(engPath)
	var eng schema.Engagement
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("lab.go: read %s: %w", engPath, err)
		}
		// engagement.yaml is optional
	} else {
		if err := yaml.Unmarshal(engData, &eng); err != nil {
			return nil, nil, fmt.Errorf("lab.go: parse %s: %w", engPath, err)
		}
	}

	return &lab, &eng, nil
}

// validateLab checks required fields and returns a descriptive error on the
// first violation found.
func validateLab(lab *schema.Lab) error {
	if lab.LabID == "" {
		return errors.New("lab.lab_id is required")
	}
	if lab.Domain.Name == "" {
		return errors.New("lab.domain.name is required")
	}
	if len(lab.WinConditions) == 0 {
		return errors.New("lab must define at least one win_condition")
	}
	return nil
}

// InitState constructs the initial SessionState for a lab run.
//
// Drop-in mode (lab.StartingPoint.Machine != ""): session starts on the
// configured machine as the configured user with the configured cwd.
//
// Full-chain mode: User and Hostname are empty, Cwd defaults to the external
// attacker working directory ("~/"). Tools that require a session user will
// emit appropriate errors via the registry Requires check.
func InitState(lab *schema.Lab, sessionID string) *schema.SessionState {
	state := &schema.SessionState{
		LabID:               lab.LabID,
		UserID:              sessionID,
		StartedAt:           time.Now().UTC(),
		LastActivityAt:      time.Now().UTC(),
		Loot:                []schema.Loot{},
		Tickets:             []schema.Ticket{},
		Tokens:              []schema.Token{},
		Journal:             []schema.JournalEntry{},
		CompletedObjectives: []schema.CompletedObjective{},
		BeaconTree:          []schema.Beacon{},
		GrantedACLs:         map[string][]string{},
		UsedHints:           []string{},
	}

	if lab.StartingPoint.Machine != "" {
		state.ActiveUser        = lab.StartingPoint.User
		state.ActiveHost        = lab.StartingPoint.Machine
		state.ActiveBeaconID    = "b1a2c3d4e5f67890"
		state.CWD               = `C:\`
		state.BeaconTree        = []schema.Beacon{
			{
				ID:          "b1a2c3d4e5f67890",
				Host:        lab.StartingPoint.Machine,
				Identity:    lab.StartingPoint.User,
				Active:      true,
				CheckedInAt: time.Now().UTC(),
			},
		}
	} else {
		state.CWD = `C:\`
	}

	return state
}