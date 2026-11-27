// Package api implements the HTTP REST handlers for TenantZero.
package api

import (
	"log"
	"errors"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine"
	"github.com/0x4D67/tenantzero/server/store"
	"github.com/0x4D67/tenantzero/shared/schema"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// contextKey is the unexported type for context keys in this package.
type contextKey string

// contextKeyUserID is the context key used by requireAuth to inject the user ID.
const contextKeyUserID contextKey = "userID"

// Handler holds the dependencies for all REST handlers.
type Handler struct {
	store   *store.Store
	labsDir string
	jwtKey  []byte
}

// NewHandler constructs an API Handler.
func NewHandler(st *store.Store, labsDir string, jwtKey []byte) *Handler {
	return &Handler{store: st, labsDir: labsDir, jwtKey: jwtKey}
}

// requireAuth is middleware that validates the JWT from the Authorization header
// and injects the user ID into the request context.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr == "" {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected alg: %v", t.Header["alg"])
			}
			return h.jwtKey, nil
		})
		if err != nil || !tok.Valid {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, _ := tok.Claims.(jwt.MapClaims)
		userID, _ := claims["sub"].(string)
		if userID == "" {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), contextKeyUserID, userID)))
	}
}

// handleRegister creates a new user account.
//
// POST /auth/register
// Body: {"username":"...","password":"..."}
// Response: {"id":"...","username":"..."}
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, err := h.store.CreateUser(body.Username, string(hash))
	if err != nil {
		jsonError(w, "username already taken", http.StatusConflict)
		return
	}
	jsonOK(w, map[string]string{"id": id, "username": body.Username})
}

// handleLogin authenticates a user and returns a signed JWT.
//
// POST /auth/login
// Body: {"username":"...","password":"..."}
// Response: {"token":"..."}
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return
	}
	user, err := h.store.GetUserByUsername(body.Username)
	if errors.Is(err, sql.ErrNoRows) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("[!] handleLogin GetUserByUsername: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := tok.SignedString(h.jwtKey)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"token": tokenStr})
}

// labMeta is the JSON shape returned by the labs listing and detail endpoints.
type labMeta struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Difficulty    string   `json:"difficulty"`
	Machines      int      `json:"machines"`
	Forests       int      `json:"forests"`
	Concepts      []string `json:"concepts"`
	EstimatedTime int      `json:"estimated_time_minutes"`
	EngagementID  string   `json:"engagement_id,omitempty"`
}

// handleListLabs walks labsDir and returns metadata for each lab.
//
// GET /labs
// Response: JSON array of labMeta
func (h *Handler) handleListLabs(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(filepath.Join(h.labsDir, "ad"))
	if err != nil {
		jsonError(w, "labs unavailable", http.StatusInternalServerError)
		return
	}
	var labs []labMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := h.loadLabMeta(e.Name())
		if err != nil {
			continue
		}
		labs = append(labs, meta)
	}
	jsonOK(w, labs)
}

// handleGetLab returns metadata for a single lab.
//
// GET /labs/{id}
// Response: labMeta
func (h *Handler) handleGetLab(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/labs/")
	id = strings.TrimSuffix(id, "/")
	if strings.Contains(id, "/") {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	meta, err := h.loadLabMeta(id)
	if err != nil {
		jsonError(w, "lab not found", http.StatusNotFound)
		return
	}
	jsonOK(w, meta)
}

// handleEngagementPDF generates and streams the engagement PDF for a lab.
//
// GET /labs/{id}/engagement.pdf
// Response: application/pdf
func (h *Handler) handleEngagementPDF(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// expect: labs / {id} / engagement.pdf
	if len(parts) != 3 {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	labID := parts[1]
	engPath := filepath.Join(h.labsDir, "ad", labID, "engagement.yaml")
	data, err := os.ReadFile(engPath)
	if err != nil {
		jsonError(w, "engagement not found", http.StatusNotFound)
		return
	}
	var eng schema.Engagement
	if err := yaml.Unmarshal(data, &eng); err != nil {
		jsonError(w, "invalid engagement", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := engine.Generate(&eng, &buf); err != nil {
		jsonError(w, "pdf generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-engagement.pdf"`, labID))
	w.Write(buf.Bytes())
}

// handleJournal returns journal entries for a session.
//
// GET /sessions/{id}/journal
// Response: JSON array of JournalRow
func (h *Handler) handleJournal(w http.ResponseWriter, r *http.Request) {
	sessID := extractSessionID(r.URL.Path, "journal")
	entries, err := h.store.GetJournal(sessID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, entries)
}

// handleProgress returns the completed objectives for a session.
//
// GET /sessions/{id}/progress
// Response: {"completed_objectives": [...]}
func (h *Handler) handleProgress(w http.ResponseWriter, r *http.Request) {
	sessID := extractSessionID(r.URL.Path, "progress")
	row, err := h.store.LoadSession(sessID)
	if err != nil {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}
	var state schema.SessionState
	if err := json.Unmarshal(row.State, &state); err != nil {
		jsonError(w, "state corrupt", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"completed_objectives": state.CompletedObjectives,
	})
}

// loadLabMeta reads lab.yaml for the given lab ID and returns labMeta.
func (h *Handler) loadLabMeta(id string) (labMeta, error) {
	labPath := filepath.Join(h.labsDir, "ad", id, "lab.yaml")
	data, err := os.ReadFile(labPath)
	if err != nil {
		return labMeta{}, err
	}
	var lab schema.Lab
	if err := yaml.Unmarshal(data, &lab); err != nil {
		return labMeta{}, err
	}
	return labMeta{
		ID:            id,
		Name:          lab.LabID,
		Difficulty:    "",
		Machines:      len(lab.Computers),
		Forests:       1 + len(lab.Trusts),
		Concepts:      nil,
		EstimatedTime: lab.Time.SessionTimeoutMinutes,
	}, nil
}

// extractSessionID pulls the session UUID from paths like /sessions/{id}/journal.
func extractSessionID(path, suffix string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	// sessions / {id} / suffix
	if len(parts) == 3 && parts[0] == "sessions" && parts[2] == suffix {
		return parts[1]
	}
	return ""
}

// jsonOK writes v as JSON with 200.
func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// jsonError writes msg as JSON with status.
func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
// handleBeaconGenerate creates a new beacon build entry in the active session.
//
// POST /api/beacon/generate
// Header: Authorization: Bearer <jwt>
// Body: {"sessionId":"...","listener":"http","format":"exe","arch":"x64"}
// Response: {"id":"a1b2c3","hostedPath":"/beacon-a1b2c3.exe","downloadUrl":"http://10.10.10.5/beacon-a1b2c3.exe"}
func (h *Handler) handleBeaconGenerate(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(contextKeyUserID).(string)

	var body struct {
		SessionID string `json:"sessionId"`
		Listener  string `json:"listener"`
		Format    string `json:"format"`
		Arch      string `json:"arch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.SessionID == "" {
		jsonError(w, "sessionId required", http.StatusBadRequest)
		return
	}

	listener := body.Listener
	if listener == "" {
		listener = "http"
	}
	format := body.Format
	if format == "" {
		format = "exe"
	}
	arch := body.Arch
	if arch == "" {
		arch = "x64"
	}

	log.Printf("handleBeaconGenerate: sessionId=%q userID=%q", body.SessionID, userID)
	// Always use the user's most recently active session so BeaconGen and
	// beacon consoles (which open separate WS sessions) share the same state.
	sess, err := h.store.LoadLatestSession(userID)
	if err != nil {
		log.Printf("handleBeaconGenerate: LoadLatestSession error: %v", err)
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}

	// Unmarshal state
	var state schema.SessionState
	if len(sess.State) > 0 {
		if err := json.Unmarshal(sess.State, &state); err != nil {
			log.Printf("handleBeaconGenerate: unmarshal error: %v", err)
			jsonError(w, "corrupt session state", http.StatusInternalServerError)
			return
		}
	}

	// Generate a 6-char hex build ID
	id := fmt.Sprintf("%06x", time.Now().UnixNano()&0xffffff)
	hostedPath := fmt.Sprintf("/beacon-%s.%s", id, format)
	downloadURL := fmt.Sprintf("http://10.10.10.5%s", hostedPath)

	state.Builds = append(state.Builds, schema.Build{
		ID:         id,
		Listener:   listener,
		Format:     format,
		Arch:       arch,
		HostedPath: hostedPath,
		CreatedAt:  time.Now(),
	})

	// Marshal state back
	stateJSON, err := json.Marshal(state)
	if err != nil {
		jsonError(w, "failed to marshal state", http.StatusInternalServerError)
		return
	}
	sess.State = stateJSON

	if err := h.store.SaveSession(sess); err != nil {
		log.Printf("handleBeaconGenerate: save session: %v", err)
		jsonError(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{
		"id":          id,
		"hostedPath":  hostedPath,
		"downloadUrl": downloadURL,
	})
}
