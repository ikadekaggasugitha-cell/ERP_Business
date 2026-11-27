package eventbus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTLSConfigDisabledByDefault(t *testing.T) {
	t.Setenv("KAFKA_TLS_ENABLED", "")
	cfg, err := loadTLSConfig()
	if err != nil || cfg != nil {
		t.Fatalf("want nil config and nil error, got cfg=%v err=%v", cfg, err)
	}
}

func TestLoadTLSConfigUsesSystemRootsWithoutCA(t *testing.T) {
	t.Setenv("KAFKA_TLS_ENABLED", "true")
	t.Setenv("KAFKA_TLS_CA_FILE", "")
	cfg, err := loadTLSConfig()
	if err != nil || cfg == nil || cfg.RootCAs != nil {
		t.Fatalf("want system-roots config, got cfg=%v err=%v", cfg, err)
	}
}

func TestLoadTLSConfigFailsClosedOnBadCA(t *testing.T) {
	t.Setenv("KAFKA_TLS_ENABLED", "true")

	t.Setenv("KAFKA_TLS_CA_FILE", filepath.Join(t.TempDir(), "missing.crt"))
	if _, err := loadTLSConfig(); err == nil {
		t.Fatal("missing CA file must be an error, not a plaintext fallback")
	}

	bad := filepath.Join(t.TempDir(), "bad.crt")
	if err := os.WriteFile(bad, []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KAFKA_TLS_CA_FILE", bad)
	if _, err := loadTLSConfig(); err == nil {
		t.Fatal("invalid PEM must be an error")
	}
}
