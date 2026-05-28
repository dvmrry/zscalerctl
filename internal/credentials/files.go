package credentials

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/secret"
)

var ErrUnsafePermissions = errors.New("unsafe credential file permissions")

func ValidateOwnerOnlyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat credential file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %s is a directory", ErrUnsafePermissions, path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: %s mode %03o", ErrUnsafePermissions, path, info.Mode().Perm())
	}
	return nil
}

func ReadOwnerOnlySecretFile(path string) (secret.Secret, error) {
	if err := ValidateOwnerOnlyFile(path); err != nil {
		return secret.Secret{}, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return secret.Secret{}, fmt.Errorf("read credential file: %w", err)
	}
	return secret.New(strings.TrimRight(string(body), "\r\n")), nil
}
