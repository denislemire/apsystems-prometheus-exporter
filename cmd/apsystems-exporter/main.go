package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/denislemire/apsystems-prometheus-exporter/internal/config"
	"github.com/denislemire/apsystems-prometheus-exporter/internal/exporter"
	"github.com/denislemire/apsystems-prometheus-exporter/internal/layout"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	panels, err := layout.Load(cfg.LayoutPath)
	if err != nil {
		slog.Warn("panel layout not loaded; using uid/channel labels only", "err", err, "path", cfg.LayoutPath)
		panels = layout.File{Panels: map[string]layout.Panel{}}
	}

	exp, err := exporter.New(cfg, panels)
	if err != nil {
		slog.Error("exporter init", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := exp.Run(ctx); err != nil {
		slog.Error("exporter stopped", "err", err)
		os.Exit(1)
	}
}
