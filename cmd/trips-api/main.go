package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/applycoding/k8s-microservice-deploy/internal/config"
	"github.com/applycoding/k8s-microservice-deploy/internal/httpapi"
	"github.com/applycoding/k8s-microservice-deploy/internal/store"
)

func main() {
	cfg := config.FromEnv()
	log := newLogger(cfg.LogLevel)
	srv := httpapi.New(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go connectStore(ctx, cfg, srv, log)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown", "err", err)
		}
	}()

	log.Info("listening", "addr", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func connectStore(ctx context.Context, cfg config.Config, srv *httpapi.Server, log *slog.Logger) {
	if strings.EqualFold(os.Getenv("STORE"), "memory") {
		srv.SetStore(store.NewMemory())
		log.Info("using in-memory store")
		return
	}

	backoff := time.Second
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		db, err := store.Connect(attemptCtx, cfg.Database.DSN())
		cancel()
		if err == nil {
			srv.SetStore(db)
			log.Info("connected to postgres", "host", cfg.Database.Host, "database", cfg.Database.Name)
			go func() {
				<-ctx.Done()
				db.Close()
			}()
			return
		}
		log.Warn("postgres not ready", "host", cfg.Database.Host, "err", err.Error())
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 5*time.Second {
			backoff += time.Second
		}
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
