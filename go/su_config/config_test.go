package su_config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testConfig struct {
	Name    string        `json:"name"`
	Port    int           `json:"port"`
	Enabled bool          `json:"enabled"`
	Timeout time.Duration `json:"timeout"`
	Nested  nestedConfig  `json:"nested"`
}

type nestedConfig struct {
	Addr string `json:"addr"`
}

func TestLoadAndEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"name":"svc","port":1000,"enabled":false,"timeout":1000000000,"nested":{"addr":"a"}}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("APP_PORT", "2000")
	t.Setenv("APP_ENABLED", "true")
	t.Setenv("APP_TIMEOUT", "2s")
	t.Setenv("APP_NESTED_ADDR", "b")

	var cfg testConfig
	if err := LoadWithEnv(path, "APP", &cfg); err != nil {
		t.Fatalf("LoadWithEnv() error = %v", err)
	}
	if cfg.Name != "svc" || cfg.Port != 2000 || !cfg.Enabled || cfg.Timeout != 2*time.Second || cfg.Nested.Addr != "b" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}
