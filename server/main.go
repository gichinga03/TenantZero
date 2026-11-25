// Package main is the TenantZero server entry point.
package main

import (
	"log"
	"net/http"

	adtools "github.com/0x4D67/tenantzero/server/engine/tools/ad"
	"github.com/0x4D67/tenantzero/server/engine/tools"
	"github.com/0x4D67/tenantzero/server/api"
	"github.com/0x4D67/tenantzero/server/config"
	"github.com/0x4D67/tenantzero/server/store"
	"github.com/0x4D67/tenantzero/server/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[!] %v", err)
	}

	db, err := store.New(cfg.DBURL)
	if err != nil {
		log.Fatalf("[!] store.New: %v", err)
	}
	if err := db.Migrate("./store/migrations"); err != nil {
		log.Fatalf("[!] migrate: %v", err)
	}

	registry := tools.NewRegistry()
	registry.Register(&adtools.Rubeus{})
	registry.Register(&adtools.PowerView{})
	registry.Register(&adtools.Mimikatz{})
	registry.Register(&adtools.CrackMapExec{})
	registry.Register(&adtools.Impacket{})
	registry.Register(&adtools.Certify{})

	wsHandler := ws.NewHandler(db, cfg.LabsDir, registry, []byte(cfg.JWTKey))
	apiHandler := api.NewHandler(db, cfg.LabsDir, []byte(cfg.JWTKey))

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, apiHandler, wsHandler)

	// CORS for React dev server on :5173
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})

	log.Printf("[*] TenantZero server listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
}