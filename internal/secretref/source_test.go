package secretref

import (
	"context"
	"errors"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/secret"
)

func TestResolvedSourceReturnsCapturedSecret(t *testing.T) {
	t.Parallel()

	src := Resolved("env", secret.New("hunter2"))
	if src.Scheme() != "env" || !src.IsConfigured() {
		t.Fatalf("Scheme()=%q IsConfigured()=%v, want env/true", src.Scheme(), src.IsConfigured())
	}
	got, err := src.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if got.Reveal() != "hunter2" {
		t.Errorf("Resolve().Reveal() = %q, want hunter2", got.Reveal())
	}
}

func TestUnsetSourceIsNotConfigured(t *testing.T) {
	t.Parallel()

	src := Unset()
	if src.Scheme() != "" || src.IsConfigured() {
		t.Fatalf("Scheme()=%q IsConfigured()=%v, want unset", src.Scheme(), src.IsConfigured())
	}
	got, err := src.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if got.IsSet() {
		t.Errorf("Resolve().IsSet() = true, want false")
	}
}

func TestDeferredSourceResolvesOnce(t *testing.T) {
	t.Parallel()

	resolver := &countingResolver{sec: secret.New("resolved")}
	src := Deferred(SecretRef{Scheme: "test"}, resolver)
	for range 2 {
		got, err := src.Resolve(context.Background())
		if err != nil {
			t.Fatalf("Resolve() error = %v, want nil", err)
		}
		if got.Reveal() != "resolved" {
			t.Fatalf("Resolve().Reveal() = %q, want resolved", got.Reveal())
		}
	}
	if resolver.calls != 1 {
		t.Errorf("resolver calls = %d, want 1", resolver.calls)
	}
}

func TestDeferredSourceCachesErrors(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	resolver := &countingResolver{err: want}
	src := Deferred(SecretRef{Scheme: "test"}, resolver)
	for range 2 {
		_, err := src.Resolve(context.Background())
		if !errors.Is(err, want) {
			t.Fatalf("Resolve() error = %v, want %v", err, want)
		}
	}
	if resolver.calls != 1 {
		t.Errorf("resolver calls = %d, want 1", resolver.calls)
	}
}

type countingResolver struct {
	calls int
	sec   secret.Secret
	err   error
}

func (r *countingResolver) Resolve(context.Context, SecretRef) (secret.Secret, error) {
	r.calls++
	return r.sec, r.err
}
