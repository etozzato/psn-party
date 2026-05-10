package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"psnadd/internal/app"
	"psnadd/internal/config"
	"psnadd/internal/db"
	"psnadd/internal/handlers"
	"psnadd/internal/services/groups"
	psnsvc "psnadd/internal/services/psn"
)

var Version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	portFlag := flag.Int("port", 0, "port to listen on")
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if *portFlag > 0 {
		cfg.Port = *portFlag
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.OpenPool(ctx, cfg)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		logger.Error("database migrations failed", "error", err)
		os.Exit(1)
	}

	psnService := psnsvc.New(logger, time.Duration(cfg.ProfileTimeoutSeconds)*time.Second)
	groupService := groups.New(pool, psnService)
	handler := handlers.New(cfg, groupService)
	router := app.NewRouter(cfg, logger, Version, handler)

	addr := fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting server", "name", cfg.Name, "env", cfg.Env, "version", Version, "address", addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("server shutdown complete")
	}
}
