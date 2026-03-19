package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           int
	DatabasePath   string
	SidecarURL     string
	SidecarTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:           getEnvInt("PRACTICEHELPER_SERVER_PORT", 8080),
		DatabasePath:   getEnv("PRACTICEHELPER_SERVER_DB_PATH", "../data/practicehelper.db"),
		SidecarURL:     getEnv("PRACTICEHELPER_SERVER_SIDECAR_URL", "http://127.0.0.1:8000"),
		SidecarTimeout: 30 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}
