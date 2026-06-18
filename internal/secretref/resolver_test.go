package secretref

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolverResolvesEnvRef(t *testing.T) {
	t.Setenv("ZSCALERCTL_TEST_SECRET", "from-env")
	got, err := NewResolver(ResolverOpts{}).Resolve(context.Background(), SecretRef{Scheme: "env", Name: "ZSCALERCTL_TEST_SECRET"})
	if err != nil {
		t.Fatalf("Resolve(env) error = %v, want nil", err)
	}
	if got.Reveal() != "from-env" {
		t.Errorf("Resolve(env).Reveal() = %q, want from-env", got.Reveal())
	}
}

func TestResolverRejectsMissingEnvRef(t *testing.T) {
	t.Parallel()

	if _, err := NewResolver(ResolverOpts{}).Resolve(context.Background(), SecretRef{Scheme: "env", Name: "ZSCALERCTL_TEST_MISSING"}); err == nil {
		t.Fatal("Resolve(missing env) error = nil, want error")
	}
}

func TestResolverResolvesFileRef(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}

	got, err := NewResolver(ResolverOpts{}).Resolve(context.Background(), SecretRef{Scheme: "file", Path: path})
	if err != nil {
		t.Fatalf("Resolve(file) error = %v, want nil", err)
	}
	if got.Reveal() != "from-file" {
		t.Errorf("Resolve(file).Reveal() = %q, want from-file", got.Reveal())
	}
}

func TestResolverRejectsUnknownScheme(t *testing.T) {
	t.Parallel()

	if _, err := NewResolver(ResolverOpts{}).Resolve(context.Background(), SecretRef{Scheme: "bogus"}); err == nil {
		t.Fatal("Resolve(unknown) error = nil, want error")
	}
}
