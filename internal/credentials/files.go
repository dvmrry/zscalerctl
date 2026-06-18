package credentials

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/fileperm"
	"github.com/dvmrry/zscalerctl/internal/secret"
)

var ErrUnsafePermissions = errors.New("unsafe credential file permissions")

// ReadOwnerOnlySecretFile opens the file, checks permissions on the open
// handle (eliminating the TOCTOU race between a separate stat and read), reads
// the contents, and returns the trimmed secret value.
//
// This is the only supported way to consume an owner-only credential file.
// Do not add a path-based "validate then read later" helper: checking
// permissions with a separate os.Stat reintroduces the stat-then-use race
// that checking the open handle exists to close.
func ReadOwnerOnlySecretFile(path string) (secret.Secret, error) {
	f, err := fileperm.OpenOwnerOnly(path)
	if err != nil {
		if errors.Is(err, fileperm.ErrInsecurePermissions) {
			return secret.Secret{}, fmt.Errorf("%w: %w", ErrUnsafePermissions, err)
		}
		return secret.Secret{}, fmt.Errorf("open credential file: %w", err)
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return secret.Secret{}, fmt.Errorf("read credential file: %w", err)
	}
	return secret.New(strings.TrimRight(string(body), "\r\n")), nil
}
