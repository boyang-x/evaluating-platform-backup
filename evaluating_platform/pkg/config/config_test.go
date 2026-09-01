package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsMaclawConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
maclaw:
  base_url: "http://127.0.0.1:18080"
  runtime_kind: "maclawsrv"
  runtime_mode: "external"
  capability_profile: "native-catalog"
  admin_secret: "admin-secret-from-file"
  provisioning_enabled: true
  timeout_seconds: 17
  redteam_target_concurrency: 7
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Maclaw.BaseURL != "http://127.0.0.1:18080" {
		t.Fatalf("base url = %q", cfg.Maclaw.BaseURL)
	}
	if cfg.Maclaw.RuntimeKind != "maclawsrv" {
		t.Fatalf("runtime kind = %q", cfg.Maclaw.RuntimeKind)
	}
	if cfg.Maclaw.RuntimeMode != "external" {
		t.Fatalf("runtime mode = %q", cfg.Maclaw.RuntimeMode)
	}
	if cfg.Maclaw.CapabilityProfile != "native-catalog" {
		t.Fatalf("capability profile = %q", cfg.Maclaw.CapabilityProfile)
	}
	if cfg.Maclaw.AdminSecret != "admin-secret-from-file" {
		t.Fatalf("admin secret = %q", cfg.Maclaw.AdminSecret)
	}
	if !cfg.Maclaw.ProvisioningEnabled {
		t.Fatalf("expected maclaw provisioning to be enabled")
	}
	if cfg.Maclaw.TimeoutSeconds != 17 {
		t.Fatalf("timeout = %d", cfg.Maclaw.TimeoutSeconds)
	}
	if cfg.Maclaw.RedteamTargetConcurrency != 7 {
		t.Fatalf("redteam target concurrency = %d", cfg.Maclaw.RedteamTargetConcurrency)
	}
}

func TestLoadDefaultsMaclawProvisioningToEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 8080\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Maclaw.ProvisioningEnabled {
		t.Fatalf("maclaw provisioning should be enabled by default")
	}
}

func TestLoadReadsRuntimeEnvOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", "18082")
	t.Setenv("DATABASE_DSN", "host=127.0.0.1 user=postgres dbname=e2e")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("STORAGE_ENDPOINT", "127.0.0.1:9000")
	t.Setenv("MACLAW_BASE_URL", "http://maclaw-runtime:18080")
	t.Setenv("MACLAW_RUNTIME_KIND", "maclawsrv")
	t.Setenv("MACLAW_RUNTIME_MODE", "compose")
	t.Setenv("MACLAW_CAPABILITY_PROFILE", "shadow")
	t.Setenv("MACLAW_REDTEAM_TARGET_CONCURRENCY", "9")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  port: 8080
database:
  dsn: "host=postgres user=postgres dbname=file"
redis:
  addr: "redis:6379"
storage:
  endpoint: "minio:9000"
maclaw:
  base_url: "http://127.0.0.1:18080"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 18082 {
		t.Fatalf("server port = %d", cfg.Server.Port)
	}
	if cfg.Database.DSN != "host=127.0.0.1 user=postgres dbname=e2e" {
		t.Fatalf("database dsn = %q", cfg.Database.DSN)
	}
	if cfg.Redis.Addr != "127.0.0.1:6379" {
		t.Fatalf("redis addr = %q", cfg.Redis.Addr)
	}
	if cfg.Storage.Endpoint != "127.0.0.1:9000" {
		t.Fatalf("storage endpoint = %q", cfg.Storage.Endpoint)
	}
	if cfg.Maclaw.BaseURL != "http://maclaw-runtime:18080" {
		t.Fatalf("maclaw base url = %q", cfg.Maclaw.BaseURL)
	}
	if cfg.Maclaw.RuntimeKind != "maclawsrv" {
		t.Fatalf("maclaw runtime kind = %q", cfg.Maclaw.RuntimeKind)
	}
	if cfg.Maclaw.RuntimeMode != "compose" {
		t.Fatalf("maclaw runtime mode = %q", cfg.Maclaw.RuntimeMode)
	}
	if cfg.Maclaw.CapabilityProfile != "shadow" {
		t.Fatalf("maclaw capability profile = %q", cfg.Maclaw.CapabilityProfile)
	}
	if cfg.Maclaw.RedteamTargetConcurrency != 9 {
		t.Fatalf("redteam target concurrency = %d", cfg.Maclaw.RedteamTargetConcurrency)
	}
}
