//go:build windows && amd64

package keyring

import (
	"testing"
	"unsafe"
)

func TestCredentialWLayout(t *testing.T) {
	var c credentialW
	if off := unsafe.Offsetof(c.CredentialBlob); off != 40 {
		t.Fatalf("credentialW.CredentialBlob offset = %d, want 40", off)
	}
	if sz := unsafe.Sizeof(c); sz != 80 {
		t.Fatalf("unsafe.Sizeof(credentialW) = %d, want 80", sz)
	}
}
