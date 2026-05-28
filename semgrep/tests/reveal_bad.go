//go:build semgrepfixtures

package tests

import "github.com/dvmrry/zscalerctl/internal/secret"

func leakSecret(value secret.Secret) string {
	return value.Reveal()
}
