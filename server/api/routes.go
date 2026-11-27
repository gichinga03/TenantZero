// Package api registers all HTTP routes for TenantZero.
package api

import (
	"net/http"
	"strings"

	"github.com/0x4D67/tenantzero/server/ws"
)

// RegisterRoutes mounts all REST and WebSocket routes on mux.
//
// Routes:
//   POST   /auth/register              — create user account
//   POST   /auth/login                 — authenticate, return JWT
//   GET    /labs                       — list all labs
//   GET    /labs/{id}                  — single lab metadata
//   GET    /labs/{id}/engagement.pdf   — stream engagement PDF
//   GET    /sessions/{id}/journal      — command journal for session
//   GET    /sessions/{id}/progress     — completed objectives for session
//   GET    /ws                         — WebSocket upgrade
func RegisterRoutes(mux *http.ServeMux, h *Handler, wsHandler *ws.Handler) {
	mux.HandleFunc("POST /auth/register", h.handleRegister)
	mux.HandleFunc("POST /auth/login", h.handleLogin)

	mux.HandleFunc("GET /labs", h.handleListLabs)
	// /labs/{id} and /labs/{id}/engagement.pdf are disambiguated inside the handler.
	mux.HandleFunc("GET /labs/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/engagement.pdf") {
			h.handleEngagementPDF(w, r)
		} else {
			h.handleGetLab(w, r)
		}
	})

	mux.HandleFunc("GET /sessions/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/journal") {
			h.requireAuth(h.handleJournal)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/progress") {
			h.requireAuth(h.handleProgress)(w, r)
		} else {
			jsonError(w, "not found", http.StatusNotFound)
		}
	})

	mux.HandleFunc("POST /api/beacon/generate", h.requireAuth(h.handleBeaconGenerate))
	mux.HandleFunc("GET /ws", wsHandler.ServeWS)
}