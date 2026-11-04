//go:build !windows

package fileperm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestOpenOwnerOnlyPOSIXRejectsGroupOrWorldBits(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("profiles: {}\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	file, err := OpenOwnerOnly(path)
	if err != nil {
		t.Fatalf("OpenOwnerOnly(0600) error = %v, want nil", err)
	}
	_ = file.Close()
	for _, mode := range []os.FileMode{0o640, 0o604} {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatalf("os.Chmod(%q, %03o) error = %v, want nil", path, mode, err)
		}
		if file, err := OpenOwnerOnly(path); !errors.Is(err, ErrInsecurePermissions) {
			if err == nil {
				_ = file.Close()
			}
			t.Fatalf("OpenOwnerOnly(%03o) error = %v, want ErrInsecurePermissions", mode, err)
		}
	}
}

func TestWriteOwnerOnlyPOSIXCreates0600AndValidates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("profiles: {}\n")
	if err := WriteOwnerOnly(path, data); err != nil {
		t.Fatalf("WriteOwnerOnly(%q) error = %v, want nil", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", path, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("WriteOwnerOnly mode = %03o, want 600", info.Mode().Perm())
	}
	// Self-verify: the file it wrote must pass the read-side validator.
	file, err := OpenOwnerOnly(path)
	if err != nil {
		t.Fatalf("OpenOwnerOnly(written) error = %v, want nil", err)
	}
	_ = file.Close()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, err)
	}
	if string(got) != string(data) {
		t.Fatalf("WriteOwnerOnly wrote %q, want %q", got, data)
	}
}

func TestWriteOwnerOnlyPOSIXRefusesExisting(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := WriteOwnerOnly(path, []byte("first\n")); err != nil {
		t.Fatalf("WriteOwnerOnly(first) error = %v, want nil", err)
	}
	if err := WriteOwnerOnly(path, []byte("second\n")); !errors.Is(err, os.ErrExist) {
		t.Fatalf("WriteOwnerOnly(existing) error = %v, want os.ErrExist", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, err)
	}
	if string(got) != "first\n" {
		t.Fatalf("WriteOwnerOnly overwrote existing file: got %q", got)
	}
}

func TestOpenOwnerOnlyPOSIXRejectsSymlink(t *testing.T) {
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
	if file, err := OpenOwnerOnly(link); !errors.Is(err, ErrInsecurePermissions) {
		if err == nil {
			_ = file.Close()
		}
		t.Fatalf("OpenOwnerOnly(symlink) error = %v, want ErrInsecurePermissions", err)
	}
}

func TestOpenOwnerOnlyPOSIXRejectsFIFO(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "secret.fifo")
	if err := unix.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("unix.Mkfifo(%q) error = %v, want nil", path, err)
	}

	errCh := make(chan error, 1)
	go func() {
		file, err := OpenOwnerOnly(path)
		if err == nil {
			_ = file.Close()
		}
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrInsecurePermissions) {
			t.Fatalf("OpenOwnerOnly(fifo) error = %v, want ErrInsecurePermissions", err)
		}
		if !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("OpenOwnerOnly(fifo) error = %v, want non-regular file message", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OpenOwnerOnly(fifo) did not return; want non-regular file rejection")
	}
}
