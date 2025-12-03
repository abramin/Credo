package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"id-gateway/internal/platform/config"
	"id-gateway/internal/platform/httpserver"
	"id-gateway/internal/platform/logger"
	httptransport "id-gateway/internal/transport/http"
)

// main wires high-level dependencies, exposes the HTTP router, and keeps the
// server lifecycle small. Business logic lives in internal services packages.
func main() {
	cfg := config.FromEnv()
	log := logger.New()

	// TODO: introduce real services when domain logic is implemented.
	handler := httptransport.NewHandler(cfg.RegulatedMode)
	router := httptransport.NewRouter(handler)

	srv := httpserver.New(cfg.Addr, router)

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
