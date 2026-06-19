//go:build windows

package fileperm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	accessAllowedObjectACEType         = 0x5
	accessAllowedCallbackACEType       = 0x9
	accessAllowedCallbackObjectACEType = 0xb
)

const windowsReadWriteMask = windows.ACCESS_MASK(
	windows.GENERIC_READ |
		windows.GENERIC_WRITE |
		windows.GENERIC_ALL |
		windows.FILE_GENERIC_READ |
		windows.FILE_GENERIC_WRITE |
		windows.FILE_READ_DATA |
		windows.FILE_WRITE_DATA |
		windows.FILE_APPEND_DATA |
		windows.READ_CONTROL |
		windows.DELETE |
		windows.WRITE_DAC |
		windows.WRITE_OWNER,
)

func validate(path string) error {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read file security descriptor: %w", err)
	}
	return validateSecurityDescriptor(sd)
}

func validateOpenFile(file *os.File) error {
	sd, err := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read file security descriptor: %w", err)
	}
	return validateSecurityDescriptor(sd)
}

func validateSecurityDescriptor(sd *windows.SECURITY_DESCRIPTOR) error {
	if sd == nil {
		return fmt.Errorf("%w: missing security descriptor", ErrInsecurePermissions)
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return fmt.Errorf("read security descriptor owner: %w", err)
	}
	if owner == nil {
		return fmt.Errorf("%w: missing owner", ErrInsecurePermissions)
	}
	if sidIsBlocked(owner) {
		return fmt.Errorf("%w: broad Windows owner %s", ErrInsecurePermissions, owner.String())
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("%w: missing DACL", ErrInsecurePermissions)
	}
	if dacl == nil {
		return fmt.Errorf("%w: empty DACL", ErrInsecurePermissions)
	}

	allowed, err := windowsAllowedSIDs(owner)
	if err != nil {
		return err
	}

	for i := uint16(0); i < dacl.AceCount; i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			return fmt.Errorf("read DACL ACE: %w", err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			if isUnsupportedAllowACEType(ace.Header.AceType) && ace.Mask&windowsReadWriteMask != 0 {
				return fmt.Errorf("%w: unsupported Windows allow ACE type %d", ErrInsecurePermissions, ace.Header.AceType)
			}
			continue
		}
		if ace.Mask&windowsReadWriteMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if sidIsBlocked(sid) {
			return fmt.Errorf("%w: broad Windows principal %s has read/write access", ErrInsecurePermissions, sid.String())
		}
		if sidInSet(sid, allowed) {
			continue
		}
		if ace.Header.AceFlags&windows.INHERITED_ACE != 0 {
			return fmt.Errorf("%w: inherited non-owner principal %s has read/write access", ErrInsecurePermissions, sid.String())
		}
		return fmt.Errorf("%w: non-owner principal %s has read/write access", ErrInsecurePermissions, sid.String())
	}
	return nil
}

func writeOwnerOnly(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- caller-supplied config path; created O_EXCL and locked down via icacls + re-validated below.
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close file: %w", err)
	}
	// Newly created files inherit the parent directory's DACL, which on a shared
	// or fold-redirected profile can grant broad principals. Break inheritance
	// and grant ONLY the current user so the file passes validateOpenFile.
	if err := lockDownToCurrentUser(path); err != nil {
		_ = os.Remove(path)
		return err
	}
	// Self-verify against the same read-side validator the loader uses; never
	// leave a config whose DACL the loader would later reject.
	verify, err := openOwnerOnly(path)
	if err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("verify owner-only permissions: %w", err)
	}
	_ = verify.Close()
	return nil
}

// lockDownToCurrentUser breaks DACL inheritance on path and grants full control
// to only the current user, via the absolute icacls path with no shell. This is
// the verified A9 recipe: `icacls <path> /inheritance:r /grant:r <user>:F`.
func lockDownToCurrentUser(path string) error {
	user, err := currentUserName()
	if err != nil {
		return err
	}
	icacls := filepath.Join(os.Getenv("SystemRoot"), "System32", "icacls.exe")
	cmd := exec.Command(icacls, path, "/inheritance:r", "/grant:r", user+":F") // #nosec G204 -- absolute system binary, fixed flags; path is caller-supplied and user is derived from the process token.
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("icacls lock down %s: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// currentUserName returns the account name (DOMAIN\user) for the process token
// user, the form icacls expects for a /grant principal. It falls back to the
// USERNAME environment variable only when the token lookup fails.
func currentUserName() (string, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err == nil {
		defer token.Close()
		if user, uerr := token.GetTokenUser(); uerr == nil {
			if account, domain, _, aerr := user.User.Sid.LookupAccount(""); aerr == nil {
				if domain != "" {
					return domain + `\` + account, nil
				}
				return account, nil
			}
		}
	}
	if name := strings.TrimSpace(os.Getenv("USERNAME")); name != "" {
		return name, nil
	}
	return "", fmt.Errorf("%w: cannot determine current Windows user for owner-only write", ErrInsecurePermissions)
}

func openOwnerOnly(path string) (*os.File, error) {
	file, err := os.Open(path) // #nosec G304 -- caller-supplied config/secret path; security descriptor is validated on the opened handle before reads.
	if err != nil {
		return nil, err
	}
	if err := validateOpenFile(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func windowsAllowedSIDs(owner *windows.SID) ([]*windows.SID, error) {
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, fmt.Errorf("create SYSTEM SID: %w", err)
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, fmt.Errorf("create Administrators SID: %w", err)
	}
	current, err := currentUserSID()
	if err != nil {
		return nil, err
	}
	return []*windows.SID{owner, current, system, admins}, nil
}

func currentUserSID() (*windows.SID, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return nil, fmt.Errorf("open current process token: %w", err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("read current user SID: %w", err)
	}
	sid, err := user.User.Sid.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy current user SID: %w", err)
	}
	return sid, nil
}

func sidIsBlocked(sid *windows.SID) bool {
	types := []windows.WELL_KNOWN_SID_TYPE{
		windows.WinWorldSid,
		windows.WinBuiltinUsersSid,
		windows.WinAuthenticatedUserSid,
		windows.WinAccountDomainUsersSid,
		windows.WinInteractiveSid,
	}
	for _, typ := range types {
		if sid.IsWellKnown(typ) {
			return true
		}
	}
	return false
}

func isUnsupportedAllowACEType(aceType byte) bool {
	return aceType == accessAllowedObjectACEType ||
		aceType == accessAllowedCallbackACEType ||
		aceType == accessAllowedCallbackObjectACEType
}

func sidInSet(sid *windows.SID, set []*windows.SID) bool {
	for _, candidate := range set {
		if sid.Equals(candidate) {
			return true
		}
	}
	return false
}
