package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"practicehelper/server/internal/config"
	"practicehelper/server/internal/controller"
	"practicehelper/server/internal/infra/sqlite"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
	"practicehelper/server/internal/sidecar"
)

func main() {
	cfg := config.Load()

	logger, cleanup, err := buildLogger(cfg.LogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build logger failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	slog.SetDefault(logger)

	db, err := sqlite.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open sqlite failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	if err := sqlite.Bootstrap(db); err != nil {
		logger.Error("bootstrap sqlite failed", "error", err)
		os.Exit(1)
	}

	repository := repo.New(db)

	sidecarClient := sidecar.New(cfg.SidecarURL, cfg.SidecarTimeout)
	svc := service.New(repository, sidecarClient)
	router := controller.NewRouter(svc)

	address := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("practicehelper server started", "addr", address, "db", cfg.DatabasePath, "sidecar", cfg.SidecarURL)

	if err := router.Run(address); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func buildLogger(logPath string) (*slog.Logger, func(), error) {
	handlerOptions := &slog.HandlerOptions{Level: slog.LevelInfo}
	writer := io.Writer(os.Stdout)
	cleanup := func() {}

	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, cleanup, fmt.Errorf("create log dir: %w", err)
		}

		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, cleanup, fmt.Errorf("open log file: %w", err)
		}

		writer = io.MultiWriter(os.Stdout, file)
		cleanup = func() { _ = file.Close() }
	}

	return slog.New(slog.NewJSONHandler(writer, handlerOptions)), cleanup, nil
}
