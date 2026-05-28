package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/credentials"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/secret"
)

const (
	EnvProfile          = "ZSCALERCTL_PROFILE"
	EnvVanityDomain     = "ZSCALERCTL_VANITY_DOMAIN"
	EnvCloud            = "ZSCALERCTL_CLOUD"
	EnvClientID         = "ZSCALERCTL_CLIENT_ID"
	EnvClientSecret     = "ZSCALERCTL_CLIENT_SECRET"
	EnvClientSecretFile = "ZSCALERCTL_CLIENT_SECRET_FILE"
	EnvRedaction        = "ZSCALERCTL_REDACTION"
	EnvNoCache          = "ZSCALERCTL_NO_CACHE"
)

type Config struct {
	Profile      string
	VanityDomain string
	Cloud        string
	Credentials  Credentials
	Defaults     Defaults
}

type Credentials struct {
	ClientID         secret.Secret
	ClientSecret     secret.Secret
	ClientSecretFile string
}

type Defaults struct {
	Redaction redact.Mode
	NoCache   bool
}

type SafeConfig struct {
	Profile         string           `json:"profile"`
	VanityDomainSet bool             `json:"vanity_domain_set"`
	Cloud           string           `json:"cloud,omitempty"`
	Credentials     CredentialStatus `json:"credentials"`
	Defaults        DefaultsView     `json:"defaults"`
}

func (SafeConfig) OutputSafe() {}

type CredentialStatus struct {
	ClientIDSet         bool `json:"client_id_set"`
	ClientSecretSet     bool `json:"client_secret_set"`
	ClientSecretFileSet bool `json:"client_secret_file_set"`
}

type DefaultsView struct {
	Redaction string `json:"redaction"`
	NoCache   bool   `json:"no_cache"`
}

func LoadEnv(environ []string) (Config, error) {
	env := parseEnv(environ)
	mode := redact.ModeStandard
	if value := env[EnvRedaction]; value != "" {
		parsed, err := redact.ParseMode(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", EnvRedaction, err)
		}
		mode = parsed
	}

	noCache, err := parseBoolEnv(env[EnvNoCache])
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", EnvNoCache, err)
	}

	clientSecret := secret.New(env[EnvClientSecret])
	if env[EnvClientSecretFile] != "" {
		fileSecret, err := credentials.ReadOwnerOnlySecretFile(env[EnvClientSecretFile])
		if err != nil {
			return Config{}, fmt.Errorf("load %s: %w", EnvClientSecretFile, err)
		}
		if !clientSecret.IsSet() {
			clientSecret = fileSecret
		}
	}

	cfg := Config{
		Profile:      env[EnvProfile],
		VanityDomain: strings.TrimSpace(env[EnvVanityDomain]),
		Cloud:        strings.TrimSpace(env[EnvCloud]),
		Credentials: Credentials{
			ClientID:         secret.New(env[EnvClientID]),
			ClientSecret:     clientSecret,
			ClientSecretFile: env[EnvClientSecretFile],
		},
		Defaults: Defaults{
			Redaction: mode,
			NoCache:   noCache,
		},
	}
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	return cfg, nil
}

func (c Config) Safe() SafeConfig {
	return SafeConfig{
		Profile:         c.Profile,
		VanityDomainSet: c.VanityDomain != "",
		Cloud:           c.Cloud,
		Credentials: CredentialStatus{
			ClientIDSet:         c.Credentials.ClientID.IsSet(),
			ClientSecretSet:     c.Credentials.ClientSecret.IsSet(),
			ClientSecretFileSet: c.Credentials.ClientSecretFile != "",
		},
		Defaults: DefaultsView{
			Redaction: string(c.Defaults.Redaction),
			NoCache:   c.Defaults.NoCache,
		},
	}
}

func parseEnv(environ []string) map[string]string {
	out := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		out[key] = value
	}
	return out
}

func parseBoolEnv(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return parsed, nil
}
