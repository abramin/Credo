package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	httpapi "id-gateway/internal/http"
)

// main wires high-level dependencies, exposes the HTTP router, and keeps the
// server lifecycle small. Business logic lives in internal/domain and friends.
func main() {
	cfg := loadConfig()

	// TODO: introduce real services when domain logic is implemented.
	handler := httpapi.NewHandler(cfg.RegulatedMode)
	router := httpapi.NewRouter(handler)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("starting id-gateway on %s", cfg.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown placeholder for when we add background resources.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}

type Config struct {
	Addr          string
	RegulatedMode bool
}

// loadConfig keeps configuration concerns out of main for readability.
func loadConfig() Config {
	addr := os.Getenv("ID_GATEWAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	regulated := os.Getenv("REGULATED_MODE") == "true"
	return Config{
		Addr:          addr,
		RegulatedMode: regulated,
	}
}
