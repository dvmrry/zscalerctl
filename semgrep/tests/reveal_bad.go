//go:build semgrepfixtures

package tests

import "zscalerctl/internal/secret"

func leakSecret(value secret.Secret) string {
	return value.Reveal()
}
