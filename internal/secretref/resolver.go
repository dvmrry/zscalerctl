package secretref

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dvmrry/zscalerctl/internal/credentials"
	"github.com/dvmrry/zscalerctl/internal/secret"
)

var ErrNoResolver = errors.New("secret resolver is not configured")

type ResolverOpts struct{}

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
		return secret.Secret{}, fmt.Errorf("%w: cmd refs are not enabled in this build phase", ErrInvalidRef)
	case "keyring":
		return secret.Secret{}, fmt.Errorf("%w: keyring refs are not enabled in this build phase", ErrInvalidRef)
	default:
		return secret.Secret{}, fmt.Errorf("%w: unknown scheme %q", ErrInvalidRef, ref.Scheme)
	}
}
