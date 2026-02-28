package main

import (
	"context"
	"flag"
	"net/http"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"portfoliopulse/internal/api"
	"portfoliopulse/internal/db"
	"portfoliopulse/internal/market"
	"portfoliopulse/internal/realtime"
	"portfoliopulse/internal/store"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var (
		addr   = flag.String("addr", ":8080", "server listen address")
		dbPath = flag.String("db", envOr("DB_PATH", "./portfoliopulse.db"), "sqlite database file")
	)
	flag.Parse()

	sqlDB, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer sqlDB.Close()

	st := store.NewSQLiteStore(sqlDB)
	provider := market.NewProvider()
	hub := realtime.NewHub()
	apiServer := api.NewServer(st, provider, hub)

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go apiServer.StartPolling(ctx, 30*time.Second)

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("PortfolioPulse backend listening on %s", *addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
