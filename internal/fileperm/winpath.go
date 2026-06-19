package fileperm

import "strings"

// The helpers in this file are pure string logic shared by the Windows
// volume-restriction check (see fileperm_windows.go). They live in an
// untagged file with no Windows imports so they can be unit-tested on any
// host OS — the Windows syscalls that feed them cannot run on a dev box.

// isUNCPath reports whether a Windows path refers to a network (UNC) share.
//
// It accepts both the user-facing form (\\server\share\...) and the
// extended-length / device forms returned by GetFinalPathNameByHandleW
// (\\?\UNC\server\share\... and \\.\UNC\server\share\...). Forward slashes
// are tolerated because callers may normalize separators.
func isUNCPath(p string) bool {
	if p == "" {
		return false
	}
	q := strings.ReplaceAll(p, "/", `\`)

	// Extended-length / device UNC: \\?\UNC\server\share or \\.\UNC\server\share.
	if hasPrefixFold(q, `\\?\UNC\`) || hasPrefixFold(q, `\\.\UNC\`) {
		return true
	}
	// A drive-letter extended path (\\?\C:\...) is local, not UNC.
	if strings.HasPrefix(q, `\\?\`) || strings.HasPrefix(q, `\\.\`) {
		return false
	}
	// Plain UNC: \\server\share\...
	return strings.HasPrefix(q, `\\`)
}

// volumeRootFromFinalPath extracts the drive root (e.g. `C:\`) from a path as
// returned by GetFinalPathNameByHandleW. It returns "" when no drive-letter
// root can be determined (e.g. a UNC path), in which case the caller must not
// treat the volume as a local fixed drive.
//
// Recognized inputs:
//
//	\\?\C:\Users\me\file   -> C:\
//	\\.\C:\Users\me\file   -> C:\
//	C:\Users\me\file       -> C:\
//	\\server\share\file    -> "" (UNC, no drive letter)
func volumeRootFromFinalPath(p string) string {
	if p == "" {
		return ""
	}
	q := strings.ReplaceAll(p, "/", `\`)

	// Strip a \\?\ or \\.\ extended-length prefix, but leave UNC paths alone
	// so they fall through to the "no drive letter" result.
	if strings.HasPrefix(q, `\\?\`) || strings.HasPrefix(q, `\\.\`) {
		if hasPrefixFold(q[4:], `UNC\`) {
			return ""
		}
		q = q[4:]
	}

	// At this point a local volume must look like "X:\..." or "X:".
	if len(q) >= 2 && isDriveLetter(q[0]) && q[1] == ':' {
		return string(q[0]) + `:\`
	}
	return ""
}

func isDriveLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// hasPrefixFold reports whether s begins with prefix, ignoring ASCII case.
// Used for the case-insensitive "UNC" marker in extended-length paths.
func hasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return strings.EqualFold(s[:len(prefix)], prefix)
}
