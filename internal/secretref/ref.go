package secretref

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultCmdTimeout = 10 * time.Second

var ErrInvalidRef = errors.New("invalid secret reference")

type SecretRef struct {
	Scheme  string
	Name    string
	Path    string
	Service string
	Key     string
	Argv    []string
	Timeout time.Duration
}

func (r *SecretRef) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return r.parseString(node.Value)
	case yaml.MappingNode:
		return r.parseStructured(node)
	default:
		return fmt.Errorf("%w: must be a string or a cmd mapping", ErrInvalidRef)
	}
}

func (r *SecretRef) parseString(raw string) error {
	scheme, value, ok := strings.Cut(strings.TrimSpace(raw), ":")
	if !ok || scheme == "" {
		return fmt.Errorf("%w: secret refs require a provider scheme", ErrInvalidRef)
	}
	switch scheme {
	case "env":
		if value == "" {
			return fmt.Errorf("%w: env ref requires a variable name", ErrInvalidRef)
		}
		*r = SecretRef{Scheme: scheme, Name: value}
	case "file":
		if value == "" {
			return fmt.Errorf("%w: file ref requires a path", ErrInvalidRef)
		}
		*r = SecretRef{Scheme: scheme, Path: value}
	case "keyring":
		service, key, ok := strings.Cut(value, "/")
		if !ok || service == "" || key == "" || strings.Contains(key, "/") {
			return fmt.Errorf("%w: keyring refs must be service/key", ErrInvalidRef)
		}
		*r = SecretRef{Scheme: scheme, Service: service, Key: key}
	default:
		return fmt.Errorf("%w: unknown scheme %q", ErrInvalidRef, scheme)
	}
	return nil
}

func (r *SecretRef) parseStructured(node *yaml.Node) error {
	var model struct {
		Cmd *struct {
			Argv    []string `yaml:"argv"`
			Timeout string   `yaml:"timeout"`
		} `yaml:"cmd"`
	}
	if err := node.Decode(&model); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRef, err)
	}
	if model.Cmd == nil {
		return fmt.Errorf("%w: structured ref must contain cmd", ErrInvalidRef)
	}
	if len(model.Cmd.Argv) == 0 || strings.TrimSpace(model.Cmd.Argv[0]) == "" {
		return fmt.Errorf("%w: cmd.argv must be non-empty", ErrInvalidRef)
	}
	ref := SecretRef{Scheme: "cmd", Argv: append([]string(nil), model.Cmd.Argv...)}
	if model.Cmd.Timeout != "" {
		timeout, err := time.ParseDuration(model.Cmd.Timeout)
		if err != nil || timeout <= 0 {
			return fmt.Errorf("%w: cmd.timeout must be a positive duration", ErrInvalidRef)
		}
		ref.Timeout = timeout
	}
	*r = ref
	return nil
}
