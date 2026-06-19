//go:build windows

package fileperm

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	accessAllowedObjectACEType         = 0x5
	accessAllowedCallbackACEType       = 0x9
	accessAllowedCallbackObjectACEType = 0xb
)

// Flags for GetFinalPathNameByHandleW. Both are 0 in the Win32 headers
// (FILE_NAME_NORMALIZED, VOLUME_NAME_DOS) but x/sys/windows does not export
// named constants for them, so we spell the value out with this name.
const fileNameNormalizedDOS = 0

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
	// Open a handle so the volume check and the security-descriptor read see
	// the same object (TOCTOU-consistent with the handle-based entry point).
	file, err := os.Open(path) // #nosec G304 -- caller-supplied config/secret path; the security descriptor is validated on the opened handle before any reads.
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return validateOpenFile(file)
}

func validateOpenFile(file *os.File) error {
	path := file.Name()
	handle := windows.Handle(file.Fd())

	// Run the volume restriction BEFORE the DACL check so a network/removable/
	// non-NTFS/UNC file produces the clear remediation message instead of a
	// cryptic SID error (DAV-38: NFS/SMB/FAT files surface as opaque SIDs).
	if err := validateLocalFixedNTFS(handle, path); err != nil {
		return err
	}

	sd, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read file security descriptor: %w", err)
	}
	return validateSecurityDescriptor(path, sd)
}

func validateSecurityDescriptor(path string, sd *windows.SECURITY_DESCRIPTOR) error {
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
		return rejectSID(path, owner)
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
				return fmt.Errorf("%w: %s has an unsupported Windows allow ACE type %d; reset its permissions:  icacls %q /inheritance:r /grant:r \"%%USERNAME%%:F\"",
					ErrInsecurePermissions, path, ace.Header.AceType, path)
			}
			continue
		}
		if ace.Mask&windowsReadWriteMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if sidIsBlocked(sid) {
			return rejectSID(path, sid)
		}
		if sidInSet(sid, allowed) {
			continue
		}
		return rejectSID(path, sid)
	}
	return nil
}

// rejectSID builds a value-free rejection error that names both the friendly
// principal and its raw SID, and appends the verified icacls remediation
// (DAV-38 A9). It carries only the path, the principal, and a fixed command —
// never a secret value.
func rejectSID(path string, sid *windows.SID) error {
	return fmt.Errorf("%w: %s is accessible by %s (%s); make it owner-only:  icacls %q /inheritance:r /grant:r \"%%USERNAME%%:F\"",
		ErrInsecurePermissions, path, friendlyName(sid), sid.String(), path)
}

// friendlyName resolves a SID to a human-readable "DOMAIN\account" principal
// via LookupAccountSid. On ANY error (unmapped SID, RPC failure, etc.) or an
// empty result it falls back to the raw SID string. It never panics.
func friendlyName(sid *windows.SID) string {
	if sid == nil {
		return "<nil SID>"
	}
	account, domain, _, err := sid.LookupAccount("")
	if err != nil || account == "" {
		return sid.String()
	}
	if domain == "" {
		return account
	}
	return domain + "\\" + account
}

// validateLocalFixedNTFS rejects config/secret files that do not live on a
// local, fixed, NTFS volume. DAV-38 confirmed the NTFS-DACL read cannot see
// SMB share permissions (false-ACCEPT risk) and mis-maps remote/NFS/removable
// identities (false-REJECT). We restrict to DRIVE_FIXED + NTFS + non-UNC so
// the user gets one clear message instead of an opaque SID error.
//
// Everything is handle-based (TOCTOU-safe): the file is already open.
func validateLocalFixedNTFS(handle windows.Handle, path string) error {
	// 1) Filesystem must be NTFS.
	fsName, err := fileSystemName(handle)
	if err != nil {
		return fmt.Errorf("read file system name: %w", err)
	}
	if !strings.EqualFold(fsName, "NTFS") {
		return rejectVolume(path)
	}

	// 2) Recover the canonical path; reject UNC outright.
	finalPath, err := finalPathName(handle)
	if err != nil {
		return fmt.Errorf("resolve final path: %w", err)
	}
	if isUNCPath(finalPath) {
		return rejectVolume(path)
	}

	// 3) The volume must be a fixed local drive.
	root := volumeRootFromFinalPath(finalPath)
	if root == "" {
		return rejectVolume(path)
	}
	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return fmt.Errorf("convert volume root: %w", err)
	}
	if windows.GetDriveType(rootPtr) != windows.DRIVE_FIXED {
		return rejectVolume(path)
	}
	return nil
}

func rejectVolume(path string) error {
	return fmt.Errorf("%w: %s must be on a local fixed NTFS drive (e.g. %%LOCALAPPDATA%%\\zscalerctl); network, removable, and UNC paths can't be securely validated on Windows",
		ErrInsecurePermissions, path)
}

// fileSystemName returns the file-system name (e.g. "NTFS") for the volume that
// backs the open handle, via GetVolumeInformationByHandleW.
func fileSystemName(handle windows.Handle) (string, error) {
	fsNameBuf := make([]uint16, windows.MAX_PATH+1)
	err := windows.GetVolumeInformationByHandle(
		handle,
		nil, // volume name buffer (unused)
		0,
		nil, // serial number (unused)
		nil, // max component length (unused)
		nil, // file system flags (unused)
		&fsNameBuf[0],
		uint32(len(fsNameBuf)),
	)
	if err != nil {
		return "", err
	}
	return windows.UTF16ToString(fsNameBuf), nil
}

// finalPathName returns the normalized DOS path for the open handle, via
// GetFinalPathNameByHandleW. The result looks like `\\?\C:\...` for a local
// file or `\\?\UNC\server\share\...` for a UNC path.
func finalPathName(handle windows.Handle) (string, error) {
	// First call sizes the buffer (returns required length excluding the NUL).
	n, err := windows.GetFinalPathNameByHandle(handle, nil, 0, fileNameNormalizedDOS)
	if err != nil {
		return "", err
	}
	buf := make([]uint16, n+1)
	n, err = windows.GetFinalPathNameByHandle(handle, &buf[0], uint32(len(buf)), fileNameNormalizedDOS)
	if err != nil {
		return "", err
	}
	if int(n) >= len(buf) {
		// Path grew between calls; treat as unverifiable rather than truncating.
		return "", fmt.Errorf("final path length %d exceeds buffer %d", n, len(buf))
	}
	return windows.UTF16ToString(buf[:n]), nil
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
