package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sms/sms-store/config"
	"github.com/sms/sms-store/internal/handler"
	"github.com/sms/sms-store/internal/kafka"
	"github.com/sms/sms-store/internal/repository"
	"github.com/sms/sms-store/internal/service"
)

func main() {
	// ── Config ───────────────────────────────────────────────────────────────
	cfg := config.Load()

	// ── MongoDB repository ────────────────────────────────────────────────────
	repo, err := repository.New(cfg)
	if err != nil {
		log.Fatalf("[main] Failed to connect to MongoDB: %v", err)
	}

	// ── Service ───────────────────────────────────────────────────────────────
	svc := service.New(repo)

	// ── HTTP server ───────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	h := handler.New(svc)
	h.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      requestLogger(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Kafka consumer ─────────────────────────────────────────────────────────
	consumer := kafka.New(cfg, svc)
	ctx, cancel := context.WithCancel(context.Background())
	go consumer.Start(ctx)

	// ── Start HTTP ─────────────────────────────────────────────────────────────
	go func() {
		log.Printf("[main] SMS Store listening on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[main] HTTP server error: %v", err)
		}
	}()

	// ── Graceful shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[main] Shutting down...")
	cancel() // stop Kafka consumer

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[main] HTTP shutdown error: %v", err)
	}

	log.Println("[main] Shutdown complete")
}

// requestLogger is a simple middleware that logs every HTTP request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[http] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
