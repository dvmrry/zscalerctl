//go:build windows

package credentials_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/credentials"
	"golang.org/x/sys/windows"
)

func TestReadOwnerOnlySecretFileWindowsDACL(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte("fake-secret\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	restrictWindowsFileDACL(t, path)

	got, err := credentials.ReadOwnerOnlySecretFile(path)
	if err != nil {
		t.Fatalf("ReadOwnerOnlySecretFile(%q) error = %v, want nil", path, err)
	}
	if got.Reveal() != "fake-secret" {
		t.Errorf("ReadOwnerOnlySecretFile(%q).Reveal() = %q, want %q", path, got.Reveal(), "fake-secret")
	}
}

func restrictWindowsFileDACL(t *testing.T, path string) {
	t.Helper()

	current, err := currentUserSIDForTest()
	if err != nil {
		t.Fatalf("currentUserSIDForTest() error = %v, want nil", err)
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		t.Fatalf("CreateWellKnownSid(SYSTEM) error = %v, want nil", err)
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		t.Fatalf("CreateWellKnownSid(Administrators) error = %v, want nil", err)
	}

	entries := []windows.EXPLICIT_ACCESS{
		grantReadWriteForSID(current, windows.TRUSTEE_IS_USER),
		grantReadWriteForSID(system, windows.TRUSTEE_IS_USER),
		grantReadWriteForSID(admins, windows.TRUSTEE_IS_GROUP),
	}
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		t.Fatalf("ACLFromEntries() error = %v, want nil", err)
	}
	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	); err != nil {
		t.Fatalf("SetNamedSecurityInfo(%q) error = %v, want nil", path, err)
	}
}

func grantReadWriteForSID(sid *windows.SID, trusteeType windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE,
		AccessMode:        windows.GRANT_ACCESS,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  trusteeType,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

func currentUserSIDForTest() (*windows.SID, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return user.User.Sid.Copy()
}
