package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                     int
	DatabasePath             string
	SidecarURL               string
	SidecarTimeout           time.Duration
	InternalToken            string
	LogPath                  string
	VectorStoreURL           string
	VectorStoreAPIKey        string
	VectorStoreCollection    string
	VectorStoreTimeout       time.Duration
	VectorWriteEnabled       bool
	VectorReadEnabled        bool
	VectorRerankEnabled      bool
	MemoryHotIndexTimeout    time.Duration
	MemoryEmbeddingClaimTTL  time.Duration
	MemoryEmbeddingPollEvery time.Duration
}

func Load() Config {
	return Config{
		Port:                     getEnvInt("PRACTICEHELPER_SERVER_PORT", 8090),
		DatabasePath:             getEnv("PRACTICEHELPER_SERVER_DB_PATH", "../data/practicehelper.db"),
		SidecarURL:               getEnv("PRACTICEHELPER_SERVER_SIDECAR_URL", "http://127.0.0.1:8000"),
		SidecarTimeout:           getEnvDurationSeconds("PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS", 90*time.Second),
		InternalToken:            getEnv("PRACTICEHELPER_INTERNAL_TOKEN", ""),
		LogPath:                  getEnv("PRACTICEHELPER_SERVER_LOG_PATH", "../data/logs/server.log"),
		VectorStoreURL:           getEnv("PRACTICEHELPER_SERVER_VECTOR_STORE_URL", ""),
		VectorStoreAPIKey:        getEnv("PRACTICEHELPER_SERVER_VECTOR_STORE_API_KEY", ""),
		VectorStoreCollection:    getEnv("PRACTICEHELPER_SERVER_VECTOR_STORE_COLLECTION", "practicehelper_memory"),
		VectorStoreTimeout:       getEnvDurationSeconds("PRACTICEHELPER_SERVER_VECTOR_STORE_TIMEOUT_SECONDS", 8*time.Second),
		VectorWriteEnabled:       getEnvBool("PRACTICEHELPER_SERVER_VECTOR_WRITE_ENABLED", false),
		VectorReadEnabled:        getEnvBool("PRACTICEHELPER_SERVER_VECTOR_READ_ENABLED", false),
		VectorRerankEnabled:      getEnvBool("PRACTICEHELPER_SERVER_VECTOR_RERANK_ENABLED", false),
		MemoryHotIndexTimeout:    getEnvDurationSeconds("PRACTICEHELPER_SERVER_MEMORY_HOT_INDEX_TIMEOUT_SECONDS", 2*time.Second),
		MemoryEmbeddingClaimTTL:  getEnvDurationSeconds("PRACTICEHELPER_SERVER_MEMORY_EMBEDDING_CLAIM_TTL_SECONDS", 20*time.Second),
		MemoryEmbeddingPollEvery: getEnvDurationSeconds("PRACTICEHELPER_SERVER_MEMORY_EMBEDDING_POLL_SECONDS", 4*time.Second),
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

func getEnvBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	switch strconv.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
