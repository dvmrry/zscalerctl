package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/config"
)

func TestLoadEnvSafeConfigDoesNotExposeSecrets(t *testing.T) {
	t.Parallel()

	const clientID = "client-id-value"
	const clientSecret = "client-secret-value"
	cfg, err := config.LoadEnv([]string{
		config.EnvProfile + "=prod",
		config.EnvVanityDomain + "=acme",
		config.EnvCloud + "=zscalerthree",
		config.EnvClientID + "=" + clientID,
		config.EnvClientSecret + "=" + clientSecret,
		config.EnvNoCache + "=true",
	})
	if err != nil {
		t.Fatalf("LoadEnv() error = %v, want nil", err)
	}

	body, err := json.Marshal(cfg.Safe())
	if err != nil {
		t.Fatalf("json.Marshal(Config.Safe()) error = %v, want nil", err)
	}
	got := string(body)
	for _, forbidden := range []string{clientID, clientSecret, "acme"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("json.Marshal(Config.Safe()) = %s, want no %q", got, forbidden)
		}
	}
	if !cfg.Safe().Credentials.ClientIDSet || !cfg.Safe().Credentials.ClientSecretSet {
		t.Errorf("Config.Safe().Credentials = %+v, want client id and secret marked set", cfg.Safe().Credentials)
	}
	if !cfg.Safe().VanityDomainSet {
		t.Errorf("Config.Safe().VanityDomainSet = false, want true")
	}
}

func TestLoadEnvRejectsRedactionOff(t *testing.T) {
	t.Parallel()

	if _, err := config.LoadEnv([]string{config.EnvRedaction + "=off"}); err == nil {
		t.Errorf("LoadEnv(%s=off) error = nil, want error", config.EnvRedaction)
	}
}

func TestLoadEnvLoadsOwnerOnlySecretFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "client-secret.txt")
	if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}

	cfg, err := config.LoadEnv([]string{config.EnvClientSecretFile + "=" + path})
	if err != nil {
		t.Fatalf("LoadEnv(secret file) error = %v, want nil", err)
	}
	if cfg.Credentials.ClientSecret.Reveal() != "file-secret" {
		t.Errorf("LoadEnv(secret file).Credentials.ClientSecret = %q, want %q", cfg.Credentials.ClientSecret.Reveal(), "file-secret")
	}
}

func TestLoadEnvRejectsUnsafeSecretFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "client-secret.txt")
	if err := os.WriteFile(path, []byte("file-secret\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}

	if _, err := config.LoadEnv([]string{config.EnvClientSecretFile + "=" + path}); err == nil {
		t.Errorf("LoadEnv(unsafe secret file) error = nil, want error")
	}
}
