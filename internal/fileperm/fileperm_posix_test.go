//go:build !windows

package fileperm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePOSIXOwnerOnly(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("profiles: {}\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	if err := Validate(path); err != nil {
		t.Fatalf("Validate(0600) error = %v, want nil", err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("os.Chmod(%q, 0640) error = %v, want nil", path, err)
	}
	if err := Validate(path); !errors.Is(err, ErrInsecurePermissions) {
		t.Fatalf("Validate(0640) error = %v, want ErrInsecurePermissions", err)
	}
	if err := os.Chmod(path, 0o604); err != nil {
		t.Fatalf("os.Chmod(%q, 0604) error = %v, want nil", path, err)
	}
	if err := Validate(path); !errors.Is(err, ErrInsecurePermissions) {
		t.Fatalf("Validate(0604) error = %v, want ErrInsecurePermissions", err)
	}
}

func TestValidatePOSIXRejectsSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "target.yaml")
	if err := os.WriteFile(target, []byte("profiles: {}\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", target, err)
	}
	link := filepath.Join(dir, "config.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("os.Symlink(%q, %q) error = %v, want nil", target, link, err)
	}
	if err := Validate(link); !errors.Is(err, ErrInsecurePermissions) {
		t.Fatalf("Validate(symlink) error = %v, want ErrInsecurePermissions", err)
	}
	if file, err := OpenOwnerOnly(link); !errors.Is(err, ErrInsecurePermissions) {
		if err == nil {
			_ = file.Close()
		}
		t.Fatalf("OpenOwnerOnly(symlink) error = %v, want ErrInsecurePermissions", err)
	}
}
