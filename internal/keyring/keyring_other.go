//go:build !darwin && !linux && !(windows && (amd64 || arm64))

package keyring

import (
	"context"
	"fmt"
)

type unsupportedGetter struct{}

func newBackend() Getter {
	return unsupportedGetter{}
}

func (unsupportedGetter) Get(context.Context, string, string) (string, error) {
	// Wrap ErrUnavailable so callers (e.g. secretref resolver) surface the
	// actionable hint rather than a generic "keyring lookup failed" message.
	return "", fmt.Errorf("keyring: not supported on this platform; use env:/file:/cmd: (%w)", ErrUnavailable)
}
