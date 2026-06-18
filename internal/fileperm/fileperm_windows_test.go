//go:build windows

package fileperm

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsDACLAcceptsOwnerAdminSystemOnly(t *testing.T) {
	t.Parallel()

	for _, sddl := range []string{
		"O:BAD:P(A;;GRGW;;;BA)(A;;GRGW;;;SY)",
	} {
		sddl := sddl
		t.Run(sddl, func(t *testing.T) {
			t.Parallel()
			sd := securityDescriptorFromString(t, sddl)
			if err := validateSecurityDescriptor(sd); err != nil {
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
			if err := validateSecurityDescriptor(sd); !errors.Is(err, ErrInsecurePermissions) {
				t.Fatalf("validateSecurityDescriptor(%q) error = %v, want ErrInsecurePermissions", sddl, err)
			}
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

	if err := validateSecurityDescriptor(sd); !errors.Is(err, ErrInsecurePermissions) {
		t.Fatalf("validateSecurityDescriptor(object ACE) error = %v, want ErrInsecurePermissions", err)
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
