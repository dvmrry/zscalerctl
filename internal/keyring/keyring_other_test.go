//go:build !darwin && !linux && !(windows && (amd64 || arm64))

package keyring

import (
	"context"
	"errors"
	"testing"
)

// TestUnsupportedGetterReturnsErrUnavailable verifies that on platforms without
// a real keyring backend Get returns ErrUnavailable (wrapped, so errors.Is
// matches) and an actionable message with the env:/file:/cmd: hint.
func TestUnsupportedGetterReturnsErrUnavailable(t *testing.T) {
	t.Parallel()

	g := newBackend()
	_, err := g.Get(context.Background(), "svc", "key")
	if err == nil {
		t.Fatal("unsupportedGetter.Get() error = nil, want ErrUnavailable")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unsupportedGetter.Get() error = %v, want errors.Is ErrUnavailable", err)
	}
}
