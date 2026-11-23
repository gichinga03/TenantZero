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
	if err := db.Migrate("./server/store/migrations"); err != nil {
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

	log.Printf("[*] TenantZero server listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}