package secretref

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/credentials"
	"github.com/dvmrry/zscalerctl/internal/secret"
)

var ErrNoResolver = errors.New("secret resolver is not configured")

type ResolverOpts struct {
	AllowCmd bool
}

type Resolver struct {
	opts ResolverOpts
}

func NewResolver(opts ResolverOpts) *Resolver {
	return &Resolver{opts: opts}
}

func (r *Resolver) Resolve(ctx context.Context, ref SecretRef) (secret.Secret, error) {
	select {
	case <-ctx.Done():
		return secret.Secret{}, ctx.Err()
	default:
	}

	switch ref.Scheme {
	case "env":
		value, ok := os.LookupEnv(ref.Name)
		if !ok {
			return secret.Secret{}, fmt.Errorf("%w: env ref is not set: %s", ErrInvalidRef, ref.Name)
		}
		return secret.New(value), nil
	case "file":
		return credentials.ReadOwnerOnlySecretFile(ref.Path)
	case "cmd":
		return r.resolveCmd(ctx, ref)
	case "keyring":
		return secret.Secret{}, fmt.Errorf("%w: keyring refs are not enabled in this build phase", ErrInvalidRef)
	default:
		return secret.Secret{}, fmt.Errorf("%w: unknown scheme %q", ErrInvalidRef, ref.Scheme)
	}
}

func (r *Resolver) resolveCmd(ctx context.Context, ref SecretRef) (secret.Secret, error) {
	if !r.opts.AllowCmd {
		return secret.Secret{}, fmt.Errorf("%w: cmd refs are disabled", ErrInvalidRef)
	}
	if len(ref.Argv) == 0 || strings.TrimSpace(ref.Argv[0]) == "" {
		return secret.Secret{}, fmt.Errorf("%w: cmd.argv must be non-empty", ErrInvalidRef)
	}
	timeout := ref.Timeout
	if timeout <= 0 {
		timeout = DefaultCmdTimeout
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 -- owner-only profile cmd refs intentionally execute the
	// operator-specified argv directly, with no shell and a bounded timeout.
	cmd := exec.CommandContext(cmdCtx, ref.Argv[0], ref.Argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return secret.Secret{}, fmt.Errorf("%w: cmd provider %q timed out after %s", ErrInvalidRef, ref.Argv[0], timeout)
		}
		if cmdCtx.Err() != nil {
			return secret.Secret{}, fmt.Errorf("%w: cmd provider %q cancelled", ErrInvalidRef, ref.Argv[0])
		}
		return secret.Secret{}, fmt.Errorf("%w: cmd provider %q failed: %s", ErrInvalidRef, ref.Argv[0], summarizeStderr(stderr.String()))
	}
	return secret.New(strings.TrimRight(stdout.String(), "\r\n")), nil
}

func summarizeStderr(stderr string) string {
	if stderr == "" {
		return "no stderr"
	}
	return fmt.Sprintf("stderr omitted (%d bytes)", len(stderr))
}
