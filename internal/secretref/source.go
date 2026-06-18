package secretref

import (
	"context"
	"sync"

	"github.com/dvmrry/zscalerctl/internal/secret"
)

// SecretSource carries safe provenance metadata and resolves to a secret only
// at the authentication boundary.
type SecretSource interface {
	Scheme() string
	IsConfigured() bool
	Resolve(context.Context) (secret.Secret, error)
}

type resolved struct {
	scheme string
	sec    secret.Secret
}

func Resolved(scheme string, sec secret.Secret) SecretSource {
	if !sec.IsSet() {
		return Unset()
	}
	return resolved{scheme: scheme, sec: sec}
}

func (r resolved) Scheme() string { return r.scheme }

func (r resolved) IsConfigured() bool { return r.sec.IsSet() }

func (r resolved) Resolve(context.Context) (secret.Secret, error) { return r.sec, nil }

type unset struct{}

func Unset() SecretSource { return unset{} }

func (unset) Scheme() string { return "" }

func (unset) IsConfigured() bool { return false }

func (unset) Resolve(context.Context) (secret.Secret, error) { return secret.Secret{}, nil }

type deferred struct {
	ref      SecretRef
	resolver interface {
		Resolve(context.Context, SecretRef) (secret.Secret, error)
	}

	once sync.Once
	sec  secret.Secret
	err  error
}

func Deferred(ref SecretRef, resolver interface {
	Resolve(context.Context, SecretRef) (secret.Secret, error)
}) SecretSource {
	return &deferred{ref: ref, resolver: resolver}
}

func (d *deferred) Scheme() string { return d.ref.Scheme }

func (d *deferred) IsConfigured() bool { return d.ref.Scheme != "" }

func (d *deferred) Resolve(ctx context.Context) (secret.Secret, error) {
	d.once.Do(func() {
		if d.resolver == nil {
			d.err = ErrNoResolver
			return
		}
		d.sec, d.err = d.resolver.Resolve(ctx, d.ref)
	})
	return d.sec, d.err
}
