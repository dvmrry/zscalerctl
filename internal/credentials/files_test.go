package credentials_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/credentials"
)

func TestValidateOwnerOnlyFileRejectsGroupReadableFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte("fake-secret"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}

	err := credentials.ValidateOwnerOnlyFile(path)
	if !errors.Is(err, credentials.ErrUnsafePermissions) {
		t.Errorf("ValidateOwnerOnlyFile(%q) error = %v, want ErrUnsafePermissions", path, err)
	}
}

func TestValidateOwnerOnlyFileAcceptsOwnerOnlyFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte("fake-secret"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}

	if err := credentials.ValidateOwnerOnlyFile(path); err != nil {
		t.Errorf("ValidateOwnerOnlyFile(%q) error = %v, want nil", path, err)
	}
}

func TestReadOwnerOnlySecretFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte("fake-secret\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}

	got, err := credentials.ReadOwnerOnlySecretFile(path)
	if err != nil {
		t.Fatalf("ReadOwnerOnlySecretFile(%q) error = %v, want nil", path, err)
	}
	if got.Reveal() != "fake-secret" {
		t.Errorf("ReadOwnerOnlySecretFile(%q).Reveal() = %q, want %q", path, got.Reveal(), "fake-secret")
	}
}
