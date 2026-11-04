//go:build !darwin && !windows

package fileperm

import "os"

func validateOwnerOnlyACLFile(*os.File) error {
	return nil
}
