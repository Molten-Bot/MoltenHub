package main

import (
	"log"
	"net/http"
	"os"

	"statocyst/internal/api"
	"statocyst/internal/auth"
	"statocyst/internal/longpoll"
	"statocyst/internal/store"
)

func main() {
	addr := os.Getenv("STATOCYST_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	st := store.NewMemoryStore()
	waiters := longpoll.NewWaiters()
	humanAuth := auth.NewHumanAuthProviderFromEnv()
	handler := api.NewHandler(
		st,
		waiters,
		humanAuth,
		os.Getenv("SUPABASE_URL"),
		os.Getenv("SUPABASE_ANON_KEY"),
	)
	router := api.NewRouter(handler)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("statocyst listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
