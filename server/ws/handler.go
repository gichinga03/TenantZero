// Package ws handles WebSocket connections for TenantZero lab sessions.
package ws

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine"
	"github.com/0x4D67/tenantzero/server/engine/providers/ad"
	"github.com/0x4D67/tenantzero/server/engine/tools"
	"github.com/0x4D67/tenantzero/server/store"
	"github.com/0x4D67/tenantzero/shared/schema"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// clientMsg is the wire format for messages received from the client.
type clientMsg struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// serverMsg is the wire format for messages sent to the client.
type serverMsg struct {
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`
	Payload   string `json:"payload"`
}

// upgrader configures the WebSocket upgrade.
// CheckOrigin returns true unconditionally — restrict this in production.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Handler wires together the store, lab directory, and tool registry to serve
// WebSocket lab sessions.
type Handler struct {
	store    *store.Store
	labsDir  string
	registry *tools.Registry
	jwtKey   []byte
}

// NewHandler constructs a Handler.
func NewHandler(st *store.Store, labsDir string, registry *tools.Registry, jwtKey []byte) *Handler {
	return &Handler{store: st, labsDir: labsDir, registry: registry, jwtKey: jwtKey}
}

// ServeWS upgrades the HTTP connection to WebSocket and runs the lab session
// message loop for the authenticated user.
//
// Auth: JWT is read from the `token` query parameter or the Authorization header
// (Bearer scheme). The sub claim must be a valid user ID.
//
// Session lifecycle:
//   - If a session_id query param is present, the existing session is loaded.
//   - Otherwise a new session is created for the lab_id query param.
//
// Message loop (per command):
//  1. Receive JSON {"type":"command","payload":"..."}
//  2. Parse → engine.Parse
//  3. Dispatch → session.Dispatch
//  4. Apply side effects → session.Apply
//  5. Mail tick → mailClient.Tick
//  6. Win condition check → evaluator.Check
//  7. Append journal → session.AppendJournal + store.AppendJournal
//  8. Persist state → store.SaveSession
//  9. Send {"type":"response","payload":"..."} to client
//  10. Send {"type":"event",...} for each PushEvent
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, err := h.authenticate(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error: %v", err)
		return
	}
	defer conn.Close()

	sess, mailClient, evaluator, sessRow, err := h.initSession(r, userID)
	if err != nil {
		writeMsg(conn, serverMsg{Type: "event", EventType: "error", Payload: err.Error()})
		return
	}

	// Send initial beacon state so frontend can replace SEED_BEACONS
	initPayload, _ := json.Marshal(sess.State.BeaconTree)
	writeMsg(conn, serverMsg{Type: "event", EventType: "session_init", Payload: string(initPayload)})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg clientMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		// Handle beacon context switch.
		if msg.Type == "switch_beacon" {
			beaconID := msg.Payload
			if beaconID == "" {
				continue
			}
			for _, b := range sess.State.BeaconTree {
				if b.ID == beaconID {
					sess.State.ActiveBeaconID = b.ID
					sess.State.ActiveHost = b.Host
					sess.State.ActiveUser = b.Identity
					writeMsg(conn, serverMsg{Type: "response", Payload: ""})
					break
				}
			}
			continue
		}

		if msg.Type != "command" {
			continue
		}

		// Parse and dispatch.
		parsed := engine.Parse(msg.Payload)
		result := sess.Dispatch(parsed)

		// Apply side effects.
		sess.ApplySideEffects(result.Effects)
		var pushEvents []schema.PushEvent

		// Mail tick.
		mailPush, mailEffect := mailClient.Tick()
		if mailEffect != nil {
			sess.ApplySideEffects([]schema.SideEffect{*mailEffect})
		}
		if mailPush != nil {
			pushEvents = append(pushEvents, *mailPush)
		}

		// Win condition check.
		winPush := evaluator.Check(parsed, result)
		pushEvents = append(pushEvents, winPush...)

		// Journal.
		sess.AppendJournal(msg.Payload, result.Output)
		storeEntry := store.JournalRow{
			SessionID:  sessRow.ID,
			Sequence:   len(sess.State.Journal),
			Input:      msg.Payload,
			Output:     result.Output,
			Machine:    sess.State.ActiveHost,
			Username:   sess.State.ActiveUser,
			RecordedAt: time.Now().UTC(),
		}
		if err := h.store.AppendJournal(storeEntry); err != nil {
			log.Printf("ws: journal append error: %v", err)
		}

		// Persist state.
		stateJSON, _ := json.Marshal(sess.State)
		sessRow.State = stateJSON
		sessRow.LastActive = time.Now().UTC()
		if err := h.store.SaveSession(sessRow); err != nil {
			log.Printf("ws: save session error: %v", err)
		}

		// Send response.
		writeMsg(conn, serverMsg{Type: "response", Payload: result.Output})

		// Emit beacon_spawned event for any SpawnBeacon side effects.
		for _, eff := range result.Effects {
			if eff.Type == schema.SideEffectTypeSpawnBeacon {
				// Find the newly added beacon — match on host only since identity may differ in casing.
				log.Printf("ws: looking for beacon host=%q identity=%q tree=%d", eff.BeaconHost, eff.BeaconIdentity, len(sess.State.BeaconTree))
				for _, b := range sess.State.BeaconTree {
					log.Printf("ws: tree beacon host=%q identity=%q active=%v", b.Host, b.Identity, b.Active)
					if strings.EqualFold(b.Host, eff.BeaconHost) && b.Active {
						beaconJSON, _ := json.Marshal(b)
						log.Printf("ws: sending beacon_spawned: %s", string(beaconJSON))
						writeMsg(conn, serverMsg{
							Type:      "event",
							EventType: "beacon_spawned",
							Payload:   string(beaconJSON),
						})
						break
					}
				}
			}
		}

		// Send push events.
		for _, ev := range pushEvents {
			payloadStr, _ := ev.Payload.(string)
			writeMsg(conn, serverMsg{
				Type:      "event",
				EventType: ev.Type,
				Payload:   payloadStr,
			})
			if ev.Type == "lab_complete" {
				h.store.MarkComplete(sessRow.ID)
			}
		}
	}

	// Final persist on disconnect.
	stateJSON, _ := json.Marshal(sess.State)
	sessRow.State = stateJSON
	sessRow.LastActive = time.Now().UTC()
	h.store.SaveSession(sessRow)
}

// authenticate extracts and validates the JWT from the request.
// Accepts token query param or Authorization: Bearer <token> header.
func (h *Handler) authenticate(r *http.Request) (string, error) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		auth := r.Header.Get("Authorization")
		tokenStr = strings.TrimPrefix(auth, "Bearer ")
	}
	if tokenStr == "" {
		return "", fmt.Errorf("no token")
	}
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return h.jwtKey, nil
	})
	if err != nil || !tok.Valid {
		return "", fmt.Errorf("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing sub claim")
	}
	return sub, nil
}

// initSession loads an existing session or creates a new one.
// Returns the engine.Session, mail client, win condition evaluator, and store row.
func (h *Handler) initSession(r *http.Request, userID string) (
	*engine.Session,
	*engine.MailClient,
	*engine.Evaluator,
	store.SessionRow,
	error,
) {
	sessID := r.URL.Query().Get("session_id")
	labID := r.URL.Query().Get("lab_id")

	var sessRow store.SessionRow
	var state schema.SessionState
	var lab schema.Lab

	if sessID != "" {
		// Load existing session.
		row, err := h.store.LoadSession(sessID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil, nil, store.SessionRow{}, fmt.Errorf("session not found: %s", sessID)
			}
			return nil, nil, nil, store.SessionRow{}, err
		}
		if err := json.Unmarshal(row.State, &state); err != nil {
			return nil, nil, nil, store.SessionRow{}, fmt.Errorf("unmarshal state: %w", err)
		}
		labID = row.LabID
		sessRow = row

		labPath := fmt.Sprintf("%s/ad/%s", h.labsDir, labID)
		loadedLab, _, err := engine.LoadLab(labPath)
		if err != nil {
			return nil, nil, nil, store.SessionRow{}, fmt.Errorf("load lab: %w", err)
		}
		lab = *loadedLab
	} else {
		// New session.
		if labID == "" {
			return nil, nil, nil, store.SessionRow{}, fmt.Errorf("lab_id required")
		}
		labPath := fmt.Sprintf("%s/ad/%s", h.labsDir, labID)
		loadedLab, _, err := engine.LoadLab(labPath)
		if err != nil {
			return nil, nil, nil, store.SessionRow{}, fmt.Errorf("load lab: %w", err)
		}
		lab = *loadedLab
		newSessID := uuid.New().String()
			state = *engine.InitState(&lab, newSessID)
		sessRow = store.SessionRow{
			ID:        uuid.New().String(),
			UserID:    userID,
			LabID:     labID,
			StartedAt: time.Now().UTC(),
		}
		stateJSON, _ := json.Marshal(state)
		sessRow.State = stateJSON
		sessRow.LastActive = time.Now().UTC()
		if err := h.store.SaveSession(sessRow); err != nil {
			return nil, nil, nil, store.SessionRow{}, fmt.Errorf("save new session: %w", err)
		}
	}

	adProvider := ad.NewADProvider(lab, &state)
	sess := engine.NewSession(lab, userID)
	sess.State = state
	sess.Provider = adProvider
	sess.Registry = h.registry
	mailClient := engine.NewMailClient(&lab, &state)
	evaluator := engine.NewEvaluator(&lab, &sess.State, adProvider)
	return sess, mailClient, evaluator, sessRow, nil
}

// writeMsg serialises and sends a serverMsg to the WebSocket connection.
func writeMsg(conn *websocket.Conn, msg serverMsg) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("ws: write error: %v", err)
	}
}

// contextKey is an unexported type for context keys in this package.
type contextKey string

// ContextKeyUserID is the context key for the authenticated user ID.
const ContextKeyUserID contextKey = "userID"

// withUserID returns a context with the userID stored under ContextKeyUserID.
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}
// keepAlive sends a ping every 30s to prevent idle connection drops.
func keepAlive(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}
