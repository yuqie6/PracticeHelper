package main

import (
	"fmt"
	"log/slog"
	"os"

	"practicehelper/server/internal/config"
	"practicehelper/server/internal/controller"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
	"practicehelper/server/internal/sidecar"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repository, err := repo.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open store failed", "error", err)
		os.Exit(1)
	}
	defer repository.Close()

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
