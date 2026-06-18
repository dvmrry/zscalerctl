package secretref

import (
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestSecretRefStringForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want SecretRef
	}{
		{name: "env", raw: "env:ZS_SECRET", want: SecretRef{Scheme: "env", Name: "ZS_SECRET"}},
		{name: "file", raw: "file:/etc/zscalerctl/secret", want: SecretRef{Scheme: "file", Path: "/etc/zscalerctl/secret"}},
		{name: "keyring", raw: "keyring:zscalerctl/prod-client-secret", want: SecretRef{Scheme: "keyring", Service: "zscalerctl", Key: "prod-client-secret"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got SecretRef
			if err := got.UnmarshalYAML(yamlScalar(tt.raw)); err != nil {
				t.Fatalf("UnmarshalYAML(%q) error = %v, want nil", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalYAML(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSecretRefRejectsInvalidStringForms(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"",
		"noscheme",
		"bogus:value",
		"env:",
		"file:",
		"keyring:onlyservice",
		"keyring:/key",
		"keyring:service/",
		"keyring:service/key/extra",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			var ref SecretRef
			if err := ref.UnmarshalYAML(yamlScalar(raw)); err == nil {
				t.Fatalf("UnmarshalYAML(%q) error = nil, want error", raw)
			}
		})
	}
}

func TestSecretRefStructuredCmd(t *testing.T) {
	t.Parallel()

	var got SecretRef
	if err := got.UnmarshalYAML(yamlNode(t, "cmd:\n  argv: [\"/bin/get\", \"--profile\", \"prod\"]\n  timeout: 5s\n")); err != nil {
		t.Fatalf("UnmarshalYAML(cmd) error = %v, want nil", err)
	}
	want := SecretRef{Scheme: "cmd", Argv: []string{"/bin/get", "--profile", "prod"}, Timeout: 5 * time.Second}
	if got.Scheme != want.Scheme || got.Timeout != want.Timeout || len(got.Argv) != len(want.Argv) {
		t.Fatalf("UnmarshalYAML(cmd) = %+v, want %+v", got, want)
	}
	for i := range want.Argv {
		if got.Argv[i] != want.Argv[i] {
			t.Fatalf("Argv[%d] = %q, want %q", i, got.Argv[i], want.Argv[i])
		}
	}
}

func TestSecretRefStructuredCmdRejectsEmptyArgv(t *testing.T) {
	t.Parallel()

	var ref SecretRef
	if err := ref.UnmarshalYAML(yamlNode(t, "cmd:\n  argv: []\n")); err == nil {
		t.Fatal("UnmarshalYAML(cmd empty argv) error = nil, want error")
	}
}

func TestSecretRefStructuredCmdRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()

	var ref SecretRef
	if err := ref.UnmarshalYAML(yamlNode(t, "cmd:\n  argv: [\"/bin/get\"]\n  timeout: 0s\n")); err == nil {
		t.Fatal("UnmarshalYAML(cmd zero timeout) error = nil, want error")
	}
}

func yamlScalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
}

func yamlNode(t *testing.T, raw string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &node); err != nil {
		t.Fatalf("yaml.Unmarshal(%q) error = %v, want nil", raw, err)
	}
	if len(node.Content) != 1 {
		t.Fatalf("yaml.Unmarshal(%q) produced %d root nodes, want 1", raw, len(node.Content))
	}
	return node.Content[0]
}
