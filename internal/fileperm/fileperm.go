package fileperm

import (
	"errors"
	"os"
)

var ErrInsecurePermissions = errors.New("unsafe file permissions")

func Validate(path string) error {
	return validate(path)
}

func ValidateOpenFile(file *os.File) error {
	return validateOpenFile(file)
}

func OpenOwnerOnly(path string) (*os.File, error) {
	return openOwnerOnly(path)
}
