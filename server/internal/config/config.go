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
	LogPath        string
}

func Load() Config {
	return Config{
		Port:           getEnvInt("PRACTICEHELPER_SERVER_PORT", 8090),
		DatabasePath:   getEnv("PRACTICEHELPER_SERVER_DB_PATH", "../data/practicehelper.db"),
		SidecarURL:     getEnv("PRACTICEHELPER_SERVER_SIDECAR_URL", "http://127.0.0.1:8000"),
		SidecarTimeout: getEnvDurationSeconds("PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS", 90*time.Second),
		LogPath:        getEnv("PRACTICEHELPER_SERVER_LOG_PATH", "../data/logs/server.log"),
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

func getEnvDurationSeconds(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds * float64(time.Second))
}
