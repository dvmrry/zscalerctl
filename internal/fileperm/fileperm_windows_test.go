//go:build windows

package fileperm

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

const testPath = `C:\Users\test\config.yaml`

func TestWindowsDACLAcceptsOwnerAdminSystemOnly(t *testing.T) {
	t.Parallel()

	for _, sddl := range []string{
		"O:BAD:P(A;;GRGW;;;BA)(A;;GRGW;;;SY)",
	} {
		sddl := sddl
		t.Run(sddl, func(t *testing.T) {
			t.Parallel()
			sd := securityDescriptorFromString(t, sddl)
			if err := validateSecurityDescriptor(testPath, sd); err != nil {
				t.Fatalf("validateSecurityDescriptor(%q) error = %v, want nil", sddl, err)
			}
		})
	}
}

func TestWindowsDACLRejectsBroadPrincipal(t *testing.T) {
	t.Parallel()

	for _, sddl := range []string{
		"O:BUD:P(A;;GR;;;BA)",
		"O:BAD:P(A;;GR;;;WD)",
		"O:BAD:P(A;;GR;;;BU)",
		"O:BAD:P(A;;GR;;;AU)",
		"O:BAD:P(A;ID;GR;;;BU)",
		"O:BAD:P(A;ID;GRGW;;;S-1-5-21-1-2-3-500)",
	} {
		sddl := sddl
		t.Run(sddl, func(t *testing.T) {
			t.Parallel()
			sd := securityDescriptorFromString(t, sddl)
			err := validateSecurityDescriptor(testPath, sd)
			if !errors.Is(err, ErrInsecurePermissions) {
				t.Fatalf("validateSecurityDescriptor(%q) error = %v, want ErrInsecurePermissions", sddl, err)
			}
			// Every reject must carry the actionable icacls remediation and the
			// file path, and must not leak any value (only path + command).
			assertRemediation(t, sddl, err)
		})
	}
}

func TestWindowsDACLRejectsUnsupportedAllowACE(t *testing.T) {
	t.Parallel()

	sd := securityDescriptorFromString(t, "O:BAD:P(A;;GR;;;BA)")
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("DACL() error = %v, want nil", err)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatalf("GetAce() error = %v, want nil", err)
	}
	ace.Header.AceType = accessAllowedObjectACEType

	verr := validateSecurityDescriptor(testPath, sd)
	if !errors.Is(verr, ErrInsecurePermissions) {
		t.Fatalf("validateSecurityDescriptor(object ACE) error = %v, want ErrInsecurePermissions", verr)
	}
	assertRemediation(t, "unsupported ACE", verr)
}

// TestFriendlyNameFallsBackToSIDString verifies friendlyName never panics and
// degrades to the raw SID string for an unmapped (synthetic) SID.
func TestFriendlyNameFallsBackToSIDString(t *testing.T) {
	t.Parallel()

	sid, err := windows.StringToSid("S-1-5-21-1-2-3-1234")
	if err != nil {
		t.Fatalf("StringToSid() error = %v, want nil", err)
	}
	got := friendlyName(sid)
	if got == "" {
		t.Fatalf("friendlyName() = empty, want SID string or principal name")
	}
}

// assertRemediation checks that a reject message names the path, includes the
// icacls remediation, and embeds the %USERNAME%:F grant.
func assertRemediation(t *testing.T, label string, err error) {
	t.Helper()
	msg := err.Error()
	for _, want := range []string{testPath, "icacls", "/inheritance:r", "%USERNAME%:F"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("reject %q message %q missing %q", label, msg, want)
		}
	}
}

func securityDescriptorFromString(t *testing.T, sddl string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatalf("SecurityDescriptorFromString(%q) error = %v, want nil", sddl, err)
	}
	return sd
}
