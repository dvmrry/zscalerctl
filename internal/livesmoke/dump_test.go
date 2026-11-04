package livesmoke

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDumpFileSetFailsOnUnreadableDirectory(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads directories regardless of mode")
	}
	dumpDir := t.TempDir()
	locked := filepath.Join(dumpDir, "resources", "zia")
	if err := os.MkdirAll(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "locations.json"), []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	var out, errb bytes.Buffer
	s := &smoke{rep: &reporter{out: &out, errw: &errb}, resources: []string{"zia/locations"}}
	s.validateDumpFileSet(dumpDir)

	if s.rep.failures != 1 || !strings.Contains(errb.String(), "[FAIL] dump resources directory unreadable") {
		t.Fatalf("failures = %d, stderr = %q, want one unreadable-directory failure", s.rep.failures, errb.String())
	}
	if strings.Contains(out.String(), "[PASS]") {
		t.Fatalf("stdout = %q, want no pass", out.String())
	}
}
