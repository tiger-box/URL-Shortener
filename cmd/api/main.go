package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tiger-box/URL-Shortener/internal"
	"github.com/tiger-box/URL-Shortener/internal/config"
	"github.com/tiger-box/URL-Shortener/internal/handler"
)

func main() {
	var lvl slog.LevelVar
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &lvl})
	logger := slog.New(h)
	slog.SetDefault(logger)

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}
	lvl.Set(cfg.LogLevel)

	shortener := internal.NewURLShortener()
	urlHandler := handler.NewURL(shortener, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("POST /shorten", urlHandler.GetShorter)
	mux.HandleFunc("GET /{code}", urlHandler.GetURL)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.WithTimeout(mux, 5*time.Second),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	serverErr := make(chan error, 1)
	// Запускаем сервер на порту 8080
	logger.Info("server starting", "addr", cfg.HTTPAddr)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Error("run failed", "err", err)
		os.Exit(1)
	case <-stop:
		logger.Info("shutting down server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}
