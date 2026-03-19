package config

import (
	"testing"
	"time"
)

func TestLoadReadsSidecarTimeoutFromEnv(t *testing.T) {
	t.Setenv("PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS", "135")

	cfg := Load()

	if cfg.SidecarTimeout != 135*time.Second {
		t.Fatalf("unexpected sidecar timeout: got %s want %s", cfg.SidecarTimeout, 135*time.Second)
	}
}

func TestLoadFallsBackWhenSidecarTimeoutEnvIsInvalid(t *testing.T) {
	t.Setenv("PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS", "oops")

	cfg := Load()

	if cfg.SidecarTimeout != 90*time.Second {
		t.Fatalf("unexpected fallback timeout: got %s want %s", cfg.SidecarTimeout, 90*time.Second)
	}
}
